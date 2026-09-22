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

func (m *WebServer) getAllFindings(scanId string, queryId uint64) ([]Cx1ClientGo.ScanSASTResult, error) {
	filter := Cx1ClientGo.ScanSASTResultsFilter{
		BaseFilter: Cx1ClientGo.BaseFilter{Limit: 100},
		ScanID:     scanId,
		QueryIDs:   []uint64{queryId},
	}

	_, results, err := m.Cx1Client.GetAllScanSASTResultsFiltered(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %v", err)
	}

	return results, nil
}

// escapeFilePathForURL percent-encodes each segment of a file path so it can
// be safely embedded in a request URL. Characters such as '#' or '?' are
// otherwise interpreted as URL syntax (fragment/query delimiters) rather than
// literal path characters, truncating or corrupting the request when a
// scanned file lives under a folder like "C#".
func escapeFilePathForURL(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

func (m *WebServer) createCodeExtract(sid, rid string) ([]Cx1ClientGo.ScanSASTResult, error) {
	filter := Cx1ClientGo.ScanSASTResultsFilter{
		BaseFilter: Cx1ClientGo.BaseFilter{Limit: 1},
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
			if !m.ScanSources.HasFile(sid, n.FileName) {
				m.logger.Debugf("Node %d: in new file %s:%d,%d '%s'", i, n.FileName, n.Line, n.Column, n.Name)
				fileSource, err := m.Cx1Client.GetScannedFileSourceByID(sid, escapeFilePathForURL(n.FileName))
				if err != nil {
					return nil, fmt.Errorf("failed to get file source: %v", err)
				}
				m.ScanSources.AddFile(sid, n.FileName, fileSource)
			} else {
				m.logger.Debugf("Node %d: in cached file %s:%d,%d '%s'", i, n.FileName, n.Line, n.Column, n.Name)
			}
		}
	}
	return results, nil
}
