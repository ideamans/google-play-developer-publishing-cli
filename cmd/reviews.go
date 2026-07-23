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
	reviewID          string
	reviewTranslateTo string
	reviewReplyText   string
	reviewLimit       int
)

var reviewsCmd = &cobra.Command{
	Use:   "reviews",
	Short: "User reviews and developer replies",
	Long: `Reads reviews that have a written comment and posts developer replies.

Google only exposes reviews from the last week or so through this API; use the
review export in Play Console for the full history.`,
}

var reviewsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent reviews",
	Example: `  gplay reviews list
  gplay reviews list --limit 20 --translate-to ja`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			params := url.Values{"maxResults": {"100"}}
			if reviewTranslateTo != "" {
				params.Set("translationLanguage", reviewTranslateTo)
			}
			items, err := c.ListAll(ctx, appPath(pkg, "/reviews?%s", params.Encode()), "reviews")
			if err != nil {
				return err
			}
			if reviewLimit > 0 && len(items) > reviewLimit {
				items = items[:reviewLimit]
			}
			if jsonFlag {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("No reviews in the window the API exposes.")
				return nil
			}
			w := newTable("REVIEW ID", "STARS", "AUTHOR", "VERSION", "REPLIED", "COMMENT")
			for _, r := range items {
				stars, version, text, replied := reviewFields(r)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\n",
					truncate(r.Str("reviewId"), 12), stars, dash(r.Str("authorName")), version, replied,
					truncate(strings.ReplaceAll(text, "\n", " "), 60))
			}
			return w.Flush()
		})
	},
}

var reviewsGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show one review with its full comment thread",
	Example: `  gplay reviews get --review <review-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			path := appPath(pkg, "/reviews/%s", esc(reviewID))
			if reviewTranslateTo != "" {
				path += "?translationLanguage=" + url.QueryEscape(reviewTranslateTo)
			}
			doc, err := c.GetMap(ctx, path)
			if err != nil {
				return err
			}
			return printJSON(doc)
		})
	},
}

var reviewsReplyCmd = &cobra.Command{
	Use:   "reply",
	Short: "Reply to a review",
	Long: `Posts (or replaces) the developer reply. A reply is limited to 350
characters and is public.`,
	Example: `  gplay reviews reply --review <review-id> --text "ご報告ありがとうございます。次のアップデートで修正します。"
  gplay reviews reply --review <review-id> --text @reply.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd, func(ctx context.Context, c *api.Client, pkg string) error {
			text, err := valueOrFile(reviewReplyText)
			if err != nil {
				return err
			}
			if n := len([]rune(text)); n > 350 {
				return fmt.Errorf("the reply is %d characters; Google Play allows at most 350", n)
			}
			doc, err := c.Post(ctx, appPath(pkg, "/reviews/%s:reply", esc(reviewID)),
				map[string]any{"replyText": text})
			if err != nil {
				return err
			}
			fmt.Printf("Replied to review %s.\n", reviewID)
			if jsonFlag {
				return printJSON(doc)
			}
			return nil
		})
	},
}

// reviewFields pulls the user comment and reply state out of a review's
// comment thread.
func reviewFields(r api.Doc) (stars, version, text string, replied bool) {
	for _, comment := range r.Docs("comments") {
		if user := comment.Doc("userComment"); user != nil {
			stars = strings.Repeat("★", int(user.Int("starRating")))
			version = user.Str("appVersionName")
			if version == "" {
				version = user.Str("appVersionCode")
			}
			text = user.Str("text")
		}
		if comment.Doc("developerComment") != nil {
			replied = true
		}
	}
	return stars, dash(version), text, replied
}

func init() {
	reviewsListCmd.Flags().IntVar(&reviewLimit, "limit", 0, "stop after this many reviews")
	for _, sub := range []*cobra.Command{reviewsListCmd, reviewsGetCmd} {
		sub.Flags().StringVar(&reviewTranslateTo, "translate-to", "", "translate reviews into this language, e.g. ja")
	}
	for _, sub := range []*cobra.Command{reviewsGetCmd, reviewsReplyCmd} {
		sub.Flags().StringVar(&reviewID, "review", "", "review id (required)")
		_ = sub.MarkFlagRequired("review")
	}
	reviewsReplyCmd.Flags().StringVar(&reviewReplyText, "text", "", "reply text (max 350 chars), or @file (required)")
	_ = reviewsReplyCmd.MarkFlagRequired("text")

	reviewsCmd.AddCommand(reviewsListCmd, reviewsGetCmd, reviewsReplyCmd)
	rootCmd.AddCommand(reviewsCmd)
}
