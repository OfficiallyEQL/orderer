package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	got := &bytes.Buffer{}
	cfg := &Config{out: got, Store: "julias-delights"}
	err := cfg.Run()
	require.NoError(t, err)
	want := `Syncing order for store "julias-delights"` + "\n"
	require.Equal(t, want, got.String())
}
