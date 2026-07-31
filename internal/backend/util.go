package backend

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
)

func extractIDFromURL(path string) (ProjectID, ScanID, ResultID string, err error) {
	u, err := url.Parse(path)
	if err != nil {
		err = fmt.Errorf("failed to parse URL: %w", err)
		return
	}

	// Extract IDs from the path: /sast-results/{projectID}/{scanID}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 3 {
		err = fmt.Errorf("invalid URL structure: projectID or scanID missing in path")
		return
	}

	ProjectID = segments[1]
	ScanID = segments[2]
	ResultID = u.Query().Get("resultId")
	return
}

func (m *WebServer) createCodeExtract(sid, rid string) ([]Cx1ClientGo.ScanSASTResult, error) {
	filter := Cx1ClientGo.ScanSASTResultsFilter{
		BaseFilter: Cx1ClientGo.BaseFilter{Limit: 10},
		ScanID:     sid,
		ResultIDs:  []string{rid},
	}

	_, results, err := m.Cx1Client.GetAllScanSASTResultsFiltered(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %v", err)
	}

	for _, result := range results {
		m.logger.Debugf("Result: %+v", result)
		for i, n := range result.Data.Nodes {
			m.logger.Debugf("Node %d: %+v", i, n)
			if !m.ScanSources.HasFile(sid, n.FileName) {
				fileSource, err := m.Cx1Client.GetScannedFileSourceByID(sid, n.FileName)
				if err != nil {
					return nil, fmt.Errorf("failed to get file source: %v", err)
				}
				m.ScanSources.AddFile(sid, n.FileName, fileSource)
			}
		}
	}
	return results, nil

}
