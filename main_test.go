package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	got := &bytes.Buffer{}
	cfg := testConfig(t, got)
	err := cfg.Run()
	require.NoError(t, err)
	store := getenv(t, "SHOPIFY_STORE")
	want := `
Syncing order for store "` + store + `"
Order count: 0
`
	require.Equal(t, want[1:], got.String())
}

func testConfig(t *testing.T, out io.Writer) *Config {
	t.Helper()
	return &Config{
		Store: getenv(t, "SHOPIFY_STORE"),
		Token: getenv(t, "SHOPIFY_TOKEN"),
		out:   out,
	}
}

func getenv(t *testing.T, name string) string {
	t.Helper()
	val, ok := os.LookupEnv(name)
	if !ok {
		t.Fatalf("missing environment variable %s", name)
	}
	return val
}
