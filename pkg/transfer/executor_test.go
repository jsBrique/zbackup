package transfer

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/ui"
)

func TestExecutorExecute(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	content := []byte("hello world")
	dirName := "子 目录"
	fileName := "数 据.txt"
	if err := os.MkdirAll(filepath.Join(srcDir, dirName), 0o755); err != nil {
		t.Fatalf("make dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, dirName, fileName), content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	srcFS := endpoint.NewLocalFS(srcDir)
	dstFS := endpoint.NewLocalFS(dstDir)
	exec := Executor{
		SourceFS: srcFS,
		DestFS:   dstFS,
		Src:      endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: srcDir},
		Dst:      endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: dstDir},
		Checksum: endpoint.ChecksumSHA256,
		Logger:   slogDiscard(),
		Progress: ui.NoopProgress{},
	}
	plan := Plan{}
	plan.AddItem(TransferItem{
		RelPath: filepath.ToSlash(dirName),
		Meta: endpoint.FileMeta{
			RelPath: filepath.ToSlash(dirName),
			IsDir:   true,
		},
		Action: ActionMkdir,
	})
	plan.AddItem(TransferItem{
		RelPath: filepath.ToSlash(filepath.Join(dirName, fileName)),
		Meta: endpoint.FileMeta{
			RelPath: filepath.ToSlash(filepath.Join(dirName, fileName)),
			Size:    int64(len(content)),
			ModTime: time.Now(),
		},
		Action: ActionUpload,
	})
	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	targetFile := filepath.ToSlash(filepath.Join(dirName, fileName))
	if _, ok := result.Success[targetFile]; !ok {
		t.Fatalf("file not marked success")
	}
	if _, err := os.Stat(filepath.Join(dstDir, dirName)); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, dirName, fileName))
	if err != nil {
		t.Fatalf("read dest failed: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestExecutorRemoteHash(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	content := []byte("remote hash data")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file.txt"), content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	srcFS := mockRemoteFS{
		LocalFS: endpoint.NewLocalFS(srcDir),
		Algo:    endpoint.ChecksumSHA256,
	}
	dstFS := endpoint.NewLocalFS(dstDir)
	exec := Executor{
		SourceFS: srcFS,
		DestFS:   dstFS,
		Src:      endpoint.Endpoint{Type: endpoint.EndpointRemote, Path: srcDir},
		Dst:      endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: dstDir},
		Checksum: endpoint.ChecksumSHA256,
		Logger:   slogDiscard(),
		Progress: ui.NoopProgress{},
	}
	plan := Plan{}
	plan.AddItem(TransferItem{
		RelPath: "file.txt",
		Meta: endpoint.FileMeta{
			RelPath: "file.txt",
			Size:    int64(len(content)),
			ModTime: time.Now(),
		},
		Action: ActionDownload,
	})
	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	meta, ok := result.Success["file.txt"]
	if !ok {
		t.Fatalf("file not marked success")
	}
	if meta.Checksum == "" {
		t.Fatalf("checksum missing, remote hash not used")
	}
}

func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockRemoteFS struct {
	*endpoint.LocalFS
	Algo endpoint.ChecksumAlgo
}

func (m mockRemoteFS) ComputeRemoteHash(relPath string, algo endpoint.ChecksumAlgo) ([]byte, error) {
	if algo != m.Algo {
		return nil, endpoint.ErrHashCommandUnavailable
	}
	reader, err := m.LocalFS.Open(relPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	h := newHash(algo)
	if h == nil {
		return nil, endpoint.ErrHashCommandUnavailable
	}
	if _, err := io.Copy(h, reader); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
