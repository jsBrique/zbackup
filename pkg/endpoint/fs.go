package endpoint

import (
	"errors"
	"io"
	"io/fs"
)

// FileSystem 抽象化的端点文件系统能力
type FileSystem interface {
	Root() string
	List(excludes []string) ([]FileMeta, error)
	Open(relPath string) (io.ReadCloser, error)
	Create(relPath string, perm fs.FileMode) (io.WriteCloser, error)
	MkdirAll(relPath string) error
	Remove(relPath string) error
	Stat(relPath string) (FileMeta, error)
	Close() error
}

// ErrNotImplemented 用于表示某些操作尚未支持
var ErrNotImplemented = errors.New("not implemented")
