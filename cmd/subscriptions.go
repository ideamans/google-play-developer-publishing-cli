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
	subProduct        string
	subProducts       []string
	subBasePlan       string
	subOffer          string
	subFromJSON       string
	subUpdateMask     string
	subShowArchived   bool
	subRegionsVersion string
	subLanguage       string
	subTitle          string
	subDescription    string
	subBenefits       []string
	subLatency        string
	subAllowMissing   bool
)

// defaultRegionsVersion is the region catalogue version Google documents for
// subscription writes; without it the API rejects regional price changes.
const defaultRegionsVersion = "2022/02"

var subscriptionsCmd = &cobra.Command{
	Use:     "subscriptions",
	Aliases: []string{"subs"},
	Short:   "Subscriptions, base plans and offers (monetization API)",
	Long: `Manages the modern subscription model: a subscription holds localized
listings and one or more base plans, and each base plan can carry offers.

Base plans and their regional prices are nested inside the subscription
resource, so they are edited through "subscriptions update" with an
--update-mask (for example --update-mask basePlans). The base-plans and offers
subcommands cover the operations that have dedicated endpoints: activate,
deactivate, delete and price migration.

Subscriptions are not edit-scoped — changes take effect immediately.`,
}

var subsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List subscriptions",
	Example: `  gplay subscriptions list --package com.example.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			path := appPath(pkg, "/subscriptions?pageSize=100")
			if subShowArchived {
				path += "&showArchived=true"
			}
			items, err := c.ListAll(ctx, path, "subscriptions")
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No subscriptions.")
				return nil
			}
			w := newTable("PRODUCT ID", "ARCHIVED", "BASE PLANS", "TITLE")
			for _, s := range items {
				plans := []string{}
				for _, bp := range s.Docs("basePlans") {
					plans = append(plans, fmt.Sprintf("%s(%s)", bp.Str("basePlanId"), bp.Str("state")))
				}
				title := ""
				if listings := s.Docs("listings"); len(listings) > 0 {
					title = listings[0].Str("title")
				}
				fmt.Fprintf(w, "%s\t%t\t%s\t%s\n", s.Str("productId"), s.Bool("archived"), commaJoin(plans), dash(title))
			}
			return w.Flush()
		})
	},
}

var subsGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one subscription",
	Example: `  gplay subscriptions get --product premium_monthly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/subscriptions/%s", esc(subProduct)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var subsBatchGetCmd = &cobra.Command{
	Use:     "batch-get",
	Short:   "Fetch several subscriptions in one request",
	Example: `  gplay subscriptions batch-get --product monthly --product yearly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{}
			for _, id := range subProducts {
				params.Add("productIds", id)
			}
			doc, err := c.GetMap(ctx, appPath(pkg, "/subscriptions:batchGet?%s", params.Encode()))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var subsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a subscription",
	Long: `Creates a subscription from a Subscription resource. Base plans, prices and
regional availability are nested in that resource, so --from-json is the
practical way to supply them; --title/--description/--benefit fill in a single
listing for simple cases.`,
	Example: `  gplay subscriptions create --product premium_monthly --from-json @subscription.json
  gplay subscriptions create --product premium_monthly --language ja --title "プレミアム" --description "広告非表示"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body, err := buildSubscription(cmd, pkg, nil)
			if err != nil {
				return err
			}
			params := url.Values{"productId": {subProduct}, "regionsVersion.version": {subRegionsVersion}}
			doc, err := c.Post(ctx, appPath(pkg, "/subscriptions?%s", params.Encode()), body)
			if err != nil {
				return err
			}
			fmt.Printf("Subscription %s created.\n", subProduct)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var subsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a subscription",
	Long: `Patches a subscription. The API requires an update mask naming the top-level
fields to replace, e.g. --update-mask listings or --update-mask basePlans.`,
	Example: `  gplay subscriptions update --product premium_monthly --update-mask basePlans --from-json @subscription.json
  gplay subscriptions update --product premium_monthly --update-mask listings --language ja --title "プレミアム"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var current api.Doc
			if len(subBenefits) > 0 || subTitle != "" || subDescription != "" {
				var err error
				current, err = c.GetMap(ctx, appPath(pkg, "/subscriptions/%s", esc(subProduct)))
				if err != nil {
					return err
				}
			}
			body, err := buildSubscription(cmd, pkg, current)
			if err != nil {
				return err
			}
			mask := subUpdateMask
			if mask == "" {
				mask = strings.Join(topLevelKeys(body, "packageName", "productId"), ",")
			}
			if mask == "" {
				return fmt.Errorf("nothing to update: pass --from-json or the listing flags")
			}
			params := url.Values{
				"updateMask":             {mask},
				"regionsVersion.version": {subRegionsVersion},
			}
			if subAllowMissing {
				params.Set("allowMissing", "true")
			}
			if subLatency != "" {
				params.Set("latencyTolerance", subLatency)
			}
			if _, err := c.Patch(ctx, appPath(pkg, "/subscriptions/%s?%s", esc(subProduct), params.Encode()), body); err != nil {
				return err
			}
			fmt.Printf("Subscription %s updated (mask: %s).\n", subProduct, mask)
			return nil
		})
	},
}

var subsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a draft subscription",
	Long:  `Only subscriptions that have never been activated can be deleted; archive the others.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if err := c.Delete(ctx, appPath(pkg, "/subscriptions/%s", esc(subProduct))); err != nil {
				return err
			}
			fmt.Printf("Subscription %s deleted.\n", subProduct)
			return nil
		})
	},
}

var subsArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive a subscription",
	Long:  `Archiving hides a subscription from new purchases; existing subscribers keep it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if _, err := c.Post(ctx, appPath(pkg, "/subscriptions/%s:archive", esc(subProduct)), map[string]any{}); err != nil {
				return err
			}
			fmt.Printf("Subscription %s archived.\n", subProduct)
			return nil
		})
	},
}

// --- Base plans ------------------------------------------------------------------

var basePlansCmd = &cobra.Command{
	Use:   "base-plans",
	Short: "Base plans of a subscription",
}

var basePlansListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List a subscription's base plans",
	Example: `  gplay subscriptions base-plans list --product premium_monthly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, appPath(pkg, "/subscriptions/%s", esc(subProduct)))
			if err != nil {
				return err
			}
			plans := doc.Docs("basePlans")
			if jsonFlag {
				return printJSON(plans)
			}
			if len(plans) == 0 {
				fmt.Println("No base plans.")
				return nil
			}
			w := newTable("BASE PLAN", "STATE", "TYPE", "PERIOD", "REGIONS")
			for _, bp := range plans {
				kind, period := basePlanKind(bp)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
					bp.Str("basePlanId"), bp.Str("state"), kind, dash(period), len(bp.Docs("regionalConfigs")))
			}
			return w.Flush()
		})
	},
}

var basePlansActivateCmd = &cobra.Command{
	Use:     "activate",
	Short:   "Activate a base plan",
	Example: `  gplay subscriptions base-plans activate --product premium_monthly --base-plan monthly`,
	RunE:    basePlanState("activate"),
}

var basePlansDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate a base plan (existing subscribers keep it)",
	RunE:  basePlanState("deactivate"),
}

var basePlansDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a draft base plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if err := c.Delete(ctx, appPath(pkg, "/subscriptions/%s/basePlans/%s", esc(subProduct), esc(subBasePlan))); err != nil {
				return err
			}
			fmt.Printf("Base plan %s deleted.\n", subBasePlan)
			return nil
		})
	},
}

var basePlansMigrateCmd = &cobra.Command{
	Use:   "migrate-prices",
	Short: "Migrate existing subscribers to the current base plan prices",
	Long: `Applies the base plan's current regional prices to subscribers who are still
on an older price. The request body names the regions and the oldest price
version to migrate; see MigrateBasePlanPricesRequest.`,
	Example: `  gplay subscriptions base-plans migrate-prices --product premium_monthly --base-plan monthly --from-json @migrate.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{}
			if subFromJSON != "" {
				if err := jsonFromFlag(subFromJSON, &body); err != nil {
					return err
				}
			}
			body["packageName"] = pkg
			body["productId"] = subProduct
			body["basePlanId"] = subBasePlan
			doc, err := c.Post(ctx, appPath(pkg, "/subscriptions/%s/basePlans/%s:migratePrices",
				esc(subProduct), esc(subBasePlan)), body)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

// --- Offers -----------------------------------------------------------------------

var offersCmd = &cobra.Command{
	Use:   "offers",
	Short: "Introductory and promotional offers on a base plan",
}

var offersListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the offers of a base plan",
	Example: `  gplay subscriptions offers list --product premium_monthly --base-plan monthly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			items, err := c.ListAll(ctx, appPath(pkg, "/subscriptions/%s/basePlans/%s/offers?pageSize=100",
				esc(subProduct), esc(subBasePlan)), "subscriptionOffers")
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
			w := newTable("OFFER", "STATE", "PHASES", "REGIONS", "TAGS")
			for _, o := range items {
				tags := []string{}
				for _, t := range o.Docs("offerTags") {
					tags = append(tags, t.Str("tag"))
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", o.Str("offerId"), o.Str("state"),
					len(o.Docs("phases")), len(o.Docs("regionalConfigs")), commaJoin(tags))
			}
			return w.Flush()
		})
	},
}

var offersGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show one offer",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			doc, err := c.GetMap(ctx, offerPath(pkg, ""))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var offersCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create an offer on a base plan",
	Example: `  gplay subscriptions offers create --product premium_monthly --base-plan monthly --offer intro-7d --from-json @offer.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(subFromJSON, &body); err != nil {
				return err
			}
			body["packageName"] = pkg
			body["productId"] = subProduct
			body["basePlanId"] = subBasePlan
			body["offerId"] = subOffer
			params := url.Values{"offerId": {subOffer}, "regionsVersion.version": {subRegionsVersion}}
			doc, err := c.Post(ctx, appPath(pkg, "/subscriptions/%s/basePlans/%s/offers?%s",
				esc(subProduct), esc(subBasePlan), params.Encode()), body)
			if err != nil {
				return err
			}
			fmt.Printf("Offer %s created.\n", subOffer)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

var offersUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an offer",
	Example: `  gplay subscriptions offers update --product p --base-plan monthly --offer intro-7d --update-mask phases --from-json @offer.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			var body map[string]any
			if err := jsonFromFlag(subFromJSON, &body); err != nil {
				return err
			}
			mask := subUpdateMask
			if mask == "" {
				mask = strings.Join(topLevelKeys(body, "packageName", "productId", "basePlanId", "offerId"), ",")
			}
			params := url.Values{"updateMask": {mask}, "regionsVersion.version": {subRegionsVersion}}
			if _, err := c.Patch(ctx, offerPath(pkg, "?"+params.Encode()), body); err != nil {
				return err
			}
			fmt.Printf("Offer %s updated (mask: %s).\n", subOffer, mask)
			return nil
		})
	},
}

var offersActivateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate an offer",
	RunE:  offerState("activate"),
}

var offersDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate an offer",
	RunE:  offerState("deactivate"),
}

var offersDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a draft offer",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			if err := c.Delete(ctx, offerPath(pkg, "")); err != nil {
				return err
			}
			fmt.Printf("Offer %s deleted.\n", subOffer)
			return nil
		})
	},
}

// --- Helpers ------------------------------------------------------------------------

func offerPath(pkg, suffix string) string {
	return appPath(pkg, "/subscriptions/%s/basePlans/%s/offers/%s%s",
		esc(subProduct), esc(subBasePlan), esc(subOffer), suffix)
}

func basePlanState(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{"packageName": pkg, "productId": subProduct, "basePlanId": subBasePlan}
			path := appPath(pkg, "/subscriptions/%s/basePlans/%s:%s", esc(subProduct), esc(subBasePlan), action)
			if _, err := c.Post(ctx, path, body); err != nil {
				return err
			}
			fmt.Printf("Base plan %s %sd.\n", subBasePlan, strings.TrimSuffix(action, "e"))
			return nil
		})
	}
}

func offerState(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			body := map[string]any{
				"packageName": pkg, "productId": subProduct,
				"basePlanId": subBasePlan, "offerId": subOffer,
			}
			if _, err := c.Post(ctx, offerPath(pkg, ":"+action), body); err != nil {
				return err
			}
			fmt.Printf("Offer %s %sd.\n", subOffer, strings.TrimSuffix(action, "e"))
			return nil
		})
	}
}

// buildSubscription assembles a Subscription body. current, when given, is the
// existing resource whose listings are merged with the listing flags.
func buildSubscription(cmd *cobra.Command, pkg string, current api.Doc) (map[string]any, error) {
	body := map[string]any{}
	if subFromJSON != "" {
		if err := jsonFromFlag(subFromJSON, &body); err != nil {
			return nil, err
		}
	}
	body["packageName"] = pkg
	body["productId"] = subProduct

	if subTitle != "" || subDescription != "" || len(subBenefits) > 0 {
		if subLanguage == "" {
			return nil, fmt.Errorf("--language is required with --title / --description / --benefit")
		}
		if subDescription != "" {
			resolved, err := valueOrFile(subDescription)
			if err != nil {
				return nil, err
			}
			subDescription = resolved
		}
		listings := []any{}
		found := false
		for _, l := range current.Docs("listings") {
			listing := map[string]any(l)
			if l.Str("languageCode") == subLanguage {
				found = true
				applyListing(listing)
			}
			listings = append(listings, listing)
		}
		if !found {
			listing := map[string]any{"languageCode": subLanguage}
			applyListing(listing)
			listings = append(listings, listing)
		}
		body["listings"] = listings
	}
	return body, nil
}

func applyListing(listing map[string]any) {
	if subTitle != "" {
		listing["title"] = subTitle
	}
	if subDescription != "" {
		listing["description"] = subDescription
	}
	if len(subBenefits) > 0 {
		listing["benefits"] = subBenefits
	}
}

// topLevelKeys returns the field names of a body, minus the identifying ones,
// so an update mask can be inferred when the caller does not pass one.
func topLevelKeys(body map[string]any, skip ...string) []string {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
	out := []string{}
	for key := range body {
		if !skipped[key] {
			out = append(out, key)
		}
	}
	return out
}

func basePlanKind(bp api.Doc) (string, string) {
	switch {
	case bp.Has("autoRenewingBasePlanType"):
		return "auto-renewing", bp.Doc("autoRenewingBasePlanType").Str("billingPeriodDuration")
	case bp.Has("prepaidBasePlanType"):
		return "prepaid", bp.Doc("prepaidBasePlanType").Str("billingPeriodDuration")
	case bp.Has("installmentsBasePlanType"):
		return "installments", bp.Doc("installmentsBasePlanType").Str("billingPeriodDuration")
	default:
		return "-", ""
	}
}

func init() {
	productCmds := []*cobra.Command{
		subsGetCmd, subsCreateCmd, subsUpdateCmd, subsDeleteCmd, subsArchiveCmd,
		basePlansListCmd, basePlansActivateCmd, basePlansDeactivateCmd, basePlansDeleteCmd, basePlansMigrateCmd,
		offersListCmd, offersGetCmd, offersCreateCmd, offersUpdateCmd, offersActivateCmd, offersDeactivateCmd, offersDeleteCmd,
	}
	for _, sub := range productCmds {
		sub.Flags().StringVar(&subProduct, "product", "", "subscription product id (required)")
		_ = sub.MarkFlagRequired("product")
	}
	basePlanCmds := []*cobra.Command{
		basePlansActivateCmd, basePlansDeactivateCmd, basePlansDeleteCmd, basePlansMigrateCmd,
		offersListCmd, offersGetCmd, offersCreateCmd, offersUpdateCmd, offersActivateCmd, offersDeactivateCmd, offersDeleteCmd,
	}
	for _, sub := range basePlanCmds {
		sub.Flags().StringVar(&subBasePlan, "base-plan", "", "base plan id (required)")
		_ = sub.MarkFlagRequired("base-plan")
	}
	for _, sub := range []*cobra.Command{offersGetCmd, offersCreateCmd, offersUpdateCmd, offersActivateCmd, offersDeactivateCmd, offersDeleteCmd} {
		sub.Flags().StringVar(&subOffer, "offer", "", "offer id (required)")
		_ = sub.MarkFlagRequired("offer")
	}
	for _, sub := range []*cobra.Command{subsCreateCmd, subsUpdateCmd, basePlansMigrateCmd, offersCreateCmd, offersUpdateCmd} {
		sub.Flags().StringVar(&subFromJSON, "from-json", "", "resource JSON, or @file")
		sub.Flags().StringVar(&subRegionsVersion, "regions-version", defaultRegionsVersion, "region catalogue version for price changes")
	}
	for _, sub := range []*cobra.Command{subsUpdateCmd, offersUpdateCmd} {
		sub.Flags().StringVar(&subUpdateMask, "update-mask", "", "comma-separated fields to replace (inferred from the body when omitted)")
	}
	for _, sub := range []*cobra.Command{subsCreateCmd, subsUpdateCmd} {
		sub.Flags().StringVarP(&subLanguage, "language", "l", "", "listing language, e.g. ja")
		sub.Flags().StringVar(&subTitle, "title", "", "listing title")
		sub.Flags().StringVar(&subDescription, "description", "", "listing description, or @file")
		sub.Flags().StringArrayVar(&subBenefits, "benefit", nil, "listing benefit line; repeatable")
	}
	subsUpdateCmd.Flags().BoolVar(&subAllowMissing, "allow-missing", false, "create the subscription when it does not exist")
	subsUpdateCmd.Flags().StringVar(&subLatency, "latency-tolerance", "", "PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT for bulk updates")
	subsListCmd.Flags().BoolVar(&subShowArchived, "show-archived", false, "include archived subscriptions")
	subsBatchGetCmd.Flags().StringArrayVar(&subProducts, "product", nil, "subscription product id; repeatable (required)")
	_ = subsBatchGetCmd.MarkFlagRequired("product")

	basePlansCmd.AddCommand(basePlansListCmd, basePlansActivateCmd, basePlansDeactivateCmd, basePlansDeleteCmd, basePlansMigrateCmd)
	offersCmd.AddCommand(offersListCmd, offersGetCmd, offersCreateCmd, offersUpdateCmd, offersActivateCmd, offersDeactivateCmd, offersDeleteCmd)
	subscriptionsCmd.AddCommand(subsListCmd, subsGetCmd, subsBatchGetCmd, subsCreateCmd, subsUpdateCmd,
		subsDeleteCmd, subsArchiveCmd, basePlansCmd, offersCmd)
	rootCmd.AddCommand(subscriptionsCmd)
}
