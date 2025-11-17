package endpoint

import (
	"path/filepath"
	"strings"
)

// shouldExclude 根据 glob 模式决定是否跳过
func shouldExclude(rel string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	norm := filepath.ToSlash(rel)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if matched, _ := filepath.Match(p, norm); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(norm)); matched {
			return true
		}
	}
	return false
}
