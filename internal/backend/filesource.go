package backend

import (
	"os"
	"path/filepath"
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
	cachePath := cs.cachePath(sid, path)
	cs.Files[cachePath] = code
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err == nil {
		_ = os.WriteFile(cachePath, []byte(code), 0644)
	}
}

func (cs *CodeSet) GetFile(sid, path string) string {
	if src, ok := cs.Files[cs.cachePath(sid, path)]; ok {
		return src
	}
	return ""
}

func (cs *CodeSet) HasFile(sid, path string) bool {

	cachePath := cs.cachePath(sid, path)

	if _, ok := cs.Files[cachePath]; ok {
		return true
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return false
	}
	cs.Files[cachePath] = string(data)
	return true
}
