package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
)

var (
	// version vars set by goreleaser
	version = "tip"
	commit  = "HEAD"
	date    = "now"

	description = `
orderer imports orders into shopify store
`
	cli struct {
		Config
		Version kong.VersionFlag `help:"Show version."`
	}
)

type Config struct {
	Store string `arg:"" required:"" help:"Shopify store name as found in myshopify.com URL"`
	out   io.Writer
}

func main() {
	kctx := kong.Parse(&cli,
		kong.Description(description),
		kong.Vars{"version": fmt.Sprintf("%s (%s on %s)", version, commit, date)},
	)
	kctx.FatalIfErrorf(kctx.Run())
}

func (c *Config) AfterApply() error {
	c.out = os.Stdout
	return nil
}

func (c *Config) Run() error {
	fmt.Fprintf(c.out, "Syncing order for store %q\n", c.Store)
	return nil
}
