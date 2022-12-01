package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

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
	whitelist = map[string]bool{"eql-dev": true, "julias-delights": true}
)

type CLI struct {
	Get          GetCmd          `cmd:"" help:"Get order by order ID"`
	List         ListCmd         `cmd:"" help:"List first 50 orders with matching name"`
	Meta         MetaCmd         `cmd:"" help:"List metafields for given order"`
	Transactions TransactionsCmd `cmd:"" help:"List Transactions for given order"`
	Create       CreateCmd       `cmd:"" help:"Create order"`
	Update       UpdateCmd       `cmd:"" help:"Update order"`
	Merge        MergeCmd        `cmd:"" help:"Create or update order"`
	Delete       DeleteCmd       `cmd:"" help:"Delete order"`
	BatchDelete  BatchDeleteCmd  `cmd:"" help:"Delete orders"`
	Replace      ReplaceCmd      `cmd:"" help:"Replace order first then create new one"`

	Variant   VariantCmd   `cmd:"" help:"Get product variant by variant ID"`
	Inventory InventoryCmd `cmd:"" help:"Get inventory level including location for inventory_item_id or variant_id"`
	Customer  CustomerCmd  `cmd:""  help:"Get, List, Create, Merge and Delete customer"`
	Scopes    ScopesCmd    `cmd:"" help:"Get scopes for given Admin token"`

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
	Name  string           `help:"name of order(s) to be listed"`
}

type MetaCmd struct {
	Config
	ID int64 `arg:"" required:"" help:"order ID"`
}

type TransactionsCmd struct {
	Config
	ID int64 `arg:"" required:"" help:"order ID"`
}

type CreateCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be created"`
	Unique        bool             `short:"u" help:"assert order name is new"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
	Inventory     bool             `short:"i" help:"update inventory (-1) when order is created"`
}

type MergeCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be merged (created or updated)"`
	Unique        bool             `short:"u" help:"assert order name is used at most once"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
	Inventory     bool             `short:"i" help:"update inventory (-1) if order is created"`
}

type UpdateCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be updated"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
}

type DeleteCmd struct {
	Config
	Order  *goshopify.Order `optional:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be deleted"`
	Name   string           `help:"name of order(s) to be deleted" xor:"id"`
	ID     int64            `help:"id of order to be deleted" xor:"id"`
	Unique bool             `short:"u" help:"assert order name is used at most once"`
}

type BatchDeleteCmd struct {
	Config
	Max int `arg:"" help:"maximum number of orders to be deleted. <= 50 (page size). default: no limit" default:"-1"`
}

type ReplaceCmd struct {
	Config
	Order         *goshopify.Order `required:"" arg:"" type:"jsonfile" placeholder:"order.json" help:"File containing JSON encoded order to be replaced"`
	Unique        bool             `short:"u" help:"assert order name is new"`
	VerifyProduct bool             `short:"p" help:"verify that product variant for given variant id exists before creating order"`
	Inventory     bool             `short:"i" help:"update inventory (-1) when order is created"`
}

type VariantCmd struct {
	Get    VariantGetCmd    `cmd:"" help:"Get Variant by ID"`
	Create VariantCreateCmd `cmd:"" help:"Get Variant"`
}

type VariantGetCmd struct {
	Config
	ID  int64  `optional:"" arg:"" help:"variant ID" xor:"id"`
	SKU string `help:"variant SKU" xor:"id"`
}

type VariantCreateCmd struct {
	Config
	Variant *goshopify.Variant `arg:"" type:"jsonfile" placeholder:"variant.json" help:"File containing JSON encoded variant to be created"`
}

type InventoryCmd struct {
	Get    InventoryGetCmd    `cmd:"" help:"Get inventory levels by variant ID or inventory item ID"`
	Adjust InventoryAdjustCmd `cmd:"" help:"Update inventory levels for given variant ID or inventory item ID."`
}

type InventoryGetCmd struct {
	Config
	InventoryItemID int64 `help:"inventory item ID" xor:"id"`
	VariantID       int64 `help:"variant ID" xor:"id"`
}

type InventoryAdjustCmd struct {
	Config
	InventoryItemID int64 `help:"inventory item ID" xor:"id"`
	VariantID       int64 `help:"variant ID" xor:"id"`
	LocationID      int64 `help:"location ID of inventory to be adjusted"`
	Amount          int   `help:"adjust inventory levels for given product. Use negative number to reduce inventory."`
}

type CustomerCmd struct {
	Get         CustomerGetCmd         `cmd:"" help:"Get customer levels by variant ID or Customer item ID"`
	List        CustomerListCmd        `cmd:"" help:"List customers by identified by email address or phone number."`
	Update      CustomerUpdateCmd      `cmd:"" help:"Update customer matched by email."`
	Merge       CustomerMergeCmd       `cmd:"" help:"Update or create matched by email."`
	Delete      CustomerDeleteCmd      `cmd:"" help:"Delete customer matched by email."`
	BatchDelete CustomerBatchDeleteCmd `cmd:"" help:"Update customers."`
	Create      CustomerCreateCmd      `cmd:"" help:"Create customer from JSON."`
}

type CustomerGetCmd struct {
	Config
	ID int64 `arg:"" help:"Get customer by ID"`
}

type CustomerListCmd struct {
	Config
	Email string `arg:"" optional:"" help:"Customer item ID" xor:"id"`
	Phone string `help:"variant ID" xor:"id"`
}

type CustomerCreateCmd struct {
	Config
	Customer *goshopify.Customer `arg:"" type:"jsonfile" placeholder:"custoiemr.json" help:"File containing JSON encoded customer to be created"`
}

type CustomerUpdateCmd struct {
	Config
	Customer *goshopify.Customer `arg:"" type:"jsonfile" placeholder:"custoiemr.json" help:"File containing JSON encoded customer to be updated, matched by email"`
}

type CustomerMergeCmd struct {
	Config
	Customer *goshopify.Customer `optional:"" arg:"" type:"jsonfile" placeholder:"custoiemr.json" help:"File containing JSON encoded customer to be merged (created or updated), matched by email"`
}

type CustomerDeleteCmd struct {
	Config
	Customer *goshopify.Customer `optional:"" arg:"" type:"jsonfile" placeholder:"custoiemr.json" help:"File containing JSON encoded customer to be created" xor:"id"`
	Email    string              `help:"email of customer to be deleted." xor:"id"`
	ID       int64               `help:"ID of customer to be deleted." xor:"id"`
	Max      int                 `help:"maximum number of customers to be deleted. <= 50 (page size). default: no limit" default:"-1"`
}

type CustomerBatchDeleteCmd struct {
	Config
	Max int `arg:"" help:"maximum number of customers to be deleted. <= 50 (page size). default: no limit" default:"-1"`
}

type ScopesCmd struct {
	Config
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
	c.client = newClient(c, true)
	return nil
}

func newClient(c *Config, withVersion bool) *goshopify.Client {
	opts := []goshopify.Option{
		goshopify.WithRetry(5),
	}
	if withVersion {
		opts = append(opts, goshopify.WithVersion("2022-10"))
	}
	if c.ShopifyLogs != LogLevelNone {
		logger := NewLogger(os.Stdout, c.ShopifyLogs)
		opts = append(opts, goshopify.WithLogger(logger))
	}
	client := goshopify.NewClient(goshopify.App{}, c.Store, c.Token, opts...)
	client.Client.Timeout = 30 * time.Second
	return client
}

func (c *GetCmd) Run() error {
	order, err := c.client.Order.Get(c.ID, nil)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(order)
}

func (c *MetaCmd) Run() error {
	meta, err := order.Meta(c.client, c.ID)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(meta)
}

func (c *TransactionsCmd) Run() error {
	transactions, err := order.Transactions(c.client, c.ID)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(transactions)
}

func (c *VariantGetCmd) Run() error {
	id := c.ID
	if id == 0 {
		var err error
		id, err = order.GetVariantIDBySKU(c.client, c.SKU)
		if err != nil {
			return err
		}
	}
	variant, err := c.client.Variant.Get(id, nil)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(variant)
}

func (c *VariantCreateCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	variant, err := c.client.Variant.Create(c.Variant.ProductID, *c.Variant)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(variant)
}

func (c *InventoryGetCmd) Run() error {
	levels, err := order.GetIventoryLevels(c.client, c.InventoryItemID, c.VariantID)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(levels)
}

func (c *InventoryAdjustCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	resp, err := order.AdjustIventoryLevel(c.client, c.LocationID, c.InventoryItemID, c.VariantID, c.Amount)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(resp)
}

func (c *CustomerGetCmd) Run() error {
	customer, err := c.client.Customer.Get(c.ID, nil)
	if err != nil {
		return err
	}
	return json.NewEncoder(c.out).Encode(customer)
}

func (c *CustomerUpdateCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	customer, err := c.client.Customer.Update(*c.Customer)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "customer updated, ID:", customer.ID)
	return nil
}

func (c *CustomerMergeCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	customer, err := order.CustomerMerge(c.client, c.Customer)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "customer merged, ID:", customer.ID)
	return nil
}

func (c *CustomerDeleteCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	id := c.ID
	if id == 0 {
		email := c.Email
		if email == "" {
			email = c.Customer.Email
		}
		customers, err := order.CustomerListByEmail(c.client, email)
		if err != nil {
			return err
		}
		if len(customers) == 0 {
			return nil
		}
		if len(customers) > 1 {
			return fmt.Errorf("more than one customer found with email %q", email)
		}
		id = customers[0].ID
	}
	err := c.client.Customer.Delete(id)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "customer deleted, ID:", id)
	return nil
}

func (c *CustomerBatchDeleteCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	customers, err := c.client.Customer.List(nil)
	if err != nil {
		return err
	}
	cnt := c.Max
	if len(customers) < cnt {
		cnt = len(customers)
	}
	fmt.Fprintf(c.out, "deleting %d customers\n", cnt)

	for i := 0; i < cnt; i++ {
		err := c.client.Customer.Delete(customers[i].ID)
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, "customer deleted, ID:", customers[i].ID)
	}
	return nil
}

func (c *CustomerListCmd) Run() error {
	var customers []goshopify.Customer
	var err error
	if c.Email != "" {
		customers, err = order.CustomerListByEmail(c.client, c.Email)
	} else {
		customers, err = order.CustomerListByPhone(c.client, c.Email)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of customers:", len(customers))
	for _, customer := range customers {
		fmt.Fprintf(c.out, "id: %d name: %s %s, email: %s, phone: %s\n", customer.ID, customer.FirstName, customer.LastName, customer.Email, customer.Phone)
	}
	return nil
}

func (c *CustomerCreateCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	customer, err := c.client.Customer.Create(*c.Customer)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "customer created, ID:", customer.ID)
	return nil
}

func (c *ListCmd) AfterApply() error {
	if err := c.Config.AfterApply(); err != nil {
		return err
	}
	return nil
}

func (c *ListCmd) OrderName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Order != nil {
		return c.Order.Name
	}
	return ""
}

func (c *ListCmd) Run() error {
	name := c.OrderName()
	var orders []goshopify.Order
	var err error
	orders, err = order.List(c.client, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of orders:", len(orders))
	for _, o := range orders {
		fmt.Fprintf(c.out, "id: %d name: %s email: %s\n", o.ID, o.Name, o.Email)
	}
	return nil
}

func (c *DeleteCmd) OrderName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Order != nil {
		return c.Order.Name
	}
	return ""
}

func (c *DeleteCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	if c.ID != 0 {
		if err := order.DeleteByID(c.client, c.ID); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "order deleted, ID:", c.ID)
		return nil
	}
	opts := order.DeleteOptions{Unique: c.Unique, DryRun: true, Max: -1}
	orderIDs, err := order.Delete(c.client, c.OrderName(), opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of orders to delete:", len(orderIDs))
	for _, orderID := range orderIDs {
		if err := order.DeleteByID(c.client, orderID); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "order deleted, ID:", orderID)
	}
	return nil
}

func (c *BatchDeleteCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	opts := order.DeleteOptions{DryRun: true, Max: c.Max}
	orderIDs, err := order.Delete(c.client, "", opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "number of orders to delete:", len(orderIDs))
	for _, orderID := range orderIDs {
		if err := order.DeleteByID(c.client, orderID); err != nil {
			return err
		}
		fmt.Fprintln(c.out, "order deleted, ID:", orderID)
	}
	return nil
}

func (c *ReplaceCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	opts := order.CreateOptions{
		Unique:        c.Unique,
		VerifyProduct: c.VerifyProduct,
		Inventory:     c.Inventory,
	}
	o, err := order.Replace(c.client, c.Order, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order replaced, new ID:", o.ID)
	return nil
}

func (c *CreateCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	opts := order.CreateOptions{
		Unique:        c.Unique,
		VerifyProduct: c.VerifyProduct,
		Inventory:     c.Inventory,
	}
	o, err := order.Create(c.client, c.Order, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order created, ID:", o.ID)
	return nil
}

func (c *UpdateCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	o, err := order.Update(c.client, c.Order)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "order updated, ID:", o.ID)
	return nil
}

func (c *MergeCmd) Run() error {
	if !whitelist[c.Store] {
		return fmt.Errorf("write command for non whitelisted shop %q", c.Store)
	}
	opts := order.MergeOptions{VerifyProduct: c.VerifyProduct}
	result, err := order.Merge(c.client, c.Order, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "order merged (%s), ID: %d\n", result.Label, result.OrderID)
	return nil
}

func (c *ScopesCmd) Run() error {
	c.client = newClient(&c.Config, false)

	resource := goshopify.AccessScopesResource{}
	err := c.client.CreateAndDo("GET", "oauth/access_scopes.json", nil, nil, &resource)
	if err != nil {
		return err
	}
	scopes := make([]string, 0, len(resource.AccessScopes))

	for _, scope := range resource.AccessScopes {
		scopes = append(scopes, scope.Handle)
	}
	sort.Strings(scopes)
	fmt.Fprintf(c.out, "%d scopes:\n%s\n", len(resource.AccessScopes), strings.Join(scopes, "\n"))
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
