package backend

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
)

const nodeMatchIDSep = ","

var templateFuncs = template.FuncMap{
	"trimLeadingSlash": func(s string) string {
		return strings.TrimPrefix(s, "/")
	},
}

func (m *WebServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleIndex)
	mux.HandleFunc("/load", m.handleLoad)
	mux.Handle("/web/static/", http.StripPrefix("/web/static/",
		http.FileServer(http.Dir(filepath.Join(m.WebDir, "static")))))
	return mux
}

func (m *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	m.logger.Infof("Handling url: %s", m.lastURL)
	vm, err := buildPageViewModel(m.lastURL, m.scanID, m.loadErr, m.Result, m.AllResults, m.Triages, &m.ScanSources)
	if err != nil {
		m.logger.Errorf("Failed to prepare page: %s", err)
		http.Error(w, "failed to prepare page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmplPath := filepath.Join(m.WebDir, "templates", "index.html.tmpl")
	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(templateFuncs).ParseFiles(tmplPath)
	if err != nil {
		m.logger.Errorf("failed to parse template %s: %v", tmplPath, err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, vm); err != nil {
		m.logger.Errorf("failed to execute template: %v", err)
	}
}

func (m *WebServer) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	url := r.FormValue("url")

	m.mu.Lock()
	m.lastURL = url
	m.loadErr = m.LoadResults(url)
	if m.loadErr != nil {
		m.logger.Errorf("LoadResults failed: %v", m.loadErr)
	}
	m.mu.Unlock()

	http.Redirect(w, r, "/", http.StatusFound)
}

type HighlightViewModel struct {
	FlowID       string
	Line         uint64
	Column       uint64
	Length       uint64
	Name         string
	MatchCount   int
	MatchIDsAttr string
}

type CodeBoxViewModel struct {
	BoxID       string
	ResultIndex int
	FilePath    string
	QueryName   string
	MinLine     uint64
	MaxLine     uint64
	Highlights  []HighlightViewModel
	DataJSON    template.JS
}

// FileGroupViewModel collects consecutive CodeBoxes that share a FilePath so
// the template can render them as a single visual block with one shared
// filename heading.
type FileGroupViewModel struct {
	FilePath string
	Boxes    []CodeBoxViewModel
}

type PageViewModel struct {
	URL             string
	HasError        bool
	ErrorMessage    string
	HasResults      bool
	Finding         Cx1ClientGo.ScanSASTResult
	AllResultsCount int
	Triages         []Cx1ClientGo.SASTResultsPredicates
	FileGroups      []FileGroupViewModel
}

type highlightPayload struct {
	FlowID    string `json:"flowId"`
	NodeIndex int    `json:"nodeIndex"`
	Line      uint64 `json:"line"`
	Column    uint64 `json:"column"`
	Length    uint64 `json:"length"`
	Name      string `json:"name"`
}

type codeBoxPayload struct {
	BoxID       string             `json:"boxId"`
	FilePath    string             `json:"filePath"`
	Source      string             `json:"source"`
	ResultIndex int                `json:"resultIndex"`
	MinLine     uint64             `json:"minLine"`
	MaxLine     uint64             `json:"maxLine"`
	Highlights  []highlightPayload `json:"highlights"`
}

func buildPageViewModel(url, sid string, loadErr error, result Cx1ClientGo.ScanSASTResult, allResults []Cx1ClientGo.ScanSASTResult, triages []Cx1ClientGo.SASTResultsPredicates, sources *CodeSet) (PageViewModel, error) {
	vm := PageViewModel{
		URL:             url,
		Finding:         result,
		AllResultsCount: len(allResults),
		Triages:         triages,
	}
	if loadErr != nil {
		vm.HasError = true
		vm.ErrorMessage = loadErr.Error()
	}

	nodeResultIndex := buildNodeResultIndex(allResults)

	var boxes []CodeBoxViewModel
	for gi, group := range groupResultNodes(0, result) {
		boxID := fmt.Sprintf("box-%d-%d", 0, gi)
		minLine, maxLine := group.paddedRange()

		highlights := make([]HighlightViewModel, 0, len(group.Nodes))
		highlightPayloads := make([]highlightPayload, 0, len(group.Nodes))
		for _, ref := range group.Nodes {
			flowID := fmt.Sprintf("f%d-%d", 0, ref.NodeIndex)
			matchIDs := nodeResultIndex[nodeMatchKey(ref.Node)]
			highlights = append(highlights, HighlightViewModel{
				FlowID:       flowID,
				Line:         ref.Node.Line,
				Column:       ref.Node.Column,
				Length:       ref.Node.Length,
				Name:         ref.Node.Name,
				MatchCount:   len(matchIDs),
				MatchIDsAttr: strings.Join(matchIDs, nodeMatchIDSep),
			})
			highlightPayloads = append(highlightPayloads, highlightPayload{
				FlowID:    flowID,
				NodeIndex: ref.NodeIndex,
				Line:      ref.Node.Line,
				Column:    ref.Node.Column,
				Length:    ref.Node.Length,
				Name:      ref.Node.Name,
			})
		}

		payload := codeBoxPayload{
			BoxID:       boxID,
			FilePath:    group.FilePath,
			Source:      sources.GetFile(sid, group.FilePath),
			ResultIndex: 0,
			MinLine:     minLine,
			MaxLine:     maxLine,
			Highlights:  highlightPayloads,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return vm, err
		}
		boxes = append(boxes, CodeBoxViewModel{
			BoxID:       boxID,
			ResultIndex: 0,
			FilePath:    group.FilePath,
			QueryName:   result.Data.QueryName,
			MinLine:     minLine,
			MaxLine:     maxLine,
			Highlights:  highlights,
			DataJSON:    template.JS(raw),
		})
	}

	for _, box := range boxes {
		if n := len(vm.FileGroups); n > 0 && vm.FileGroups[n-1].FilePath == box.FilePath {
			vm.FileGroups[n-1].Boxes = append(vm.FileGroups[n-1].Boxes, box)
		} else {
			vm.FileGroups = append(vm.FileGroups, FileGroupViewModel{FilePath: box.FilePath, Boxes: []CodeBoxViewModel{box}})
		}
	}

	vm.HasResults = len(boxes) > 0
	return vm, nil
}
