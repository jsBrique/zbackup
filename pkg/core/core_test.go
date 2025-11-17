package core

import (
	"testing"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
	"zbackup/pkg/transfer"
)

func TestMergeSnapshot(t *testing.T) {
	last := &meta.Snapshot{
		Files: map[string]endpoint.FileMeta{
			"old.txt":  {RelPath: "old.txt", Size: 1},
			"keep.txt": {RelPath: "keep.txt", Size: 2},
		},
	}
	plan := transfer.Plan{
		Items: []transfer.TransferItem{
			{RelPath: "old.txt", Action: transfer.ActionDelete},
			{RelPath: "keep.txt", Action: transfer.ActionSkip},
		},
	}
	result := transfer.Result{
		Success: map[string]endpoint.FileMeta{
			"new.txt": {RelPath: "new.txt", Size: 3, ModTime: time.Now()},
		},
	}
	final := mergeSnapshot(last, plan, result)
	if _, ok := final["old.txt"]; ok {
		t.Fatalf("deleted file should not remain")
	}
	if _, ok := final["new.txt"]; !ok {
		t.Fatalf("new file missing")
	}
	if _, ok := final["keep.txt"]; !ok {
		t.Fatalf("skip file should persist")
	}
}
