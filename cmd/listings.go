package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	listingLanguage         string
	listingTitle            string
	listingShortDescription string
	listingFullDescription  string
	listingVideo            string
	listingFromJSON         string
	listingReplace          bool
)

var listingsCmd = &cobra.Command{
	Use:   "listings",
	Short: "Store listing text per language (title, descriptions, promo video)",
}

var listingsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List every localized store listing",
	Example: `  gplay listings list --package com.example.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/listings"))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			listings := doc.Docs("listings")
			if len(listings) == 0 {
				fmt.Println("No store listings.")
				return nil
			}
			w := newTable("LANGUAGE", "TITLE", "SHORT DESCRIPTION", "FULL DESC (chars)", "VIDEO")
			for _, l := range listings {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					l.Str("language"), l.Str("title"), truncate(l.Str("shortDescription"), 40),
					len([]rune(l.Str("fullDescription"))), dash(l.Str("video")))
			}
			return w.Flush()
		})
	},
}

var listingsGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one localized store listing",
	Example: `  gplay listings get --language ja`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/listings/%s", esc(listingLanguage)))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var listingsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Create or update a localized store listing",
	Long: `Updates the fields you pass and leaves the rest untouched (PATCH). Pass
--replace to send a full update (PUT) instead, which clears fields you omit.

Text flags accept @file to read the value from a file, which is the practical
way to supply the 4000-character full description.

Google Play limits: title 30 characters, short description 80, full description
4000.`,
	Example: `  gplay listings set --language ja --title "レシート読取" --short-description "..." --full-description @desc-ja.txt
  gplay listings set --language en-US --video https://www.youtube.com/watch?v=XXXXXXXXXXX
  gplay listings set --language ja --from-json @listing-ja.json --replace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			body := map[string]any{"language": listingLanguage}
			if listingFromJSON != "" {
				if err := jsonFromFlag(listingFromJSON, &body); err != nil {
					return err
				}
				body["language"] = listingLanguage
			}
			for flag, target := range map[string]*string{
				"title":             &listingTitle,
				"short-description": &listingShortDescription,
				"full-description":  &listingFullDescription,
				"video":             &listingVideo,
			} {
				if !cmd.Flags().Changed(flag) {
					continue
				}
				value, err := valueOrFile(*target)
				if err != nil {
					return err
				}
				body[listingField(flag)] = value
			}
			if len(body) == 1 {
				return fmt.Errorf("nothing to set: pass --title / --short-description / --full-description / --video / --from-json")
			}
			if err := checkListingLimits(body); err != nil {
				return err
			}

			path := e.path("/listings/%s", esc(listingLanguage))
			var err error
			if listingReplace {
				_, err = e.c.Put(ctx, path, body)
			} else {
				_, err = e.c.Patch(ctx, path, body)
			}
			if err != nil {
				return err
			}
			fmt.Printf("Listing %s updated.\n", listingLanguage)
			return nil
		})
	},
}

var listingsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete one localized store listing",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			if err := e.c.Delete(ctx, e.path("/listings/%s", esc(listingLanguage))); err != nil {
				return err
			}
			fmt.Printf("Listing %s deleted.\n", listingLanguage)
			return nil
		})
	},
}

var listingsDeleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: "Delete every localized store listing",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			if err := e.c.Delete(ctx, e.path("/listings")); err != nil {
				return err
			}
			fmt.Println("All store listings deleted.")
			return nil
		})
	},
}

// --- App details --------------------------------------------------------------

var (
	detailsContactEmail   string
	detailsContactPhone   string
	detailsContactWebsite string
	detailsDefaultLang    string
)

var detailsCmd = &cobra.Command{
	Use:   "details",
	Short: "App-level details (support contacts, default language)",
}

var detailsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the app details",
	Long: `Shows contact email, phone, website and default language.

This is also the cheapest way to verify that credentials and permissions work:
it opens an edit, reads the details and discards the edit.`,
	Example: `  gplay details get --package com.example.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/details"))
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var detailsSetCmd = &cobra.Command{
	Use:     "set",
	Short:   "Update the app details",
	Example: `  gplay details set --contact-email support@example.com --contact-website https://example.com/support`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			body := map[string]any{}
			if cmd.Flags().Changed("contact-email") {
				body["contactEmail"] = detailsContactEmail
			}
			if cmd.Flags().Changed("contact-phone") {
				body["contactPhone"] = detailsContactPhone
			}
			if cmd.Flags().Changed("contact-website") {
				body["contactWebsite"] = detailsContactWebsite
			}
			if cmd.Flags().Changed("default-language") {
				body["defaultLanguage"] = detailsDefaultLang
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to set: pass --contact-email / --contact-phone / --contact-website / --default-language")
			}
			if _, err := e.c.Patch(ctx, e.path("/details"), body); err != nil {
				return err
			}
			fmt.Println("App details updated.")
			return nil
		})
	},
}

func listingField(flag string) string {
	switch flag {
	case "title":
		return "title"
	case "short-description":
		return "shortDescription"
	case "full-description":
		return "fullDescription"
	default:
		return "video"
	}
}

// checkListingLimits rejects values Google Play would reject anyway, before an
// edit is opened and a round trip is spent.
func checkListingLimits(body map[string]any) error {
	limits := map[string]int{"title": 30, "shortDescription": 80, "fullDescription": 4000}
	for field, max := range limits {
		value, _ := body[field].(string)
		if n := len([]rune(value)); n > max {
			return fmt.Errorf("%s is %d characters; Google Play allows at most %d", field, n, max)
		}
	}
	return nil
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return dash(s)
	}
	return string(r[:max-1]) + "…"
}

func init() {
	for _, sub := range []*cobra.Command{listingsGetCmd, listingsSetCmd, listingsDeleteCmd} {
		sub.Flags().StringVarP(&listingLanguage, "language", "l", "", "BCP-47 language code, e.g. ja or en-US (required)")
		_ = sub.MarkFlagRequired("language")
	}
	listingsSetCmd.Flags().StringVar(&listingTitle, "title", "", "app title (max 30 chars), or @file")
	listingsSetCmd.Flags().StringVar(&listingShortDescription, "short-description", "", "short description (max 80 chars), or @file")
	listingsSetCmd.Flags().StringVar(&listingFullDescription, "full-description", "", "full description (max 4000 chars), or @file")
	listingsSetCmd.Flags().StringVar(&listingVideo, "video", "", "promotional YouTube URL")
	listingsSetCmd.Flags().StringVar(&listingFromJSON, "from-json", "", "listing resource as JSON, or @file")
	listingsSetCmd.Flags().BoolVar(&listingReplace, "replace", false, "send a full update (PUT), clearing omitted fields")

	detailsSetCmd.Flags().StringVar(&detailsContactEmail, "contact-email", "", "user-visible support email")
	detailsSetCmd.Flags().StringVar(&detailsContactPhone, "contact-phone", "", "user-visible support phone number")
	detailsSetCmd.Flags().StringVar(&detailsContactWebsite, "contact-website", "", "user-visible support website")
	detailsSetCmd.Flags().StringVar(&detailsDefaultLang, "default-language", "", "default language, e.g. ja")

	addEditReadFlags(listingsListCmd, listingsGetCmd, detailsGetCmd)
	addEditFlags(listingsSetCmd, listingsDeleteCmd, listingsDeleteAllCmd, detailsSetCmd)

	listingsCmd.AddCommand(listingsListCmd, listingsGetCmd, listingsSetCmd, listingsDeleteCmd, listingsDeleteAllCmd)
	detailsCmd.AddCommand(detailsGetCmd, detailsSetCmd)
	rootCmd.AddCommand(listingsCmd, detailsCmd)
}
