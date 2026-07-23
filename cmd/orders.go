package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	orderID       string
	orderIDs      []string
	orderRevoke   bool
	orderFromJSON string
)

var ordersCmd = &cobra.Command{
	Use:   "orders",
	Short: "Orders and refunds",
	Long: `Reads order details and issues refunds. Orders are identified by the order id
that appears in the purchase receipt and in Play Console (for example
GPA.1234-5678-9012-34567).`,
}

var ordersGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one order",
	Example: `  gplay orders get --order GPA.1234-5678-9012-34567`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/orders/%s", esc(orderID)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var ordersBatchGetCmd = &cobra.Command{
	Use:     "batch-get",
	Short:   "Fetch up to 1000 orders in one request",
	Example: `  gplay orders batch-get --order GPA.1111 --order GPA.2222`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{}
			for _, id := range orderIDs {
				params.Add("orderIds", id)
			}
			doc, err := c.GetMap(ctx, appPath(pkg, "/orders:batchGet?%s", params.Encode()))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var ordersRefundCmd = &cobra.Command{
	Use:   "refund",
	Short: "Refund an order",
	Long: `Refunds the order. With --revoke the entitlement is also withdrawn, which is
what you want for a purchase the user should no longer have.`,
	Example: `  gplay orders refund --order GPA.1234-5678-9012-34567
  gplay orders refund --order GPA.1234-5678-9012-34567 --revoke`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			path := appPath(pkg, "/orders/%s:refund", esc(orderID))
			if orderRevoke {
				path += "?revoke=true"
			}
			if _, err := c.Post(ctx, path, nil); err != nil {
				return err
			}
			fmt.Printf("Order %s refunded.\n", orderID)
			return nil
		})
	},
}

var ordersReviewRefundCmd = &cobra.Command{
	Use:   "review-refund",
	Short: "Answer a pending refund request with a recommendation",
	Long: `Responds to a user-initiated refund request that Google has forwarded, using
the pending refund token from the notification. The body is an
OrdersReviewRefundRequest: refundPreference (APPROVE / DECLINE / NEUTRAL),
consumption evidence, and the pending refund token.`,
	Example: `  gplay orders review-refund --order GPA.1234 --from-json @review.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(orderFromJSON, &body); err != nil {
				return err
			}
			if _, err := c.Post(ctx, appPath(pkg, "/orders/%s:reviewrefund", esc(orderID)), body); err != nil {
				return err
			}
			fmt.Printf("Refund review submitted for order %s.\n", orderID)
			return nil
		})
	},
}

func init() {
	for _, sub := range []*cobra.Command{ordersGetCmd, ordersRefundCmd, ordersReviewRefundCmd} {
		sub.Flags().StringVar(&orderID, "order", "", "order id, e.g. GPA.1234-5678-9012-34567 (required)")
		_ = sub.MarkFlagRequired("order")
	}
	ordersBatchGetCmd.Flags().StringArrayVar(&orderIDs, "order", nil, "order id; repeatable (required)")
	_ = ordersBatchGetCmd.MarkFlagRequired("order")
	ordersRefundCmd.Flags().BoolVar(&orderRevoke, "revoke", false, "also revoke the entitlement")
	ordersReviewRefundCmd.Flags().StringVar(&orderFromJSON, "from-json", "", "OrdersReviewRefundRequest JSON, or @file (required)")
	_ = ordersReviewRefundCmd.MarkFlagRequired("from-json")

	ordersCmd.AddCommand(ordersGetCmd, ordersBatchGetCmd, ordersRefundCmd, ordersReviewRefundCmd)
	rootCmd.AddCommand(ordersCmd)
}
