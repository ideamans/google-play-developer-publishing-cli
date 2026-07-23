package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	purchaseToken        string
	purchaseProductID    string
	purchasePayload      string
	purchaseDeferFrom    string
	purchaseDeferTo      string
	purchaseFromJSON     string
	voidedStartTime      string
	voidedEndTime        string
	voidedType           int
	voidedIncludePartial bool
)

var purchasesCmd = &cobra.Command{
	Use:   "purchases",
	Short: "Verify and manage purchases made in the app",
	Long: `Server-side purchase verification and management. These commands take the
purchase token the app receives from Google Play Billing.

The v2 endpoints ("purchases product-v2", "purchases subscription-v2") do not
need a product id and return the richer current-state model; prefer them for
new integrations.`,
}

// --- One-time product purchases ------------------------------------------------

var purchasesProductCmd = &cobra.Command{
	Use:   "product",
	Short: "One-time product purchases (v1)",
}

var purchasesProductGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show the state of a one-time product purchase",
	Example: `  gplay purchases product get --product premium_upgrade --token <purchase-token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/purchases/products/%s/tokens/%s",
				esc(purchaseProductID), esc(purchaseToken)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var purchasesProductAckCmd = &cobra.Command{
	Use:   "acknowledge",
	Short: "Acknowledge a one-time product purchase",
	Long: `Google Play refunds and revokes purchases that are not acknowledged within
three days, so a server-side integration must acknowledge every purchase it
grants entitlement for.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{}
			if purchasePayload != "" {
				body["developerPayload"] = purchasePayload
			}
			if _, err := c.Post(ctx, appPath(pkg, "/purchases/products/%s/tokens/%s:acknowledge",
				esc(purchaseProductID), esc(purchaseToken)), body); err != nil {
				return err
			}
			fmt.Println("Purchase acknowledged.")
			return nil
		})
	},
}

var purchasesProductConsumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Consume a one-time product purchase so it can be bought again",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if _, err := c.Post(ctx, appPath(pkg, "/purchases/products/%s/tokens/%s:consume",
				esc(purchaseProductID), esc(purchaseToken)), map[string]any{}); err != nil {
				return err
			}
			fmt.Println("Purchase consumed.")
			return nil
		})
	},
}

var purchasesProductV2Cmd = &cobra.Command{
	Use:   "product-v2",
	Short: "One-time product purchases (v2, no product id needed)",
}

var purchasesProductV2GetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show the state of a one-time product purchase (v2)",
	Example: `  gplay purchases product-v2 get --token <purchase-token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/purchases/productsv2/tokens/%s", esc(purchaseToken)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Subscription purchases ------------------------------------------------------

var purchasesSubCmd = &cobra.Command{
	Use:   "subscription",
	Short: "Subscription purchases (v1)",
}

var purchasesSubAckCmd = &cobra.Command{
	Use:   "acknowledge",
	Short: "Acknowledge a subscription purchase",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{}
			if purchasePayload != "" {
				body["developerPayload"] = purchasePayload
			}
			if _, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptions/%s/tokens/%s:acknowledge",
				esc(purchaseProductID), esc(purchaseToken)), body); err != nil {
				return err
			}
			fmt.Println("Subscription purchase acknowledged.")
			return nil
		})
	},
}

var purchasesSubCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a subscription at the end of the current billing period",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if _, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptions/%s/tokens/%s:cancel",
				esc(purchaseProductID), esc(purchaseToken)), map[string]any{}); err != nil {
				return err
			}
			fmt.Println("Subscription cancelled.")
			return nil
		})
	},
}

var purchasesSubDeferCmd = &cobra.Command{
	Use:   "defer",
	Short: "Push a subscription's next billing date out",
	Long: `Extends a subscription for free by moving its expiry time. Both timestamps
are milliseconds since epoch, or RFC 3339 timestamps that are converted for
you. --from must be the current expiry time.`,
	Example: `  gplay purchases subscription defer --product premium_monthly --token <token> \
    --from 2026-08-01T00:00:00Z --to 2026-09-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			from, err := epochMillis(purchaseDeferFrom)
			if err != nil {
				return fmt.Errorf("--from: %w", err)
			}
			to, err := epochMillis(purchaseDeferTo)
			if err != nil {
				return fmt.Errorf("--to: %w", err)
			}
			body := map[string]any{"deferralInfo": map[string]any{
				"expectedExpiryTimeMillis": from,
				"desiredExpiryTimeMillis":  to,
			}}
			doc, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptions/%s/tokens/%s:defer",
				esc(purchaseProductID), esc(purchaseToken)), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var purchasesSubV2Cmd = &cobra.Command{
	Use:   "subscription-v2",
	Short: "Subscription purchases (v2, no product id needed)",
}

var purchasesSubV2GetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show the current state of a subscription purchase (v2)",
	Example: `  gplay purchases subscription-v2 get --token <purchase-token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/purchases/subscriptionsv2/tokens/%s", esc(purchaseToken)))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			fmt.Printf("state:        %s\n", dash(doc.Str("subscriptionState")))
			fmt.Printf("start time:   %s\n", dash(doc.Str("startTime")))
			fmt.Printf("ack state:    %s\n", dash(doc.Str("acknowledgementState")))
			fmt.Printf("test purchase: %t\n", doc.Has("testPurchase"))
			for _, item := range doc.Docs("lineItems") {
				fmt.Printf("line item:    %s (expires %s)\n", item.Str("productId"), item.Str("expiryTime"))
			}
			return nil
		})
	},
}

var purchasesSubV2RevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke a subscription immediately (v2)",
	Long: `Ends a subscription right away, optionally refunding it. The request body
selects the refund behaviour; without --from-json the subscription is revoked
with no refund and immediate loss of access.`,
	Example: `  gplay purchases subscription-v2 revoke --token <token> --from-json @revoke.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{
				"revocationContext": map[string]any{"itemBasedRefund": map[string]any{}},
			}
			if purchaseFromJSON != "" {
				if err := jsonFromFlag(purchaseFromJSON, &body); err != nil {
					return err
				}
			}
			doc, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptionsv2/tokens/%s:revoke", esc(purchaseToken)), body)
			if err != nil {
				return err
			}
			fmt.Println("Subscription revoked.")
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var purchasesSubV2CancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a subscription at the end of the billing period (v2)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{}
			if purchaseFromJSON != "" {
				if err := jsonFromFlag(purchaseFromJSON, &body); err != nil {
					return err
				}
			}
			if _, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptionsv2/tokens/%s:cancel", esc(purchaseToken)), body); err != nil {
				return err
			}
			fmt.Println("Subscription cancelled.")
			return nil
		})
	},
}

var purchasesSubV2DeferCmd = &cobra.Command{
	Use:   "defer",
	Short: "Push a subscription's next billing date out (v2)",
	Long:  `Takes a DeferSubscriptionPurchaseRequest body naming the line item and the new expiry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(purchaseFromJSON, &body); err != nil {
				return err
			}
			doc, err := c.Post(ctx, appPath(pkg, "/purchases/subscriptionsv2/tokens/%s:defer", esc(purchaseToken)), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Voided purchases --------------------------------------------------------------

var voidedCmd = &cobra.Command{
	Use:   "voided",
	Short: "Refunded, charged-back or revoked purchases",
	Long: `Lists purchases that were voided, so entitlements can be withdrawn. Google
retains 30 days of history by default and up to 60 months when a start time is
given.`,
}

var voidedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List voided purchases",
	Example: `  gplay purchases voided list
  gplay purchases voided list --start 2026-06-01T00:00:00Z --type 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{"maxResults": {"1000"}}
			if voidedStartTime != "" {
				millis, err := epochMillis(voidedStartTime)
				if err != nil {
					return fmt.Errorf("--start: %w", err)
				}
				params.Set("startTime", millis)
			}
			if voidedEndTime != "" {
				millis, err := epochMillis(voidedEndTime)
				if err != nil {
					return fmt.Errorf("--end: %w", err)
				}
				params.Set("endTime", millis)
			}
			if voidedType > 0 {
				params.Set("type", strconv.Itoa(voidedType))
			}
			if voidedIncludePartial {
				params.Set("includeQuantityBasedPartialRefund", "true")
			}
			items, err := c.ListAll(ctx, appPath(pkg, "/purchases/voidedpurchases?%s", params.Encode()), "voidedPurchases")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No voided purchases.")
				return nil
			}
			w := newTable("ORDER ID", "VOIDED", "REASON", "SOURCE", "TOKEN")
			for _, v := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					v.Str("orderId"), millisToTime(v.Str("voidedTimeMillis")),
					v.Str("voidedReason"), v.Str("voidedSource"), truncate(v.Str("purchaseToken"), 20))
			}
			return w.Flush()
		})
	},
}

// epochMillis accepts milliseconds since epoch or an RFC 3339 timestamp and
// returns milliseconds as a string.
func epochMillis(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("no value given")
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", fmt.Errorf("%q is neither milliseconds since epoch nor an RFC 3339 timestamp", value)
	}
	return strconv.FormatInt(t.UnixMilli(), 10), nil
}

func millisToTime(value string) string {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return dash(value)
	}
	return time.UnixMilli(n).Format(time.RFC3339)
}

func init() {
	tokenScoped := []*cobra.Command{
		purchasesProductGetCmd, purchasesProductAckCmd, purchasesProductConsumeCmd, purchasesProductV2GetCmd,
		purchasesSubAckCmd, purchasesSubCancelCmd, purchasesSubDeferCmd,
		purchasesSubV2GetCmd, purchasesSubV2RevokeCmd, purchasesSubV2CancelCmd, purchasesSubV2DeferCmd,
	}
	for _, sub := range tokenScoped {
		sub.Flags().StringVar(&purchaseToken, "token", "", "purchase token from Google Play Billing (required)")
		_ = sub.MarkFlagRequired("token")
	}
	for _, sub := range []*cobra.Command{
		purchasesProductGetCmd, purchasesProductAckCmd, purchasesProductConsumeCmd,
		purchasesSubAckCmd, purchasesSubCancelCmd, purchasesSubDeferCmd,
	} {
		sub.Flags().StringVar(&purchaseProductID, "product", "", "product id the token belongs to (required)")
		_ = sub.MarkFlagRequired("product")
	}
	for _, sub := range []*cobra.Command{purchasesProductAckCmd, purchasesSubAckCmd} {
		sub.Flags().StringVar(&purchasePayload, "developer-payload", "", "opaque payload to store with the purchase")
	}
	purchasesSubDeferCmd.Flags().StringVar(&purchaseDeferFrom, "from", "", "current expiry time (required)")
	purchasesSubDeferCmd.Flags().StringVar(&purchaseDeferTo, "to", "", "desired expiry time (required)")
	_ = purchasesSubDeferCmd.MarkFlagRequired("from")
	_ = purchasesSubDeferCmd.MarkFlagRequired("to")
	for _, sub := range []*cobra.Command{purchasesSubV2RevokeCmd, purchasesSubV2CancelCmd, purchasesSubV2DeferCmd} {
		sub.Flags().StringVar(&purchaseFromJSON, "from-json", "", "request JSON, or @file")
	}
	_ = purchasesSubV2DeferCmd.MarkFlagRequired("from-json")

	voidedListCmd.Flags().StringVar(&voidedStartTime, "start", "", "earliest void time (RFC 3339 or epoch millis)")
	voidedListCmd.Flags().StringVar(&voidedEndTime, "end", "", "latest void time (RFC 3339 or epoch millis)")
	voidedListCmd.Flags().IntVar(&voidedType, "type", 0, "0 = voided one-time purchases only, 1 = also voided subscriptions")
	voidedListCmd.Flags().BoolVar(&voidedIncludePartial, "include-partial-refunds", false, "include quantity-based partial refunds")

	purchasesProductCmd.AddCommand(purchasesProductGetCmd, purchasesProductAckCmd, purchasesProductConsumeCmd)
	purchasesProductV2Cmd.AddCommand(purchasesProductV2GetCmd)
	purchasesSubCmd.AddCommand(purchasesSubAckCmd, purchasesSubCancelCmd, purchasesSubDeferCmd)
	purchasesSubV2Cmd.AddCommand(purchasesSubV2GetCmd, purchasesSubV2RevokeCmd, purchasesSubV2CancelCmd, purchasesSubV2DeferCmd)
	voidedCmd.AddCommand(voidedListCmd)
	purchasesCmd.AddCommand(purchasesProductCmd, purchasesProductV2Cmd, purchasesSubCmd, purchasesSubV2Cmd, voidedCmd)
	rootCmd.AddCommand(purchasesCmd)
}
