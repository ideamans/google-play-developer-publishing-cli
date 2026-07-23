package cmd

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	productSKU          string
	productSKUs         []string
	productStatus       string
	productPurchaseType string
	productLanguage     string
	productTitles       []string
	productDescriptions []string
	productDefaultPrice string
	productPrices       []string
	productAutoConvert  bool
	productFromJSON     string
	productLatency      string
)

var productsCmd = &cobra.Command{
	Use:     "products",
	Aliases: []string{"inappproducts"},
	Short:   "Managed in-app products (one-time purchases)",
	Long: `Manages the classic inappproducts resource: managed products and legacy
subscriptions. These commands are not edit-scoped — changes take effect
immediately.

Prices are written as CURRENCY:AMOUNT (for example JPY:150 or USD:1.99) and are
converted to the micros the API expects. Regional prices use REGION=CURRENCY:AMOUNT
(for example JP=JPY:150).`,
}

var productsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List in-app products",
	Example: `  gplay products list --package com.example.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			items, err := c.ListAll(ctx, appPath(pkg, "/inappproducts?maxResults=100"), "inappproduct")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No in-app products.")
				return nil
			}
			w := newTable("SKU", "STATUS", "TYPE", "DEFAULT PRICE", "TITLE")
			for _, p := range items {
				price := p.Doc("defaultPrice")
				title := p.Doc("listings").Doc(p.Str("defaultLanguage")).Str("title")
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					p.Str("sku"), p.Str("status"), p.Str("purchaseType"),
					formatPrice(price.Str("currency"), price.Str("priceMicros")), dash(title))
			}
			return w.Flush()
		})
	},
}

var productsGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one in-app product",
	Example: `  gplay products get --sku premium_upgrade`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/inappproducts/%s", esc(productSKU)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var productsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an in-app product",
	Long: `Creates a managed product. A product needs a default language, a title and
description in that language, and a default price.

Use --auto-convert-prices to let Google fill in the other regions from the
default price.`,
	Example: `  gplay products create --sku premium_upgrade --default-language ja \
    --title ja="プレミアム" --description ja="広告非表示" \
    --default-price JPY:480 --auto-convert-prices`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body, err := buildProduct(cmd, pkg)
			if err != nil {
				return err
			}
			path := appPath(pkg, "/inappproducts")
			if productAutoConvert {
				path += "?autoConvertMissingPrices=true"
			}
			doc, err := c.Post(ctx, path, body)
			if err != nil {
				if api.IsConflict(err) {
					return fmt.Errorf("product %q already exists; update it with \"gplay products update\": %w", productSKU, err)
				}
				return err
			}
			fmt.Printf("Product %s created.\n", productSKU)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var productsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an in-app product",
	Long:  `Patches the fields you pass and leaves the rest untouched.`,
	Example: `  gplay products update --sku premium_upgrade --default-price JPY:580 --auto-convert-prices
  gplay products update --sku premium_upgrade --status inactive`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body, err := buildProduct(cmd, pkg)
			if err != nil {
				return err
			}
			params := url.Values{}
			if productAutoConvert {
				params.Set("autoConvertMissingPrices", "true")
			}
			if productLatency != "" {
				params.Set("latencyTolerance", productLatency)
			}
			path := appPath(pkg, "/inappproducts/%s", esc(productSKU))
			if len(params) > 0 {
				path += "?" + params.Encode()
			}
			if _, err := c.Patch(ctx, path, body); err != nil {
				return err
			}
			fmt.Printf("Product %s updated.\n", productSKU)
			return nil
		})
	},
}

var productsDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an in-app product",
	Example: `  gplay products delete --sku premium_upgrade`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if err := c.Delete(ctx, appPath(pkg, "/inappproducts/%s", esc(productSKU))); err != nil {
				return err
			}
			fmt.Printf("Product %s deleted.\n", productSKU)
			return nil
		})
	},
}

var productsBatchGetCmd = &cobra.Command{
	Use:     "batch-get",
	Short:   "Fetch several products in one request",
	Example: `  gplay products batch-get --sku a --sku b --sku c`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{}
			for _, sku := range productSKUs {
				params.Add("sku", sku)
			}
			doc, err := c.GetMap(ctx, appPath(pkg, "/inappproducts:batchGet?%s", params.Encode()))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var productsBatchUpdateCmd = &cobra.Command{
	Use:     "batch-update",
	Short:   "Update up to 100 products in one request",
	Long:    `Takes an InappproductsBatchUpdateRequest body: {"requests":[{"inappproduct":{...}}, ...]}.`,
	Example: `  gplay products batch-update --from-json @products.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(productFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/inappproducts:batchUpdate"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var productsBatchDeleteCmd = &cobra.Command{
	Use:     "batch-delete",
	Short:   "Delete several products in one request",
	Example: `  gplay products batch-delete --sku a --sku b`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			requests := make([]any, 0, len(productSKUs))
			for _, sku := range productSKUs {
				requests = append(requests, map[string]any{"packageName": pkg, "sku": sku})
			}
			if _, err := c.Post(ctx, appPath(pkg, "/inappproducts:batchDelete"),
				map[string]any{"requests": requests}); err != nil {
				return err
			}
			fmt.Printf("Deleted %d products.\n", len(productSKUs))
			return nil
		})
	},
}

// buildProduct assembles an InAppProduct body from the flags.
func buildProduct(cmd *cobra.Command, pkg string) (map[string]any, error) {
	body := map[string]any{}
	if productFromJSON != "" {
		if err := jsonFromFlag(productFromJSON, &body); err != nil {
			return nil, err
		}
	}
	body["packageName"] = pkg
	body["sku"] = productSKU
	if cmd.Flags().Changed("status") {
		body["status"] = productStatus
	} else if _, ok := body["status"]; !ok && cmd.Name() == "create" {
		body["status"] = "active"
	}
	if cmd.Flags().Changed("purchase-type") {
		body["purchaseType"] = productPurchaseType
	} else if _, ok := body["purchaseType"]; !ok && cmd.Name() == "create" {
		body["purchaseType"] = "managedUser"
	}
	if cmd.Flags().Changed("default-language") {
		body["defaultLanguage"] = productLanguage
	}
	if cmd.Flags().Changed("default-price") {
		currency, micros, err := parsePrice(productDefaultPrice)
		if err != nil {
			return nil, err
		}
		body["defaultPrice"] = map[string]any{"currency": currency, "priceMicros": micros}
	}
	if len(productPrices) > 0 {
		prices := map[string]any{}
		for _, entry := range productPrices {
			region, price, err := splitKeyValue(entry)
			if err != nil {
				return nil, fmt.Errorf("--price: %w (expected REGION=CURRENCY:AMOUNT, e.g. JP=JPY:150)", err)
			}
			currency, micros, err := parsePrice(price)
			if err != nil {
				return nil, err
			}
			prices[strings.ToUpper(region)] = map[string]any{"currency": currency, "priceMicros": micros}
		}
		body["prices"] = prices
	}
	if len(productTitles) > 0 || len(productDescriptions) > 0 {
		listings := map[string]any{}
		if existing := api.Doc(body).Doc("listings"); existing != nil {
			listings = existing
		}
		add := func(values []string, field string) error {
			for _, entry := range values {
				lang, text, err := splitKeyValue(entry)
				if err != nil {
					return fmt.Errorf("--%s: %w (expected <language>=<text>)", field, err)
				}
				resolved, err := valueOrFile(text)
				if err != nil {
					return err
				}
				listing, _ := listings[lang].(map[string]any)
				if listing == nil {
					listing = map[string]any{}
				}
				listing[field] = resolved
				listings[lang] = listing
			}
			return nil
		}
		if err := add(productTitles, "title"); err != nil {
			return nil, err
		}
		if err := add(productDescriptions, "description"); err != nil {
			return nil, err
		}
		body["listings"] = listings
	}
	return body, nil
}

// parsePrice converts "JPY:150" or "USD:1.99" into a currency and micros.
func parsePrice(s string) (string, string, error) {
	currency, amount, ok := strings.Cut(s, ":")
	if !ok {
		return "", "", fmt.Errorf("invalid price %q: use CURRENCY:AMOUNT, e.g. JPY:150", s)
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return "", "", fmt.Errorf("invalid price %q: %w", s, err)
	}
	micros := int64(math.Round(value * 1_000_000))
	return strings.ToUpper(strings.TrimSpace(currency)), strconv.FormatInt(micros, 10), nil
}

// formatPrice renders micros back as a human-readable amount.
func formatPrice(currency, micros string) string {
	if currency == "" && micros == "" {
		return "-"
	}
	n, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return currency + " " + micros
	}
	value := float64(n) / 1_000_000
	if value == math.Trunc(value) {
		return fmt.Sprintf("%s %d", currency, int64(value))
	}
	return fmt.Sprintf("%s %.2f", currency, value)
}

func init() {
	for _, sub := range []*cobra.Command{productsGetCmd, productsCreateCmd, productsUpdateCmd, productsDeleteCmd} {
		sub.Flags().StringVar(&productSKU, "sku", "", "product id / SKU (required)")
		_ = sub.MarkFlagRequired("sku")
	}
	for _, sub := range []*cobra.Command{productsCreateCmd, productsUpdateCmd} {
		sub.Flags().StringVar(&productStatus, "status", "", "active or inactive")
		sub.Flags().StringVar(&productPurchaseType, "purchase-type", "", "managedUser (one-time) or subscription (legacy)")
		sub.Flags().StringVar(&productLanguage, "default-language", "", "default language, e.g. ja")
		sub.Flags().StringArrayVar(&productTitles, "title", nil, "<language>=<title>; repeatable")
		sub.Flags().StringArrayVar(&productDescriptions, "description", nil, "<language>=<description|@file>; repeatable")
		sub.Flags().StringVar(&productDefaultPrice, "default-price", "", "default price as CURRENCY:AMOUNT, e.g. JPY:480")
		sub.Flags().StringArrayVar(&productPrices, "price", nil, "regional price as REGION=CURRENCY:AMOUNT, e.g. JP=JPY:480; repeatable")
		sub.Flags().BoolVar(&productAutoConvert, "auto-convert-prices", false, "let Google convert the default price into missing regions")
		sub.Flags().StringVar(&productFromJSON, "from-json", "", "InAppProduct JSON, or @file; other flags override it")
	}
	productsUpdateCmd.Flags().StringVar(&productLatency, "latency-tolerance", "", "PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT for bulk updates")
	for _, sub := range []*cobra.Command{productsBatchGetCmd, productsBatchDeleteCmd} {
		sub.Flags().StringArrayVar(&productSKUs, "sku", nil, "product id / SKU; repeatable (required)")
		_ = sub.MarkFlagRequired("sku")
	}
	productsBatchUpdateCmd.Flags().StringVar(&productFromJSON, "from-json", "", "InappproductsBatchUpdateRequest JSON, or @file (required)")
	_ = productsBatchUpdateCmd.MarkFlagRequired("from-json")

	productsCmd.AddCommand(productsListCmd, productsGetCmd, productsCreateCmd, productsUpdateCmd,
		productsDeleteCmd, productsBatchGetCmd, productsBatchUpdateCmd, productsBatchDeleteCmd)
	rootCmd.AddCommand(productsCmd)
}
