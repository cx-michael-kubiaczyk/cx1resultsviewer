package backend

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

type WebServer struct {
	Cx1Client   *Cx1ClientGo.Cx1Client
	logger      *logrus.Logger
	ScanSources CodeSet
	Result      Cx1ClientGo.ScanSASTResult
	AllResults  []Cx1ClientGo.ScanSASTResult
	Triages     []Cx1ClientGo.SASTResultsPredicates

	Addr   string
	WebDir string

	mu        sync.RWMutex
	lastURL   string
	scanID    string
	projectID string
	loadErr   error
}

func NewServer(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger, address string) WebServer {
	return WebServer{
		Cx1Client:   cx1client,
		logger:      logger,
		ScanSources: NewCodeSet(),
		Addr:        address,
		WebDir:      "web",
	}
}

func (m *WebServer) LoadResults(path string) error {
	pid, sid, rid, err := extractIDFromURL(path)
	if err != nil {
		return err
	}

	m.projectID = pid
	m.scanID = sid

	results, err := m.createCodeExtract(sid, rid)
	if err != nil {
		return fmt.Errorf("failed to create code extract: %v", err)
	}

	if len(results) >= 1 {
		m.Result = results[0]
		m.Triages, err = m.Cx1Client.GetSASTResultsPredicatesByID(m.Result.SimilarityID, pid, sid)
		if err != nil {
			return fmt.Errorf("failed to get predicates for similarity ID %s: %v", m.Result.SimilarityID, err)
		}

		m.AllResults, err = m.getAllFindings(sid, m.Result.Data.QueryID)
		if err != nil {
			return fmt.Errorf("failed to get all findings with same queryId %d", m.Result.Data.QueryID)
		}
	}

	return nil
}

func (m *WebServer) Shutdown() {
}

func (m *WebServer) Run() error {
	m.logger.Infof("Starting HTTP server on %s (serving assets from %q)", m.Addr, m.WebDir)
	return http.ListenAndServe(m.Addr, m.routes())
}

func (m *WebServer) test() error {
	err := m.LoadResults(`https://deu.ast.checkmarx.net/sast-results/e25a6a86-2d86-4b1b-8d50-6c6f706decdd/0f562295-d7a8-49d6-bd37-82177647633b?resultId=wta7MY4iw%2BJ3rxS9fiHBXIHukys%3D&pagination=pageSize%3D10%3BcurrentPage%3D1&grouping=groups%255B0%255D%3Dlanguage%3Bgroups%255B1%255D%3Dseverity%3Bgroups%255B2%255D%3DqueryName`)
	if err != nil {
		return err
	}

	fmt.Printf("Result: %s\n", m.Result.ResultID)
	for i, node := range m.Result.Data.Nodes {
		fmt.Printf("Node %d: %s line %d col %d length %d - '%s'\n", i+1, node.FileName, node.Line, node.Column, node.Length, node.Name)
		fmt.Println(m.ScanSources.GetFile("0f562295-d7a8-49d6-bd37-82177647633b", node.FileName))
	}

	return nil
}
