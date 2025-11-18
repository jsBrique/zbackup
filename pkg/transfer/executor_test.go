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
		Checksum: endpoint.ChecksumNone,
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

func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
