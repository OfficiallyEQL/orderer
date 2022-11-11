package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/OfficiallyEQL/orderer/order"
	"github.com/alecthomas/kong"
	goshopify "github.com/bold-commerce/go-shopify/v3"
)

var (
	// version vars set by goreleaser
	version = "tip"
	commit  = "HEAD"
	date    = "now"

	description = `
orderer creates, updates or deletes orders in Shopify store.

It can be used as an importing tool.
The order is provided in JSON file and its "name" attribute serves as
identifier.
`
)

type CLI struct {
	Get     GetCmd           `cmd:"" help:"Get order by order ID"`
	List    ListCmd          `cmd:"" help:"List first 50 orders with matching name"`
	Create  CreateCmd        `cmd:"" help:"Create order"`
	Update  UpdateCmd        `cmd:"" help:"Update order"`
	Merge   MergeCmd         `cmd:"" help:"Create or update order"`
	Delete  DeleteCmd        `cmd:"" help:"Delete order"`
	Version kong.VersionFlag `help:"Show version." env:"-"`
}

type Config struct {
	Store       string   `required:"" help:"Shopify store name as found in <name>.myshopify.com URL."`
	Token       string   `required:"" help:"Shopify Admin token."`
	ShopifyLogs LogLevel `short:"L" help:"Log level (debug, info, warn, error, none)" enum:"debug,info,warn,error,none" default:"none"`
	out         io.Writer
	client      *goshopify.Client
}

type GetCmd struct {
	Config
	ID int64 `arg:"" required:"" help:"order ID"`
}

type ListCmd struct {
	Config
	Order *goshopify.Order `optional:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order name to be listed (only name matters)"`
	Name  string
}

type CreateCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be created"`
	Unique        bool             `short:"u" help:"assert order name is new"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
}

type MergeCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be merged (created or updated)"`
	Unique        bool             `short:"u" help:"assert order name is used at most once"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
}

type UpdateCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be updated"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
}

type DeleteCmd struct {
	Config
	Order  *goshopify.Order `optional:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be deleted"`
	Name   string
	Unique bool `short:"u" help:"assert order name is used at most once"`
}

var kongOpts = []kong.Option{
	kong.Description(description),
	kong.DefaultEnvars("shopify"),
	kong.NamedMapper("jsonfile", JSONFileMapper),
	kong.Vars{"version": fmt.Sprintf("%s (%s on %s)", version, commit, date)},
}

func main() {
	kctx := kong.Parse(&CLI{}, kongOpts...)
	kctx.FatalIfErrorf(kctx.Run())
}

func (c *Config) AfterApply() error {
	c.out = os.Stdout
	opts := []goshopify.Option{
		goshopify.WithVersion("2022-10"),
		goshopify.WithRetry(5),
	}
	if c.ShopifyLogs != LogLevelNone {
		logger := NewLogger(os.Stdout, c.ShopifyLogs)
		opts = append(opts, goshopify.WithLogger(logger))
	}
	c.client = goshopify.NewClient(goshopify.App{}, c.Store, c.Token, opts...)
	return nil
}

func (c *GetCmd) Run() error {
	order, err := c.client.Order.Get(c.ID, nil)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(order)
}

func (c *ListCmd) AfterApply() error {
	if err := c.Config.AfterApply(); err != nil {
		return err
	}
	if c.Name == "" && c.Order.Name == "" {
		return errors.New("no order name given")
	}
	return nil
}

func (c *ListCmd) OrderName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Order.Name
}

func (c *ListCmd) Run() error {
	orders, err := order.List(c.client, c.OrderName())
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of orders:", len(orders))
	e := json.NewEncoder(c.out)
	e.SetIndent("", "  ")
	for _, o := range orders {
		fmt.Fprintf(c.out, "id: %d name: %s email: %s\n", o.ID, o.Name, o.Email)
	}
	return nil
}

func (c *DeleteCmd) AfterApply() error {
	if err := c.Config.AfterApply(); err != nil {
		return err
	}
	if c.Name == "" && c.Order.Name == "" {
		return errors.New("no order name given")
	}
	return nil
}

func (c *DeleteCmd) OrderName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Order.Name
}

func (c *DeleteCmd) Run() error {
	orders, err := order.List(c.client, c.OrderName())
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of orders to delete:", len(orders))
	if c.Unique && len(orders) > 1 {
		return fmt.Errorf("more than one order with name %q", c.Name)
	}
	for _, o := range orders {
		if err := c.client.Delete(fmt.Sprintf("orders/%d.json", o.ID)); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "order deleted, ID:", o.ID)
	}
	return nil
}

func (c *CreateCmd) Run() error {
	opts := order.CreateOptions{Unique: c.Unique, VerifyProduct: c.VerifyProduct}
	o, err := order.Create(c.client, c.Order, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order created, ID:", o.ID)
	return nil
}

func (c *UpdateCmd) Run() error {
	o, err := order.Update(c.client, c.Order)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order updated, ID:", o.ID)
	return nil
}

func (c *MergeCmd) Run() error {
	opts := order.MergeOptions{VerifyProduct: c.VerifyProduct}
	result, err := order.Merge(c.client, c.Order, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "order merged (%s), ID: %d\n", result.Label, result.OrderID)
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
