package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/api"
)

var (
	apiMethod   string
	apiData     string
	apiPaginate bool
	apiItemsKey string
)

var apiCmd = &cobra.Command{
	Use:   "api <path>",
	Short: "Send a raw request to the Android Publisher API",
	Long: `Sends an authenticated request to
https://androidpublisher.googleapis.com and prints the JSON response. Use this
for endpoints not yet covered by a dedicated subcommand.

The "/androidpublisher/v3" prefix is added when the path omits it, so
"/applications/com.example.app/reviews" and
"/androidpublisher/v3/applications/com.example.app/reviews" are equivalent.`,
	Example: `  gplay api /applications/com.example.app/reviews
  gplay api -X POST /applications/com.example.app/edits
  gplay api -X PATCH "/applications/com.example.app/edits/EDIT_ID/listings/ja" -d @listing.json
  gplay api --paginate --items reviews "/applications/com.example.app/reviews?maxResults=100"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		c, err := newClient()
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if apiPaginate {
			if apiItemsKey == "" {
				return fmt.Errorf("--paginate requires --items <field>, e.g. --items reviews")
			}
			items, err := c.ListAll(ctx, path, apiItemsKey)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{apiItemsKey: items})
		}

		var body io.Reader
		if apiData != "" {
			payload := []byte(apiData)
			if strings.HasPrefix(apiData, "@") {
				payload, err = os.ReadFile(apiData[1:])
				if err != nil {
					return err
				}
			}
			body = bytes.NewReader(payload)
		}

		method := strings.ToUpper(apiMethod)
		if dryRun && method != "GET" {
			fmt.Fprintf(os.Stderr, "DRY-RUN %s %s\n", method, api.URL(path))
			if apiData != "" {
				fmt.Fprintln(os.Stderr, apiData)
			}
			return nil
		}

		data, err := c.Do(ctx, method, path, body)
		if err != nil {
			return err
		}
		var pretty bytes.Buffer
		if json.Indent(&pretty, data, "", "  ") == nil {
			fmt.Println(pretty.String())
		} else if len(data) > 0 {
			fmt.Println(string(data))
		}
		return nil
	},
}

func init() {
	apiCmd.Flags().StringVarP(&apiMethod, "method", "X", "GET", "HTTP method")
	apiCmd.Flags().StringVarP(&apiData, "data", "d", "", "JSON request body, or @file to read from a file")
	apiCmd.Flags().BoolVar(&apiPaginate, "paginate", false, "follow pagination and merge every page")
	apiCmd.Flags().StringVar(&apiItemsKey, "items", "", "response field holding the page items (with --paginate)")
	rootCmd.AddCommand(apiCmd)
}
