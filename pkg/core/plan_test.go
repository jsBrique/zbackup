package core

import (
	"testing"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
	"zbackup/pkg/transfer"
)

func TestBuildPlanIncr(t *testing.T) {
	files := []endpoint.FileMeta{
		{RelPath: "a.txt", Size: 10, ModTime: time.Unix(1, 0), IsDir: false},
		{RelPath: "b.txt", Size: 20, ModTime: time.Unix(2, 0), IsDir: false},
	}
	last := &meta.Snapshot{
		Files: map[string]endpoint.FileMeta{
			"a.txt": {RelPath: "a.txt", Size: 10, ModTime: time.Unix(1, 0), IsDir: false},
		},
	}
	cfg := BackupConfig{
		Source: endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: "/src"},
		Dest:   endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: "/dst"},
		Mode:   endpoint.ModeIncr,
	}
	plan := BuildPlan(files, last, cfg)
	if plan.TotalFiles != 1 {
		t.Fatalf("expected 1 file transfer, got %d", plan.TotalFiles)
	}
	if len(plan.Items) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan.Items))
	}
}

func TestShouldSkipWithChecksum(t *testing.T) {
	oldMeta := endpoint.FileMeta{
		RelPath:  "a.txt",
		Size:     10,
		ModTime:  time.Unix(1, 0),
		Checksum: "abc",
	}
	newMeta := endpoint.FileMeta{
		RelPath:  "a.txt",
		Size:     10,
		ModTime:  time.Unix(1, 0),
		Checksum: "abc",
	}
	last := &meta.Snapshot{Files: map[string]endpoint.FileMeta{"a.txt": oldMeta}}
	cfg := BackupConfig{Mode: endpoint.ModeIncr, Checksum: endpoint.ChecksumSHA256}
	if !shouldSkip("a.txt", newMeta, last, cfg) {
		t.Fatalf("should skip identical checksum")
	}
	newMeta.Checksum = "def"
	if shouldSkip("a.txt", newMeta, last, cfg) {
		t.Fatalf("should not skip when checksum differs")
	}
}

func TestBuildPlanCreatesDirectories(t *testing.T) {
	files := []endpoint.FileMeta{
		{RelPath: "dir", IsDir: true},
		{RelPath: "dir/file.txt", Size: 5, ModTime: time.Unix(1, 0)},
	}
	cfg := BackupConfig{
		Source: endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: "/src"},
		Dest:   endpoint.Endpoint{Type: endpoint.EndpointLocal, Path: "/dst"},
		Mode:   endpoint.ModeIncr,
	}
	plan := BuildPlan(files, nil, cfg)
	if len(plan.Items) < 2 {
		t.Fatalf("expect mkdir and upload")
	}
	if plan.Items[0].Action != transfer.ActionMkdir {
		t.Fatalf("first action should be mkdir, got %s", plan.Items[0].Action)
	}
}
