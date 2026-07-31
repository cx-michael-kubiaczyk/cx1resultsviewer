package backend

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultCacheDir = "./cache"

type CodeSet struct {
	Files    map[string]string
	CacheDir string
}

func NewCodeSet() CodeSet {
	return CodeSet{
		Files:    make(map[string]string),
		CacheDir: defaultCacheDir,
	}
}

func (cs *CodeSet) cachePath(sid, path string) string {
	return filepath.Join(cs.CacheDir, sid, path)
}

func (cs *CodeSet) AddFile(sid, path, code string) {
	cs.Files[path] = code

	cachePath := cs.cachePath(sid, path)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err == nil {
		_ = os.WriteFile(cachePath, []byte(code), 0644)
	}
}

func (cs *CodeSet) GetFile(path string) string {
	if src, ok := cs.Files[path]; ok {
		return src
	}
	return ""
}

func (cs *CodeSet) GetSources() string {
	var str strings.Builder
	for _, code := range cs.Files {
		str.WriteString(code)
		str.WriteString("\n")
	}
	return str.String()
}

func (cs *CodeSet) HasFile(sid, path string) bool {
	if _, ok := cs.Files[path]; ok {
		return true
	}

	data, err := os.ReadFile(cs.cachePath(sid, path))
	if err != nil {
		return false
	}
	cs.Files[path] = string(data)
	return true
}
