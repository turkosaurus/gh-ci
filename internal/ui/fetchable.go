package ui

import (
	"sync"
	"time"
)

// LoadState tracks the progression of data loading for a Fetchable field.
type LoadState int

const (
	LoadIdle    LoadState = iota // never fetched
	LoadLocal                   // from local/fast source
	LoadPartial                 // partial remote data
	LoadReady                   // fully loaded
	LoadError                   // failed, no prior data
)

// Fetchable wraps a value with loading state metadata.
type Fetchable[T any] struct {
	mu        sync.RWMutex
	Data      T
	State     LoadState
	Fetching  bool // orthogonal: is a fetch in flight?
	Err       error
	fetchedAt time.Time
}

// FetchedAt returns when data was last successfully fetched.
func (f *Fetchable[T]) FetchedAt() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.fetchedAt
}

// SetLocal sets Data from a fast local source.
func (f *Fetchable[T]) SetLocal(data T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Data = data
	f.State = LoadLocal
	f.Fetching = false
	f.Err = nil
	f.fetchedAt = time.Now()
}

// SetPartial sets Data from a partial remote fetch (e.g. first page).
func (f *Fetchable[T]) SetPartial(data T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Data = data
	f.State = LoadPartial
	f.Fetching = false
	f.Err = nil
	f.fetchedAt = time.Now()
}

// SetData sets Data as fully loaded.
func (f *Fetchable[T]) SetData(data T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Data = data
	f.State = LoadReady
	f.Fetching = false
	f.Err = nil
	f.fetchedAt = time.Now()
}

// SetError records an error. If prior data exists (Local/Partial/Ready),
// the state is preserved (stale data kept). Otherwise state becomes LoadError.
func (f *Fetchable[T]) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Err = err
	f.Fetching = false
	if !f.hasData() {
		f.State = LoadError
	}
}

// SetFetching marks a fetch as in-flight without changing state or data.
func (f *Fetchable[T]) SetFetching() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Fetching = true
}

// IsReady returns true when data is fully loaded.
func (f *Fetchable[T]) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.State == LoadReady
}

// HasData returns true when any data is available (local, partial, or ready).
func (f *Fetchable[T]) HasData() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.hasData()
}

// hasData is the internal unlocked version.
func (f *Fetchable[T]) hasData() bool {
	return f.State == LoadLocal || f.State == LoadPartial || f.State == LoadReady
}

// IsFetching returns true when a fetch is in-flight.
func (f *Fetchable[T]) IsFetching() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.Fetching
}
