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
	if err := os.WriteFile(filepath.Join(srcDir, "file.txt"), content, 0o644); err != nil {
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
		RelPath: "file.txt",
		Meta: endpoint.FileMeta{
			RelPath: "file.txt",
			Size:    int64(len(content)),
			ModTime: time.Now(),
		},
		Action: ActionUpload,
	})
	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if _, ok := result.Success["file.txt"]; !ok {
		t.Fatalf("file not marked success")
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "file.txt"))
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
