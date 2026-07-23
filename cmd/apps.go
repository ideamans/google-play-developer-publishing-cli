package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
	"github.com/ideamans/google-play-developer-publishing-cli/internal/auth"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Apps the service account can reach",
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the apps the service account has access to",
	Long: `The Android Publisher API has no endpoint that enumerates apps: every call
takes a package name. This command therefore queries the Play Developer
Reporting API instead, which does expose the list.

That means it needs two extra things compared with the other commands:
the "Google Play Developer Reporting API" enabled in the key's Google Cloud
project, and the playdeveloperreporting scope. If either is missing the command
explains what to enable; the rest of gplay keeps working regardless.`,
	Example: `  gplay apps list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		var apps []api.Doc
		pageToken := ""
		for {
			url := api.ReportingBaseURL + "/v1beta1/apps:search?pageSize=50"
			if pageToken != "" {
				url += "&pageToken=" + pageToken
			}
			data, err := c.DoScoped(cmd.Context(), http.MethodGet, url, auth.ScopePlayDeveloperReporting, nil)
			if err != nil {
				return reportingHint(err)
			}
			var page struct {
				Apps          []api.Doc `json:"apps"`
				NextPageToken string    `json:"nextPageToken"`
			}
			if err := json.Unmarshal(data, &page); err != nil {
				return err
			}
			apps = append(apps, page.Apps...)
			if page.NextPageToken == "" {
				break
			}
			pageToken = page.NextPageToken
		}
		if jsonFlag {
			return printJSON(apps)
		}
		if len(apps) == 0 {
			fmt.Println("No apps. The service account may not be linked to any app in Play Console yet.")
			return nil
		}
		w := newTable("PACKAGE NAME", "DISPLAY NAME")
		for _, a := range apps {
			pkg := a.Str("packageName")
			if pkg == "" {
				pkg = strings.TrimPrefix(a.Str("name"), "apps/")
			}
			fmt.Fprintf(w, "%s\t%s\n", pkg, dash(a.Str("displayName")))
		}
		return w.Flush()
	},
}

func reportingHint(err error) error {
	return fmt.Errorf("%w\n\n"+
		"This command uses the Play Developer Reporting API because Android Publisher cannot list apps.\n"+
		"Enable it once at https://console.cloud.google.com/apis/library/playdeveloperreporting.googleapis.com\n"+
		"for the service account's project. Everything else in gplay works without it — pass --package instead.", err)
}

func init() {
	appsCmd.AddCommand(appsListCmd)
	rootCmd.AddCommand(appsCmd)
}
