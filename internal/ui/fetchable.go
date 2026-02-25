package ui

import "time"

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
	Data      T
	State     LoadState
	Fetching  bool // orthogonal: is a fetch in flight?
	Err       error
	FetchedAt time.Time
}

// SetLocal sets Data from a fast local source.
func (f *Fetchable[T]) SetLocal(data T) {
	f.Data = data
	f.State = LoadLocal
	f.Fetching = false
	f.Err = nil
	f.FetchedAt = time.Now()
}

// SetPartial sets Data from a partial remote fetch (e.g. first page).
func (f *Fetchable[T]) SetPartial(data T) {
	f.Data = data
	f.State = LoadPartial
	f.Fetching = false
	f.Err = nil
	f.FetchedAt = time.Now()
}

// SetData sets Data as fully loaded.
func (f *Fetchable[T]) SetData(data T) {
	f.Data = data
	f.State = LoadReady
	f.Fetching = false
	f.Err = nil
	f.FetchedAt = time.Now()
}

// SetError records an error. If prior data exists (Local/Partial/Ready),
// the state is preserved (stale data kept). Otherwise state becomes LoadError.
func (f *Fetchable[T]) SetError(err error) {
	f.Err = err
	f.Fetching = false
	if !f.HasData() {
		f.State = LoadError
	}
}

// SetFetching marks a fetch as in-flight without changing state or data.
func (f *Fetchable[T]) SetFetching() {
	f.Fetching = true
}

// IsReady returns true when data is fully loaded.
func (f *Fetchable[T]) IsReady() bool {
	return f.State == LoadReady
}

// HasData returns true when any data is available (local, partial, or ready).
func (f *Fetchable[T]) HasData() bool {
	return f.State == LoadLocal || f.State == LoadPartial || f.State == LoadReady
}

// IsFetching returns true when a fetch is in-flight.
func (f *Fetchable[T]) IsFetching() bool {
	return f.Fetching
}
