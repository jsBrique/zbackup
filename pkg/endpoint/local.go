package endpoint

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// LocalFS 实现 FileSystem 接口，用于本地文件系统
type LocalFS struct {
	root string
}

// NewLocalFS 创建一个 LocalFS
func NewLocalFS(root string) *LocalFS {
	return &LocalFS{root: root}
}

func (l *LocalFS) Root() string {
	return l.root
}

func (l *LocalFS) List(excludes []string) ([]FileMeta, error) {
	var metas []FileMeta
	err := filepath.WalkDir(l.root, func(fullPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.root, fullPath)
		if err != nil {
			return err
		}
		if shouldExclude(rel, excludes) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		metas = append(metas, FileMeta{
			RelPath: rel,
			Size:    info.Size(),
			Mode:    uint32(info.Mode()),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return metas, nil
}

func (l *LocalFS) Open(relPath string) (io.ReadCloser, error) {
	full := filepath.Join(l.root, relPath)
	return os.Open(full)
}

func (l *LocalFS) Create(relPath string, perm fs.FileMode) (io.WriteCloser, error) {
	full := filepath.Join(l.root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
}

func (l *LocalFS) MkdirAll(relPath string) error {
	full := filepath.Join(l.root, relPath)
	return os.MkdirAll(full, 0o755)
}

func (l *LocalFS) Remove(relPath string) error {
	full := filepath.Join(l.root, relPath)
	return os.RemoveAll(full)
}

func (l *LocalFS) Stat(relPath string) (FileMeta, error) {
	full := filepath.Join(l.root, relPath)
	info, err := os.Stat(full)
	if err != nil {
		return FileMeta{}, fmt.Errorf("stat %s: %w", relPath, err)
	}
	return FileMeta{
		RelPath: relPath,
		Size:    info.Size(),
		Mode:    uint32(info.Mode()),
		ModTime: info.ModTime(),
	}, nil
}
