package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	goshopify "github.com/bold-commerce/go-shopify/v3"
)

var (
	// version vars set by goreleaser
	version = "tip"
	commit  = "HEAD"
	date    = "now"

	description = `
orderer imports orders into Shopify store
`
	cli struct {
		Config
		Version kong.VersionFlag `help:"Show version." env:"-"`
	}
)

type Config struct {
	Store string `required:"" help:"Shopify store name as found in <name>.myshopify.com URL."`
	Token string `required:"" help:"Shopify Admin token."`
	out   io.Writer
}

func main() {
	kctx := kong.Parse(&cli,
		kong.Description(description),
		kong.DefaultEnvars("shopify"),
		kong.Vars{"version": fmt.Sprintf("%s (%s on %s)", version, commit, date)},
	)
	kctx.FatalIfErrorf(kctx.Run())
}

func (c *Config) AfterApply() error {
	c.out = os.Stdout
	return nil
}

func (c *Config) Run() error {
	client := goshopify.NewClient(goshopify.App{}, c.Store, c.Token)

	orders, err := client.Order.List(nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "Syncing order for store %q\n", c.Store)
	fmt.Fprintf(c.out, "Order count: %d\n", len(orders))
	return nil
}
