package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"zbackup/pkg/endpoint"
)

const (
	metaDir       = ".zbackup"
	snapshotDir   = "snapshots"
	latestSymlink = "latest"
)

// Snapshot 描述一次备份的结果
type Snapshot struct {
	Name       string                       `json:"name"`
	CreatedAt  time.Time                    `json:"created_at"`
	SourceRoot string                       `json:"source_root"`
	DestRoot   string                       `json:"dest_root"`
	Files      map[string]endpoint.FileMeta `json:"files"`
	Completed  bool                         `json:"completed"`
}

// Store 负责在目标端存取快照
type Store struct {
	fs endpoint.FileSystem
}

// NewStore 创建 Store
func NewStore(fs endpoint.FileSystem) *Store {
	return &Store{fs: fs}
}

// LoadLatest 读取 latest 指向的快照
func (s *Store) LoadLatest() (*Snapshot, error) {
	data, err := s.readFile(filepath.Join(metaDir, latestSymlink))
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return nil, fmt.Errorf("latest 为空")
	}
	return s.Load(name)
}

// Load 按名称读取快照
func (s *Store) Load(name string) (*Snapshot, error) {
	path := filepath.Join(metaDir, snapshotDir, fmt.Sprintf("%s.json", name))
	data, err := s.readFile(path)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Save 将快照写入存储，并更新 latest
func (s *Store) Save(snap Snapshot) error {
	if snap.Files == nil {
		snap.Files = make(map[string]endpoint.FileMeta)
	}
	snap.CreatedAt = snap.CreatedAt.UTC()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(metaDir, snapshotDir)
	if err := s.fs.MkdirAll(dir); err != nil {
		return err
	}
	writer, err := s.fs.Create(filepath.Join(dir, fmt.Sprintf("%s.json", snap.Name)), 0o644)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := s.writeFile(filepath.Join(metaDir, latestSymlink), []byte(snap.Name+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func (s *Store) readFile(rel string) ([]byte, error) {
	reader, err := s.fs.Open(rel)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (s *Store) writeFile(rel string, data []byte, perm fs.FileMode) error {
	if err := s.fs.MkdirAll(filepath.Dir(rel)); err != nil {
		return err
	}
	writer, err := s.fs.Create(rel, perm)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return strings.Contains(err.Error(), "No such file") || strings.Contains(err.Error(), "not found")
}
