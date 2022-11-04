package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	goshopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	got := &bytes.Buffer{}

	cfg := testConfig(t, got)
	order := testOrder(t, "testdata/order.json")
	uniqueOrderName := fmt.Sprintf("%s-%d", order.Name, time.Now().UnixMilli()%10000) // to avoid collisions on concurrent runs
	order.Name = uniqueOrderName
	deleteCmd := DeleteCmd{Config: *cfg, Order: order}
	require.NoError(t, deleteCmd.Run())

	got.Reset()
	createCmd := CreateCmd{Config: *cfg, Order: order}
	require.NoError(t, createCmd.Run())
	want := "order created, ID: "
	gotStr := got.String()
	require.Equal(t, want, gotStr[:len(want)])
	id := gotStr[len(want) : len(gotStr)-1]

	// account for latency between order creation and availability for listing
	time.Sleep(10 * time.Second)

	got.Reset()
	listCmd := ListCmd{Config: *cfg, Order: order}
	require.NoError(t, listCmd.Run())
	want = "number of orders: 1\n"
	want += fmt.Sprintf("id: %s name: %s email: jay@example.com\n", id, uniqueOrderName)
	require.Equal(t, want, got.String())

	got.Reset()
	intID, err := strconv.ParseInt(id, 10, 64)
	require.NoError(t, err)
	getCmd := GetCmd{Config: *cfg, ID: intID}
	require.NoError(t, getCmd.Run())

	got.Reset()
	mergeCmd := MergeCmd{Config: *cfg, Order: order}
	require.NoError(t, mergeCmd.Run())
	want = "order merged (updated), ID: " + id + "\n"
	require.Equal(t, want, got.String())

	got.Reset()
	updateCmd := UpdateCmd{Config: *cfg, Order: order}
	require.NoError(t, updateCmd.Run())
	want = "order updated, ID: " + id + "\n"
	require.Equal(t, want, got.String())

	require.NoError(t, createCmd.Run())
	createCmd.Unique = true

	// account for latency between order creation and availability for listing
	time.Sleep(10 * time.Second)

	require.Error(t, createCmd.Run())
	deleteCmd.Unique = true
	require.Error(t, deleteCmd.Run())
	mergeCmd.Unique = true
	require.Error(t, mergeCmd.Run())

	got.Reset()
	deleteCmd = DeleteCmd{Config: *cfg, Order: order}
	require.NoError(t, deleteCmd.Run())
	want = "number of orders to delete: 2"
	require.Equal(t, want, got.String()[:len(want)])
}

func testConfig(t *testing.T, out io.Writer) *Config {
	t.Helper()
	store := "eql-dev"
	token := getenv(t, "SHOPIFY_TOKEN")
	logger := NewLogger(io.Discard, LogLevelDebug)
	opts := []goshopify.Option{
		goshopify.WithVersion("2019-04"),
		goshopify.WithRetry(2),
		goshopify.WithLogger(logger),
	}
	client := goshopify.NewClient(goshopify.App{}, store, token, opts...)
	return &Config{
		out:    out,
		client: client,
	}
}

func getenv(t *testing.T, key string) string {
	t.Helper()
	val, ok := os.LookupEnv(key)
	if !ok {
		t.Fatalf("cannot find %q in environment", key)
	}
	return val
}

func testOrder(t *testing.T, fname string) *goshopify.Order {
	f, err := os.Open(fname)
	require.NoError(t, err)
	defer f.Close()
	o := &goshopify.Order{}
	require.NoError(t, json.NewDecoder(f).Decode(o))
	return o
}
