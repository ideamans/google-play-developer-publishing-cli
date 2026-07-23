package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	externalID       string
	externalFromJSON string
)

var externalCmd = &cobra.Command{
	Use:     "external-transactions",
	Aliases: []string{"ext"},
	Short:   "Report transactions made with an alternative billing system",
	Long: `Apps that use an alternative billing system (or user choice billing) must
report each transaction to Google. These commands wrap that reporting API.`,
}

var externalGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show a reported external transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/externalTransactions/%s", esc(externalID)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var externalCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Report a new external transaction",
	Long: `Takes an ExternalTransaction body describing the amounts, tax and whether the
transaction is one-time or recurring.`,
	Example: `  gplay external-transactions create --id order-123 --from-json @transaction.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(externalFromJSON, &body); err != nil {
				return err
			}
			path := appPath(pkg, "/externalTransactions?externalTransactionId=%s", esc(externalID))
			doc, err := c.Post(ctx, path, body)
			if err != nil {
				return err
			}
			fmt.Printf("External transaction %s reported.\n", externalID)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var externalRefundCmd = &cobra.Command{
	Use:     "refund",
	Short:   "Report a refund of an external transaction",
	Long:    `Takes a RefundExternalTransactionRequest body: a full refund or a partial one with the refunded amount.`,
	Example: `  gplay external-transactions refund --id order-123 --from-json @refund.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{"fullRefund": map[string]any{}}
			if externalFromJSON != "" {
				if err := jsonFromFlag(externalFromJSON, &body); err != nil {
					return err
				}
			}
			doc, err := c.Post(ctx, appPath(pkg, "/externalTransactions/%s:refund", esc(externalID)), body)
			if err != nil {
				return err
			}
			fmt.Printf("Refund reported for external transaction %s.\n", externalID)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

func init() {
	for _, sub := range []*cobra.Command{externalGetCmd, externalCreateCmd, externalRefundCmd} {
		sub.Flags().StringVar(&externalID, "id", "", "external transaction id (required)")
		_ = sub.MarkFlagRequired("id")
	}
	for _, sub := range []*cobra.Command{externalCreateCmd, externalRefundCmd} {
		sub.Flags().StringVar(&externalFromJSON, "from-json", "", "request JSON, or @file")
	}
	_ = externalCreateCmd.MarkFlagRequired("from-json")

	externalCmd.AddCommand(externalGetCmd, externalCreateCmd, externalRefundCmd)
	rootCmd.AddCommand(externalCmd)
}
