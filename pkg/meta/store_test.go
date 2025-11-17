package meta

import (
	"testing"

	"zbackup/pkg/endpoint"
)

func TestStoreSaveLoad(t *testing.T) {
	fs := endpoint.NewLocalFS(t.TempDir())
	store := NewStore(fs)
	snap := Snapshot{
		Name:      "snapshot-1",
		Files:     map[string]endpoint.FileMeta{"demo.txt": {RelPath: "demo.txt", Size: 100}},
		Completed: true,
	}
	if err := store.Save(snap); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("load latest failed: %v", err)
	}
	if loaded == nil || loaded.Name != snap.Name {
		t.Fatalf("unexpected snapshot: %+v", loaded)
	}
	if len(loaded.Files) != 1 {
		t.Fatalf("unexpected file count")
	}
}
