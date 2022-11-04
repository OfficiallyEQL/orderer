package order

import (
	"fmt"

	goshopify "github.com/bold-commerce/go-shopify/v3"
)

type CreateOptions struct {
	Unique        bool
	VerifyProduct bool
}

type MergeOptions struct {
	VerifyProduct bool
}

type UpdateOptions struct {
	VerifyProduct bool
}

type MergeResult struct {
	Label   string // created or updated
	OrderID int64
}

func List(client *goshopify.Client, orderName string) ([]goshopify.Order, error) {
	if orderName == "" {
		return nil, fmt.Errorf("order name is empty")
	}
	ordersResource := goshopify.OrdersResource{}
	query := struct {
		Name string `url:"name"`
	}{Name: orderName}
	err := client.Get("orders.json", &ordersResource, query)
	if err != nil {
		return nil, err
	}
	return ordersResource.Orders, nil
}

func Create(client *goshopify.Client, order *goshopify.Order, opts CreateOptions) (*goshopify.Order, error) {
	if opts.Unique {
		orders, err := List(client, order.Name)
		if err != nil {
			return nil, err
		}
		if len(orders) != 0 {
			return nil, fmt.Errorf("order with name %q already exists", order.Name)
		}
	}
	if opts.VerifyProduct {
		err := verifyProduct(client, order)
		if err != nil {
			return nil, err
		}
	}
	result, err := client.Order.Create(*order)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Update(client *goshopify.Client, order *goshopify.Order) (*goshopify.Order, error) {
	orders, err := List(client, order.Name)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, fmt.Errorf("order with name %q does not exist", order.Name)
	}
	if len(orders) > 1 {
		return nil, fmt.Errorf("more than one order with name %q", order.Name)
	}
	o := orders[0]
	if _, err = client.Order.Update(o); err != nil {
		return nil, err
	}
	return &o, nil
}

func Merge(client *goshopify.Client, order *goshopify.Order, opts MergeOptions) (*MergeResult, error) {
	orders, err := List(client, order.Name)
	if err != nil {
		return nil, err
	}
	if len(orders) > 1 {
		return nil, fmt.Errorf("expected at most one order with name %q, found %d'", order.Name, len(orders))
	}
	if opts.VerifyProduct {
		err := verifyProduct(client, order)
		if err != nil {
			return nil, err
		}
	}

	if len(orders) == 0 {
		order, err := Create(client, order, CreateOptions{VerifyProduct: opts.VerifyProduct})
		if err != nil {
			return nil, err
		}
		result := &MergeResult{Label: "created", OrderID: order.ID}
		return result, nil
	}
	order.ID = orders[0].ID
	order, err = client.Order.Update(*order)
	if err != nil {
		return nil, err
	}
	result := &MergeResult{Label: "updated", OrderID: order.ID}
	return result, nil
}

func verifyProduct(client *goshopify.Client, order *goshopify.Order) error {
	for _, lineItem := range order.LineItems {
		if lineItem.VariantID == 0 {
			return fmt.Errorf("missing variantID for lineItem %v", lineItem)
		}
		if _, err := client.Variant.Get(lineItem.VariantID, nil); err != nil {
			return fmt.Errorf("cannot verify VariantID '%d': %w", lineItem.VariantID, err)
		}
	}
	return nil
}
