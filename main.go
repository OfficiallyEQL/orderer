package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

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
	Store string          `required:"" help:"Shopify store name as found in <name>.myshopify.com URL."`
	Token string          `required:"" help:"Shopify Admin token."`
	Order goshopify.Order `required:"" type:"jsonfile" name:"input" short:"i" help:"File containing JSON encoded order to be created"`
	out   io.Writer
}

var kongOpts = []kong.Option{
	kong.Description(description),
	kong.DefaultEnvars("shopify"),
	kong.NamedMapper("jsonfile", JSONFileMapper),
	kong.Vars{"version": fmt.Sprintf("%s (%s on %s)", version, commit, date)},
}

func main() {
	kctx := kong.Parse(&cli, kongOpts...)
	kctx.FatalIfErrorf(kctx.Run())
}

func (c *Config) AfterApply() error {
	c.out = os.Stdout
	return nil
}

func (c *Config) Run() error {
	client := goshopify.NewClient(goshopify.App{}, c.Store, c.Token)

	order, err := client.Order.Create(c.Order)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order created, ID:", order.ID)
	return nil
}

var JSONFileMapper = kong.MapperFunc(decodeJSONFile)

func decodeJSONFile(ctx *kong.DecodeContext, target reflect.Value) error {
	var fname string
	if err := ctx.Scan.PopValueInto("filename", &fname); err != nil {
		return err
	}
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(target.Addr().Interface())
}
