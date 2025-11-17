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
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755); err != nil {
		t.Fatalf("make dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "file.txt"), content, 0o644); err != nil {
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
		RelPath: "sub",
		Meta: endpoint.FileMeta{
			RelPath: "sub",
			IsDir:   true,
		},
		Action: ActionMkdir,
	})
	plan.AddItem(TransferItem{
		RelPath: "sub/file.txt",
		Meta: endpoint.FileMeta{
			RelPath: "sub/file.txt",
			Size:    int64(len(content)),
			ModTime: time.Now(),
		},
		Action: ActionUpload,
	})
	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if _, ok := result.Success["sub/file.txt"]; !ok {
		t.Fatalf("file not marked success")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "sub")); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "sub", "file.txt"))
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
