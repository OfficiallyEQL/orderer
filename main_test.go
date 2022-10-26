package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	got := &bytes.Buffer{}
	args := []string{"--input", "testdata/order1.json"}
	cfg := testConfig(t, args, got)
	err := cfg.Run()
	require.NoError(t, err)
	want := "order created, ID:"
	require.Equal(t, want, got.String()[:len(want)])
}

func testConfig(t *testing.T, args []string, out io.Writer) *Config {
	t.Helper()
	options := append([]kong.Option{kong.Exit(testExit(t))}, kongOpts...)
	cfg := &Config{}
	parser, err := kong.New(cfg, options...)
	require.NoError(t, err)
	_, err = parser.Parse(args)
	require.NoError(t, err)
	cfg.out = out
	return cfg
}

func testExit(t *testing.T) func(int) {
	return func(int) {
		t.Helper()
		t.Fatalf("unexpected exit()")
	}
}
