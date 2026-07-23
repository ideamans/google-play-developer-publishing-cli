package cmd

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	pricingPrice       string
	pricingTaxCategory string
)

var pricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "Price conversion across regions",
}

var pricingConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert one price into every other region's price",
	Long: `Asks Google to convert a price into the equivalent price for every region,
the same calculation the Play Console offers when you set a product price. It
computes prices only — nothing is written, so this is safe to run any time.

Feed the result into "gplay products update --from-json" or a subscription
base plan's regionalConfigs.`,
	Example: `  gplay pricing convert --price USD:4.99
  gplay pricing convert --price JPY:480 --tax-category withdrawalRightEu --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			money, err := parseMoney(pricingPrice)
			if err != nil {
				return err
			}
			body := map[string]any{"price": money}
			if pricingTaxCategory != "" {
				body["productTaxCategoryCode"] = pricingTaxCategory
			}
			doc, err := c.PostReadOnly(ctx, appPath(pkg, "/pricing:convertRegionPrices"), body)
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			converted := doc.Doc("convertedRegionPrices")
			if len(converted) == 0 {
				fmt.Println("No converted prices returned.")
				return nil
			}
			regions := make([]string, 0, len(converted))
			for region := range converted {
				regions = append(regions, region)
			}
			sort.Strings(regions)

			w := newTable("REGION", "PRICE", "TAX")
			for _, region := range regions {
				entry := converted.Doc(region)
				fmt.Fprintf(w, "%s\t%s\t%s\n", region,
					formatMoney(entry.Doc("price")), formatMoney(entry.Doc("taxAmount")))
			}
			if other := doc.Doc("convertedOtherRegionsPrice"); other != nil {
				fmt.Fprintf(w, "%s\t%s\t%s\n", "(other regions, USD)", formatMoney(other.Doc("usdPrice")), "-")
				fmt.Fprintf(w, "%s\t%s\t%s\n", "(other regions, EUR)", formatMoney(other.Doc("eurPrice")), "-")
			}
			return w.Flush()
		})
	},
}

// parseMoney converts "USD:1.99" into the units/nanos Money representation the
// monetization endpoints use (unlike inappproducts, which uses micros).
func parseMoney(s string) (map[string]any, error) {
	currency, amount, ok := strings.Cut(s, ":")
	if !ok {
		return nil, fmt.Errorf("invalid price %q: use CURRENCY:AMOUNT, e.g. USD:4.99", s)
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price %q: %w", s, err)
	}
	units := math.Trunc(value)
	nanos := int64(math.Round((value - units) * 1e9))
	return map[string]any{
		"currencyCode": strings.ToUpper(strings.TrimSpace(currency)),
		"units":        strconv.FormatInt(int64(units), 10),
		"nanos":        nanos,
	}, nil
}

// formatMoney renders a Money resource for tables.
func formatMoney(money api.Doc) string {
	if money == nil {
		return "-"
	}
	units, _ := strconv.ParseInt(money.Str("units"), 10, 64)
	value := float64(units) + float64(money.Int("nanos"))/1e9
	if value == math.Trunc(value) {
		return fmt.Sprintf("%s %d", money.Str("currencyCode"), int64(value))
	}
	return fmt.Sprintf("%s %.2f", money.Str("currencyCode"), value)
}

func init() {
	pricingConvertCmd.Flags().StringVar(&pricingPrice, "price", "", "price to convert as CURRENCY:AMOUNT, e.g. USD:4.99 (required)")
	pricingConvertCmd.Flags().StringVar(&pricingTaxCategory, "tax-category", "", "product tax category code to apply")
	_ = pricingConvertCmd.MarkFlagRequired("price")

	pricingCmd.AddCommand(pricingConvertCmd)
	rootCmd.AddCommand(pricingCmd)
}
