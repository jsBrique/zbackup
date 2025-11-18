package core

import (
	"sync"
	"time"

	"zbackup/pkg/endpoint"
	"zbackup/pkg/meta"
)

type checkpoint struct {
	store         *meta.Store
	snapshot      meta.Snapshot
	files         map[string]endpoint.FileMeta
	mu            sync.Mutex
	lastFlush     time.Time
	flushInterval time.Duration
	dirty         bool
}

func newCheckpoint(store *meta.Store, base *meta.Snapshot, name string, src endpoint.Endpoint, dst endpoint.Endpoint) *checkpoint {
	files := make(map[string]endpoint.FileMeta)
	if base != nil {
		for k, v := range base.Files {
			files[k] = v
		}
	}
	snap := meta.Snapshot{
		Name:       name,
		CreatedAt:  time.Now().UTC(),
		SourceRoot: src.Path,
		DestRoot:   dst.Path,
		Files:      files,
		Completed:  false,
	}
	return &checkpoint{
		store:         store,
		snapshot:      snap,
		files:         files,
		lastFlush:     time.Now(),
		flushInterval: 3 * time.Second,
	}
}

func (c *checkpoint) Record(meta endpoint.FileMeta) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[meta.RelPath] = meta
	c.dirty = true
	if time.Since(c.lastFlush) >= c.flushInterval {
		return c.flushLocked()
	}
	return nil
}

func (c *checkpoint) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.flushLocked()
}

func (c *checkpoint) flushLocked() error {
	if !c.dirty {
		return nil
	}
	c.snapshot.Files = c.files
	c.snapshot.Completed = false
	if err := c.store.SavePending(c.snapshot); err != nil {
		return err
	}
	c.lastFlush = time.Now()
	c.dirty = false
	return nil
}
