package core

import (
	"testing"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
)

func TestBuildPlanIncr(t *testing.T) {
	files := []endpoint.FileMeta{
		{RelPath: "a.txt", Size: 10, ModTime: time.Unix(1, 0)},
		{RelPath: "b.txt", Size: 20, ModTime: time.Unix(2, 0)},
	}
	last := &meta.Snapshot{
		Files: map[string]endpoint.FileMeta{
			"a.txt": {RelPath: "a.txt", Size: 10, ModTime: time.Unix(1, 0)},
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
