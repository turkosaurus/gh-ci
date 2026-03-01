package ui

import (
	"sync"
	"time"

	"github.com/turkosaurus/gh-ci/internal/types"
)

type RunsCache struct {
	mu    sync.RWMutex
	data  []types.WorkflowRun
	mtime time.Time
}

func (c *RunsCache) Set(runs []types.WorkflowRun) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = runs
	c.mtime = time.Now()
}

func (c *RunsCache) Get() ([]types.WorkflowRun, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.data) == 0 {
		return nil, false
	}
	// Return a copy to prevent external mutations
	result := make([]types.WorkflowRun, len(c.data))
	copy(result, c.data)
	return result, true
}

func (c *RunsCache) LastFetchedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mtime
}
