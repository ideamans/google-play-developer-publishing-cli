package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/auth"
)

var tokenScope string

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print an OAuth 2.0 access token for use with curl etc.",
	Long: `Exchanges the profile's service account key for a short-lived OAuth 2.0
access token (valid for about an hour) and prints it.`,
	Example: `  curl -H "Authorization: Bearer $(gplay token)" \
    https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.example.app/edits -X POST`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		token, err := c.Token(cmd.Context(), tokenScope)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	},
}

func init() {
	tokenCmd.Flags().StringVar(&tokenScope, "scope", auth.ScopeAndroidPublisher, "OAuth scope to request")
	rootCmd.AddCommand(tokenCmd)
}
