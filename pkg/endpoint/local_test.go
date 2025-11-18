package endpoint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFSListSpecialNames(t *testing.T) {
	tmp := t.TempDir()
	rootDir := filepath.Join(tmp, "子 目录")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(rootDir, "data 中 文.txt")
	if err := os.WriteFile(filePath, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	fs := NewLocalFS(tmp)
	metas, err := fs.List(nil)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	foundDir := false
	foundFile := false
	for _, m := range metas {
		if m.RelPath == filepath.ToSlash("子 目录") && m.IsDir {
			foundDir = true
		}
		if m.RelPath == filepath.ToSlash("子 目录/data 中 文.txt") && !m.IsDir {
			foundFile = true
		}
	}
	if !foundDir || !foundFile {
		t.Fatalf("missing entries dir=%v file=%v metas=%+v", foundDir, foundFile, metas)
	}
}
