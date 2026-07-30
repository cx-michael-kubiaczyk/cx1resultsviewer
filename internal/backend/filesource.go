package backend

import (
	"strings"
)

type CodeSet struct {
	Files map[string]string
}

func NewCodeSet() CodeSet {
	return CodeSet{
		Files: make(map[string]string),
	}
}

func (cs *CodeSet) AddFile(path, code string) {
	cs.Files[path] = code
}

func (cs *CodeSet) GetSources() string {
	var str strings.Builder
	for _, code := range cs.Files {
		str.WriteString(code)
		str.WriteString("\n")
	}
	return str.String()
}

func (cs *CodeSet) HasFile(path string) bool {
	_, ok := cs.Files[path]
	return ok
}
