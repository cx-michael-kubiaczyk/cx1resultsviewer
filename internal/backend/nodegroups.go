package backend

import (
	"fmt"
	"sort"

	"github.com/cxpsemea/Cx1ClientGo"
)

// mergeLineGap is the max line-number gap between a node and a group's
// current [MinLine, MaxLine] window for that node to be merged into the
// same box instead of starting a new one.
const mergeLineGap = 10

// contextPaddingLines is the extra context shown above/below a group's
// highlighted range.
const contextPaddingLines = 2

// maxBoxDisplayLines clamps how tall a merged box gets before it becomes
// internally scrollable instead of growing further.
const maxBoxDisplayLines = 20

type nodeRef struct {
	NodeIndex int
	Node      Cx1ClientGo.ScanSASTResultNodes
}

type nodeGroup struct {
	ResultIndex int
	FilePath    string
	MinLine     uint64
	MaxLine     uint64
	Nodes       []nodeRef
}

// groupResultNodes walks a single result's flow nodes in order and greedily
// merges consecutive nodes that share a file and fall within mergeLineGap
// lines of the group's running range. Groups never span across results.
func groupResultNodes(resultIndex int, result Cx1ClientGo.ScanSASTResult) []nodeGroup {
	var groups []nodeGroup
	var cur *nodeGroup

	for ni, node := range result.Data.Nodes {
		if cur != nil && cur.FilePath == node.FileName &&
			node.Line+mergeLineGap >= cur.MinLine && node.Line <= cur.MaxLine+mergeLineGap {
			if node.Line < cur.MinLine {
				cur.MinLine = node.Line
			}
			if node.Line > cur.MaxLine {
				cur.MaxLine = node.Line
			}
			cur.Nodes = append(cur.Nodes, nodeRef{NodeIndex: ni, Node: node})
			continue
		}

		if cur != nil {
			groups = append(groups, *cur)
		}
		cur = &nodeGroup{
			ResultIndex: resultIndex,
			FilePath:    node.FileName,
			MinLine:     node.Line,
			MaxLine:     node.Line,
			Nodes:       []nodeRef{{NodeIndex: ni, Node: node}},
		}
	}
	if cur != nil {
		groups = append(groups, *cur)
	}

	return groups
}

// nodeMatchKey identifies a dataflow node by file+line+column+name, so that
// the same source position reached by different results can be recognized
// as "the same node" regardless of which result it came from.
func nodeMatchKey(node Cx1ClientGo.ScanSASTResultNodes) string {
	return fmt.Sprintf("%s\x00%d\x00%d\x00%s", node.FileName, node.Line, node.Column, node.Name)
}

// nodeMatchIDs holds the two identifier lists a matching popup can show for
// a node, each independently de-duplicated and sorted.
type nodeMatchIDs struct {
	ResultIDs     []string
	SimilarityIDs []string
}

// buildNodeResultIndex maps each node key to the sorted, de-duplicated lists
// of ResultIDs and SimilarityIDs (from allResults) whose dataflow passes
// through that node.
func buildNodeResultIndex(allResults []Cx1ClientGo.ScanSASTResult) map[string]nodeMatchIDs {
	resultSets := make(map[string]map[string]struct{})
	simSets := make(map[string]map[string]struct{})
	for _, result := range allResults {
		for _, node := range result.Data.Nodes {
			key := nodeMatchKey(node)

			if resultSets[key] == nil {
				resultSets[key] = make(map[string]struct{})
			}
			resultSets[key][result.ResultID] = struct{}{}

			if simSets[key] == nil {
				simSets[key] = make(map[string]struct{})
			}
			simSets[key][result.SimilarityID] = struct{}{}
		}
	}

	index := make(map[string]nodeMatchIDs, len(resultSets))
	for key, set := range resultSets {
		index[key] = nodeMatchIDs{
			ResultIDs:     sortedKeys(set),
			SimilarityIDs: sortedKeys(simSets[key]),
		}
	}
	return index
}

func sortedKeys(set map[string]struct{}) []string {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// paddedRange returns the group's line range widened by contextPaddingLines,
// clamped to a minimum line of 1.
func (g *nodeGroup) paddedRange() (min, max uint64) {
	if g.MinLine > contextPaddingLines {
		min = g.MinLine - contextPaddingLines
	} else {
		min = 1
	}
	max = g.MaxLine + contextPaddingLines
	return
}
