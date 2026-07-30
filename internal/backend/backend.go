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
	Results     []Cx1ClientGo.ScanSASTResult

	Addr   string
	WebDir string

	mu      sync.RWMutex
	lastURL string
	loadErr error
}

func NewServer(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger) WebServer {
	return WebServer{
		Cx1Client:   cx1client,
		logger:      logger,
		ScanSources: NewCodeSet(),
		Addr:        "127.0.0.1:8080",
		WebDir:      "web",
	}
}

func (m *WebServer) LoadResults(path string) error {
	_, sid, rid, err := extractIDFromURL(path)
	if err != nil {
		return err
	}

	m.Results, err = m.createCodeExtract(sid, rid)
	if err != nil {
		return fmt.Errorf("failed to create code extract: %v", err)
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

	for _, result := range m.Results {
		fmt.Printf("Result: %s\n", result.ResultID)
		for i, node := range result.Data.Nodes {
			fmt.Printf("Node %d: %s line %d col %d length %d - '%s'\n", i+1, node.FileName, node.Line, node.Column, node.Length, node.Name)
			fmt.Println(m.ScanSources.GetFile(node.FileName))
		}
	}
	return nil
}
