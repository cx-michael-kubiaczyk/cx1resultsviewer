package backend

import (
	"fmt"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

type WebServer struct {
	Cx1Client   *Cx1ClientGo.Cx1Client
	logger      *logrus.Logger
	ScanSources CodeSet
}

func NewServer(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger) WebServer {
	return WebServer{
		Cx1Client:   cx1client,
		logger:      logger,
		ScanSources: NewCodeSet(),
	}
}

func (m *WebServer) LoadResult(path string) error {
	_, sid, rid, err := extractIDFromURL(path)
	if err != nil {
		return err
	}

	err = m.createCodeExtract(sid, rid)
	if err != nil {
		return fmt.Errorf("failed to create code extract: %v", err)
	}

	return nil
}

func (m *WebServer) Shutdown() {
}

func (m *WebServer) Run() error {
	return m.test()
}

func (m *WebServer) test() error {
	err := m.LoadResult(`https://deu.ast.checkmarx.net/sast-results/e25a6a86-2d86-4b1b-8d50-6c6f706decdd/0f562295-d7a8-49d6-bd37-82177647633b?resultId=wta7MY4iw%2BJ3rxS9fiHBXIHukys%3D&pagination=pageSize%3D10%3BcurrentPage%3D1&grouping=groups%255B0%255D%3Dlanguage%3Bgroups%255B1%255D%3Dseverity%3Bgroups%255B2%255D%3DqueryName`)
	if err != nil {
		return err
	}

	fmt.Println(m.ScanSources.GetSources())
	return nil
}
