package backend

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/cxpsemea/Cx1ClientGo"
)

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

	vm, err := buildPageViewModel(m.lastURL, m.loadErr, m.Results, &m.ScanSources)
	if err != nil {
		http.Error(w, "failed to prepare page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmplPath := filepath.Join(m.WebDir, "templates", "index.html.tmpl")
	tmpl, err := template.ParseFiles(tmplPath)
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

type CodeBoxViewModel struct {
	ResultIndex int
	NodeIndex   int
	FilePath    string
	Line        uint64
	Column      uint64
	Length      uint64
	Name        string
	QueryName   string
	DataJSON    template.JS
}

type PageViewModel struct {
	URL          string
	HasError     bool
	ErrorMessage string
	HasResults   bool
	CodeBoxes    []CodeBoxViewModel
}

type codeBoxPayload struct {
	FilePath string `json:"filePath"`
	Source   string `json:"source"`
	Line     uint64 `json:"line"`
	Column   uint64 `json:"column"`
	Length   uint64 `json:"length"`
}

func buildPageViewModel(url string, loadErr error, results []Cx1ClientGo.ScanSASTResult, sources *CodeSet) (PageViewModel, error) {
	vm := PageViewModel{URL: url}
	if loadErr != nil {
		vm.HasError = true
		vm.ErrorMessage = loadErr.Error()
	}

	for ri, result := range results {
		for ni, node := range result.Data.Nodes {
			payload := codeBoxPayload{
				FilePath: node.FileName,
				Source:   sources.GetFile(node.FileName),
				Line:     node.Line,
				Column:   node.Column,
				Length:   node.Length,
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return vm, err
			}
			vm.CodeBoxes = append(vm.CodeBoxes, CodeBoxViewModel{
				ResultIndex: ri,
				NodeIndex:   ni,
				FilePath:    node.FileName,
				Line:        node.Line,
				Column:      node.Column,
				Length:      node.Length,
				Name:        node.Name,
				QueryName:   result.Data.QueryName,
				DataJSON:    template.JS(raw),
			})
		}
	}
	vm.HasResults = len(vm.CodeBoxes) > 0
	return vm, nil
}
