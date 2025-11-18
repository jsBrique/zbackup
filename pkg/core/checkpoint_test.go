package core

import (
	"testing"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
)

func TestCheckpointRecordsAndFlushes(t *testing.T) {
	fs := endpoint.NewLocalFS(t.TempDir())
	store := meta.NewStore(fs)
	cp := newCheckpoint(store, nil, "snap", endpoint.Endpoint{Path: "/src"}, endpoint.Endpoint{Path: "/dst"})
	meta := endpoint.FileMeta{RelPath: "dir/file.txt", Size: 10}
	if err := cp.Record(meta); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if err := cp.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	pending, err := store.LoadPending()
	if err != nil {
		t.Fatalf("load pending failed: %v", err)
	}
	if pending == nil || pending.Files["dir/file.txt"].RelPath != "dir/file.txt" {
		t.Fatalf("pending snapshot missing entry: %+v", pending)
	}
}
