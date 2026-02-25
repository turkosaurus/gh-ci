package ui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchable_InitialState(t *testing.T) {
	var f Fetchable[[]string]
	require.Equal(t, LoadIdle, f.State)
	require.False(t, f.HasData())
	require.False(t, f.IsReady())
	require.False(t, f.IsFetching())
}

func TestFetchable_SetLocal(t *testing.T) {
	var f Fetchable[[]string]
	f.SetFetching()
	f.SetLocal([]string{"a"})
	require.Equal(t, LoadLocal, f.State)
	require.True(t, f.HasData())
	require.False(t, f.IsReady())
	require.False(t, f.IsFetching())
	require.Equal(t, []string{"a"}, f.Data)
}

func TestFetchable_SetPartial(t *testing.T) {
	var f Fetchable[int]
	f.SetPartial(42)
	require.Equal(t, LoadPartial, f.State)
	require.True(t, f.HasData())
	require.False(t, f.IsReady())
	require.Equal(t, 42, f.Data)
}

func TestFetchable_SetData(t *testing.T) {
	var f Fetchable[int]
	f.SetFetching()
	require.True(t, f.IsFetching())
	f.SetData(100)
	require.Equal(t, LoadReady, f.State)
	require.True(t, f.IsReady())
	require.True(t, f.HasData())
	require.False(t, f.IsFetching())
	require.NotZero(t, f.FetchedAt)
}

func TestFetchable_SetError_NoData(t *testing.T) {
	var f Fetchable[string]
	f.SetError(errors.New("boom"))
	require.Equal(t, LoadError, f.State)
	require.False(t, f.HasData())
	require.EqualError(t, f.Err, "boom")
}

func TestFetchable_SetError_PreservesStaleData(t *testing.T) {
	var f Fetchable[string]
	f.SetData("cached")
	require.Equal(t, LoadReady, f.State)
	f.SetError(errors.New("refresh failed"))
	require.Equal(t, LoadReady, f.State, "state should be preserved when prior data exists")
	require.Equal(t, "cached", f.Data)
	require.True(t, f.HasData())
	require.EqualError(t, f.Err, "refresh failed")
}

func TestFetchable_Progression(t *testing.T) {
	var f Fetchable[[]int]
	f.SetFetching()
	require.True(t, f.IsFetching())
	require.Equal(t, LoadIdle, f.State)

	f.SetLocal([]int{1})
	require.Equal(t, LoadLocal, f.State)

	f.SetPartial([]int{1, 2})
	require.Equal(t, LoadPartial, f.State)

	f.SetData([]int{1, 2, 3})
	require.Equal(t, LoadReady, f.State)
	require.True(t, f.IsReady())
}
