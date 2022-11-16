package order

import (
	"fmt"
	"strconv"
	"strings"

	goshopify "github.com/bold-commerce/go-shopify/v3"
)

type CreateOptions struct {
	Unique        bool
	VerifyProduct bool
	Inventory     bool
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

type InventoryLevel struct {
	Available       int   `json:"available"`
	LocationID      int64 `json:"location_id"`
	InventoryItemID int64 `json:"inventory_item_id"`
}

type InventoryLevelAdjustment struct {
	AvailableAdjustment int   `json:"available_adjustment"`
	LocationID          int64 `json:"location_id"`
	InventoryItemID     int64 `json:"inventory_item_id"`
}

type InventoryLevelsResource struct {
	InventoryLevels []*InventoryLevel `json:"inventory_levels"`
}

type InventoryLevelResource struct {
	InventoryLevel *InventoryLevel `json:"inventory_level"`
}

type VariantGQLResult struct {
	Data struct {
		ProductVariants struct {
			Edges []struct {
				Node struct {
					ID            string
					Title         string
					InventoryItem struct {
						ID             string
						LocationsCount int
					}
				}
			}
		}
	}
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
	var inventories []*InventoryLevel
	if opts.VerifyProduct || opts.Inventory {
		var err error
		if inventories, err = getInventories(client, order); err != nil {
			return nil, err
		}
	}
	result, err := client.Order.Create(*order)
	if err != nil {
		return nil, err
	}
	if !opts.Inventory {
		return result, nil
	}
	for _, i := range inventories {
		_, err := AdjustIventoryLevel(client, i.LocationID, i.InventoryItemID, 0, -1)
		if err != nil {
			return nil, err
		}
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
	if len(orders) == 0 {
		order, err := Create(client, order, CreateOptions{VerifyProduct: opts.VerifyProduct})
		if err != nil {
			return nil, err
		}
		result := &MergeResult{Label: "created", OrderID: order.ID}
		return result, nil
	}
	if opts.VerifyProduct {
		_, err := getInventories(client, order)
		if err != nil {
			return nil, err
		}
	}
	order.ID = orders[0].ID
	order, err = client.Order.Update(*order)
	if err != nil {
		return nil, err
	}
	result := &MergeResult{Label: "updated", OrderID: order.ID}
	return result, nil
}

func GetIventoryLevels(client *goshopify.Client, inventoryItemID, variantID int64) ([]*InventoryLevel, error) {
	if inventoryItemID == 0 {
		variant, err := client.Variant.Get(variantID, nil)
		if err != nil {
			return nil, err
		}
		inventoryItemID = variant.InventoryItemId
	}
	query := struct {
		InventoryItemID int64 `url:"inventory_item_ids"`
	}{InventoryItemID: inventoryItemID}
	resource := InventoryLevelsResource{}
	err := client.Get("inventory_levels.json", &resource, query)
	if err != nil {
		return nil, err
	}
	return resource.InventoryLevels, nil
}

func GetIventoryLevel(client *goshopify.Client, inventoryItemID, variantID int64) (*InventoryLevel, error) {
	levels, err := GetIventoryLevels(client, inventoryItemID, variantID)
	if err != nil {
		return nil, err
	}
	if len(levels) != 1 {
		return nil, fmt.Errorf("invalid inventory: product variant %d in multiple locations", variantID)
	}
	level := levels[0]
	if level.Available < 1 {
		return nil, fmt.Errorf("invalid inventory: not enough items available (%d)", level.Available)
	}
	return level, nil
}

func AdjustIventoryLevel(client *goshopify.Client, locaitonID, inventoryItemID, variantID int64, amount int) (*InventoryLevel, error) {
	adjustment := InventoryLevelAdjustment{
		InventoryItemID:     inventoryItemID,
		LocationID:          locaitonID,
		AvailableAdjustment: amount,
	}
	if inventoryItemID == 0 || locaitonID == 0 {
		level, err := GetIventoryLevel(client, inventoryItemID, variantID)
		if err != nil {
			return nil, err
		}
		adjustment.InventoryItemID = level.InventoryItemID
		adjustment.LocationID = level.LocationID
	}
	resource := InventoryLevelResource{}
	err := client.Post("inventory_levels/adjust.json", adjustment, &resource)
	if err != nil {
		return nil, err
	}
	return resource.InventoryLevel, nil
}

func GetVariantIDBySKU(client *goshopify.Client, sku string) (int64, error) {
	requestPayload := struct {
		Query     string `json:"query"`
		Variables struct {
			Filter string `json:"filter"`
		} `json:"variables"`
	}{
		Query: "query($filter: String!) { productVariants(first: 2, query: $filter) { edges { node { id  title inventoryItem  { id locationsCount } } } } }",
		Variables: struct {
			Filter string `json:"filter"`
		}{
			Filter: fmt.Sprintf("sku:%s", sku),
		},
	}
	resource := VariantGQLResult{}
	err := client.Post("graphql.json", requestPayload, &resource)
	if err != nil {
		return 0, err
	}
	e := resource.Data.ProductVariants.Edges
	if len(e) > 1 || len(e) == 0 {
		return 0, fmt.Errorf("%d product variants found with sku %q", len(e), sku)
	}
	// potentially later use locationCount for early checks
	return idFromGID(e[0].Node.ID)
}

func idFromGID(gid string) (int64, error) {
	idx := strings.LastIndex(gid, "/")
	if idx == -1 {
		return 0, fmt.Errorf("gid %q doesn't contain %q", gid, "/")
	}
	idStr := gid[idx+1:]
	i, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return int64(i), nil
}

func getInventories(client *goshopify.Client, order *goshopify.Order) ([]*InventoryLevel, error) {
	levels := make([]*InventoryLevel, 0, len(order.LineItems))
	for _, lineItem := range order.LineItems {
		if lineItem.VariantID != 0 {
			level, err := GetIventoryLevel(client, 0, lineItem.VariantID)
			if err != nil {
				return nil, err
			}
			levels = append(levels, level)
		}
	}
	return levels, nil
}
