package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	otProduct        string
	otProducts       []string
	otPurchaseOption string
	otOffer          string
	otOffers         []string
	otActivate       []string
	otDeactivate     []string
	otCancel         []string
	otFromJSON       string
	otUpdateMask     string
	otRegionsVersion string
	otAllowMissing   bool
)

var oneTimeCmd = &cobra.Command{
	Use:     "one-time-products",
	Aliases: []string{"otp"},
	Short:   "One-time products with purchase options and offers (monetization API)",
	Long: `Manages the newer one-time product model, which replaces plain managed
products for apps that need purchase options (for example rentals versus
purchases) and time-limited offers on them.

Existing managed products created with "gplay products" also appear here; the
"gplay products" commands remain the simpler way to manage them.

Purchase options and offers are nested in the product resource, so they are
written with "one-time-products update --update-mask purchaseOptions"; the
dedicated subcommands cover the state changes that have their own endpoints.`,
}

var otListCmd = &cobra.Command{
	Use:   "list",
	Short: "List one-time products",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			items, err := c.ListAll(ctx, appPath(pkg, "/oneTimeProducts?pageSize=100"), "oneTimeProducts")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No one-time products.")
				return nil
			}
			w := newTable("PRODUCT ID", "PURCHASE OPTIONS", "TITLE")
			for _, p := range items {
				options := []string{}
				for _, o := range p.Docs("purchaseOptions") {
					options = append(options, fmt.Sprintf("%s(%s)", o.Str("purchaseOptionId"), o.Str("state")))
				}
				title := ""
				if listings := p.Docs("listings"); len(listings) > 0 {
					title = listings[0].Str("title")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Str("productId"), commaJoin(options), dash(title))
			}
			return w.Flush()
		})
	},
}

var otGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show one one-time product",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/oneTimeProducts/%s", esc(otProduct)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otBatchGetCmd = &cobra.Command{
	Use:   "batch-get",
	Short: "Fetch several one-time products in one request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{}
			for _, id := range otProducts {
				params.Add("productIds", id)
			}
			doc, err := c.GetMap(ctx, appPath(pkg, "/oneTimeProducts:batchGet?%s", params.Encode()))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Create or update a one-time product",
	Long: `Patches a one-time product. Pass --allow-missing to create it when it does
not exist yet. The update mask names the top-level fields to replace, e.g.
--update-mask listings,purchaseOptions.

Note the API path for this method is lower-case "onetimeproducts" while the
other methods use "oneTimeProducts"; that quirk is handled here.`,
	Example: `  gplay one-time-products update --product rental_48h --allow-missing --update-mask listings,purchaseOptions --from-json @product.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(otFromJSON, &body); err != nil {
				return err
			}
			body["packageName"] = pkg
			body["productId"] = otProduct
			params := url.Values{"regionsVersion.version": {otRegionsVersion}}
			mask := otUpdateMask
			if mask == "" {
				mask = joinKeys(body, "packageName", "productId")
			}
			params.Set("updateMask", mask)
			if otAllowMissing {
				params.Set("allowMissing", "true")
			}
			// The patch method really is spelled "onetimeproducts" in the API.
			if _, err := c.Patch(ctx, appPath(pkg, "/onetimeproducts/%s?%s", esc(otProduct), params.Encode()), body); err != nil {
				return err
			}
			fmt.Printf("One-time product %s updated (mask: %s).\n", otProduct, mask)
			return nil
		})
	},
}

var otDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a one-time product",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if err := c.Delete(ctx, appPath(pkg, "/oneTimeProducts/%s", esc(otProduct))); err != nil {
				return err
			}
			fmt.Printf("One-time product %s deleted.\n", otProduct)
			return nil
		})
	},
}

var otBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete",
	Short: "Delete several one-time products in one request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			requests := make([]any, 0, len(otProducts))
			for _, id := range otProducts {
				requests = append(requests, map[string]any{"packageName": pkg, "productId": id})
			}
			if _, err := c.Post(ctx, appPath(pkg, "/oneTimeProducts:batchDelete"),
				map[string]any{"requests": requests}); err != nil {
				return err
			}
			fmt.Printf("Deleted %d one-time products.\n", len(otProducts))
			return nil
		})
	},
}

var otBatchUpdateCmd = &cobra.Command{
	Use:   "batch-update",
	Short: "Create or update several one-time products in one request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(otFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/oneTimeProducts:batchUpdate"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Purchase options --------------------------------------------------------------

var otOptionsCmd = &cobra.Command{
	Use:   "purchase-options",
	Short: "Purchase options of a one-time product",
}

var otOptionsStatesCmd = &cobra.Command{
	Use:   "set-states",
	Short: "Activate or deactivate purchase options in bulk",
	Long: `Takes a BatchUpdatePurchaseOptionStatesRequest body, whose requests each
activate or deactivate one purchase option.`,
	Example: `  gplay one-time-products purchase-options set-states --product rental_48h --from-json @states.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(otFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/oneTimeProducts/%s/purchaseOptions:batchUpdateStates", esc(otProduct)), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otOptionsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete purchase options in bulk",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{"requests": []any{map[string]any{
				"packageName": pkg, "productId": otProduct, "purchaseOptionId": otPurchaseOption,
			}}}
			if otFromJSON != "" {
				if err := jsonFromFlag(otFromJSON, &body); err != nil {
					return err
				}
			}
			if _, err := c.Post(ctx, appPath(pkg, "/oneTimeProducts/%s/purchaseOptions:batchDelete", esc(otProduct)), body); err != nil {
				return err
			}
			fmt.Println("Purchase options deleted.")
			return nil
		})
	},
}

// --- One-time product offers ---------------------------------------------------------

var otOffersCmd = &cobra.Command{
	Use:   "offers",
	Short: "Offers on a purchase option",
}

var otOffersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the offers of a purchase option",
	Long:  `Pass --purchase-option "-" to list the offers of every purchase option.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			items, err := c.ListAll(ctx, appPath(pkg, "/oneTimeProducts/%s/purchaseOptions/%s/offers?pageSize=100",
				esc(otProduct), esc(otPurchaseOption)), "oneTimeProductOffers")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No offers.")
				return nil
			}
			w := newTable("OFFER", "PURCHASE OPTION", "STATE", "REGIONS")
			for _, o := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", o.Str("offerId"), o.Str("purchaseOptionId"),
					o.Str("state"), len(o.Docs("regionalPricingAndAvailabilityConfigs")))
			}
			return w.Flush()
		})
	},
}

var otOffersActivateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate a one-time product offer",
	RunE:  otOfferState("activate"),
}

var otOffersDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate a one-time product offer",
	RunE:  otOfferState("deactivate"),
}

var otOffersCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a one-time product offer that is currently running",
	RunE:  otOfferState("cancel"),
}

var otOffersBatchGetCmd = &cobra.Command{
	Use:     "batch-get",
	Short:   "Fetch several offers of a purchase option in one request",
	Long:    `Pass --offer repeatedly, or a full BatchGetOneTimeProductOffersRequest with --from-json.`,
	Example: `  gplay one-time-products offers batch-get --product rental_48h --purchase-option standard --offer launch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body, err := otOfferBatch(pkg, otOffers, "")
			if err != nil {
				return err
			}
			// A batch get reads only, so it runs even under --dry-run.
			doc, err := c.PostReadOnly(ctx, otOffersPath(pkg, ":batchGet"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otOffersBatchUpdateCmd = &cobra.Command{
	Use:   "batch-update",
	Short: "Create or update several one-time product offers in one request",
	Long: `Takes a BatchUpdateOneTimeProductOffersRequest body:
{"requests":[{"oneTimeProductOffer":{...},"updateMask":"regionalPricingAndAvailabilityConfigs","allowMissing":true}, ...]}.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(otFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, otOffersPath(pkg, ":batchUpdate"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otOffersBatchStatesCmd = &cobra.Command{
	Use:   "batch-set-states",
	Short: "Activate, deactivate and cancel several offers in one request",
	Long: `Build the batch from --activate / --deactivate / --cancel, or pass a full
BatchUpdateOneTimeProductOfferStatesRequest with --from-json.`,
	Example: `  gplay one-time-products offers batch-set-states --product rental_48h --purchase-option standard --activate launch --cancel old-promo`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{}
			if otFromJSON != "" {
				if err := jsonFromFlag(otFromJSON, &body); err != nil {
					return err
				}
			} else {
				requests := []any{}
				for _, id := range otActivate {
					requests = append(requests, map[string]any{"activateOneTimeProductOfferRequest": otOfferRef(pkg, id)})
				}
				for _, id := range otDeactivate {
					requests = append(requests, map[string]any{"deactivateOneTimeProductOfferRequest": otOfferRef(pkg, id)})
				}
				for _, id := range otCancel {
					requests = append(requests, map[string]any{"cancelOneTimeProductOfferRequest": otOfferRef(pkg, id)})
				}
				if len(requests) == 0 {
					return fmt.Errorf("nothing to do: pass --activate / --deactivate / --cancel, or --from-json")
				}
				body["requests"] = requests
			}
			doc, err := c.Post(ctx, otOffersPath(pkg, ":batchUpdateStates"), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var otOffersBatchDeleteCmd = &cobra.Command{
	Use:     "batch-delete",
	Short:   "Delete several one-time product offers in one request",
	Example: `  gplay one-time-products offers batch-delete --product rental_48h --purchase-option standard --offer old-promo`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body, err := otOfferBatch(pkg, otOffers, otFromJSON)
			if err != nil {
				return err
			}
			if _, err := c.Post(ctx, otOffersPath(pkg, ":batchDelete"), body); err != nil {
				return err
			}
			fmt.Printf("Deleted %d offers.\n", len(otOffers))
			return nil
		})
	},
}

func otOffersPath(pkg, suffix string) string {
	return appPath(pkg, "/oneTimeProducts/%s/purchaseOptions/%s/offers%s",
		esc(otProduct), esc(otPurchaseOption), suffix)
}

func otOfferRef(pkg, offerID string) map[string]any {
	return map[string]any{
		"packageName": pkg, "productId": otProduct,
		"purchaseOptionId": otPurchaseOption, "offerId": offerID,
	}
}

// otOfferBatch builds a {"requests":[...]} body from repeated --offer flags, or
// decodes fromJSON when it is given.
func otOfferBatch(pkg string, offers []string, fromJSON string) (map[string]any, error) {
	if fromJSON != "" {
		var body map[string]any
		if err := jsonFromFlag(fromJSON, &body); err != nil {
			return nil, err
		}
		return body, nil
	}
	if len(offers) == 0 {
		return nil, fmt.Errorf("pass --offer (repeatable) or --from-json")
	}
	requests := make([]any, 0, len(offers))
	for _, id := range offers {
		requests = append(requests, otOfferRef(pkg, id))
	}
	return map[string]any{"requests": requests}, nil
}

func otOfferState(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{
				"packageName": pkg, "productId": otProduct,
				"purchaseOptionId": otPurchaseOption, "offerId": otOffer,
			}
			path := appPath(pkg, "/oneTimeProducts/%s/purchaseOptions/%s/offers/%s:%s",
				esc(otProduct), esc(otPurchaseOption), esc(otOffer), action)
			if _, err := c.Post(ctx, path, body); err != nil {
				return err
			}
			fmt.Printf("Offer %s: %s done.\n", otOffer, action)
			return nil
		})
	}
}

func joinKeys(body map[string]any, skip ...string) string {
	return strings.Join(topLevelKeys(body, skip...), ",")
}

func init() {
	offerCmds := []*cobra.Command{
		otOffersListCmd, otOffersActivateCmd, otOffersDeactivateCmd, otOffersCancelCmd,
		otOffersBatchGetCmd, otOffersBatchUpdateCmd, otOffersBatchStatesCmd, otOffersBatchDeleteCmd,
	}
	productScoped := append([]*cobra.Command{
		otGetCmd, otUpdateCmd, otDeleteCmd,
		otOptionsStatesCmd, otOptionsDeleteCmd,
	}, offerCmds...)
	for _, sub := range productScoped {
		sub.Flags().StringVar(&otProduct, "product", "", "one-time product id (required)")
		_ = sub.MarkFlagRequired("product")
	}
	for _, sub := range append([]*cobra.Command{otOptionsDeleteCmd}, offerCmds...) {
		sub.Flags().StringVar(&otPurchaseOption, "purchase-option", "", "purchase option id (required)")
		_ = sub.MarkFlagRequired("purchase-option")
	}
	for _, sub := range []*cobra.Command{otOffersBatchGetCmd, otOffersBatchDeleteCmd} {
		sub.Flags().StringArrayVar(&otOffers, "offer", nil, "offer id; repeatable")
	}
	otOffersBatchStatesCmd.Flags().StringArrayVar(&otActivate, "activate", nil, "offer id to activate; repeatable")
	otOffersBatchStatesCmd.Flags().StringArrayVar(&otDeactivate, "deactivate", nil, "offer id to deactivate; repeatable")
	otOffersBatchStatesCmd.Flags().StringArrayVar(&otCancel, "cancel", nil, "offer id to cancel; repeatable")
	for _, sub := range []*cobra.Command{otOffersBatchGetCmd, otOffersBatchUpdateCmd, otOffersBatchStatesCmd, otOffersBatchDeleteCmd} {
		sub.Flags().StringVar(&otFromJSON, "from-json", "", "batch request JSON, or @file")
	}
	_ = otOffersBatchUpdateCmd.MarkFlagRequired("from-json")
	for _, sub := range []*cobra.Command{otOffersActivateCmd, otOffersDeactivateCmd, otOffersCancelCmd} {
		sub.Flags().StringVar(&otOffer, "offer", "", "offer id (required)")
		_ = sub.MarkFlagRequired("offer")
	}
	for _, sub := range []*cobra.Command{otBatchGetCmd, otBatchDeleteCmd} {
		sub.Flags().StringArrayVar(&otProducts, "product", nil, "one-time product id; repeatable (required)")
		_ = sub.MarkFlagRequired("product")
	}
	for _, sub := range []*cobra.Command{otUpdateCmd, otBatchUpdateCmd, otOptionsStatesCmd, otOptionsDeleteCmd} {
		sub.Flags().StringVar(&otFromJSON, "from-json", "", "request JSON, or @file")
	}
	otUpdateCmd.Flags().StringVar(&otUpdateMask, "update-mask", "", "comma-separated fields to replace (inferred from the body when omitted)")
	otUpdateCmd.Flags().StringVar(&otRegionsVersion, "regions-version", defaultRegionsVersion, "region catalogue version for price changes")
	otUpdateCmd.Flags().BoolVar(&otAllowMissing, "allow-missing", false, "create the product when it does not exist")
	_ = otUpdateCmd.MarkFlagRequired("from-json")
	_ = otBatchUpdateCmd.MarkFlagRequired("from-json")
	_ = otOptionsStatesCmd.MarkFlagRequired("from-json")

	otOptionsCmd.AddCommand(otOptionsStatesCmd, otOptionsDeleteCmd)
	otOffersCmd.AddCommand(otOffersListCmd, otOffersActivateCmd, otOffersDeactivateCmd, otOffersCancelCmd,
		otOffersBatchGetCmd, otOffersBatchUpdateCmd, otOffersBatchStatesCmd, otOffersBatchDeleteCmd)
	oneTimeCmd.AddCommand(otListCmd, otGetCmd, otBatchGetCmd, otUpdateCmd, otDeleteCmd,
		otBatchDeleteCmd, otBatchUpdateCmd, otOptionsCmd, otOffersCmd)
	rootCmd.AddCommand(oneTimeCmd)
}
