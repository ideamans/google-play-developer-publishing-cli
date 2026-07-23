package cmd

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	imageLanguage   string
	imageType       string
	imageFiles      []string
	imageID         string
	imageAI         bool
	imageSkipChecks bool
)

// imageTypes are the appImageType values the API accepts, with the dimensions
// Google Play enforces (0 means "no fixed size").
var imageTypes = map[string]struct {
	width, height int
	note          string
}{
	"icon":                 {512, 512, "32-bit PNG, 512x512"},
	"featureGraphic":       {1024, 500, "PNG or JPEG, 1024x500"},
	"tvBanner":             {1280, 720, "PNG or JPEG, 1280x720"},
	"phoneScreenshots":     {0, 0, "2-8 images, each side 320-3840 px"},
	"sevenInchScreenshots": {0, 0, "up to 8 images, each side 320-3840 px"},
	"tenInchScreenshots":   {0, 0, "up to 8 images, each side 320-3840 px"},
	"tvScreenshots":        {0, 0, "up to 8 images, each side 320-3840 px"},
	"wearScreenshots":      {0, 0, "up to 8 images, each side 320-3840 px"},
}

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Store listing graphics (icon, feature graphic, screenshots)",
	Long: `Uploads and manages the graphic assets of a localized store listing.

Image types: ` + imageTypeList() + `

Requirements enforced before upload (bypass with --skip-validation):
  icon            32-bit PNG, exactly 512x512
  featureGraphic  PNG or JPEG, exactly 1024x500
  tvBanner        PNG or JPEG, exactly 1280x720
  *Screenshots    each side between 320 and 3840 px
  every image     at most 15 MiB (API limit)`,
}

var imagesListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List uploaded images of one type",
	Example: `  gplay images list --language ja --type phoneScreenshots`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEditRead(cmd, func(ctx context.Context, e *edit) error {
			doc, err := e.c.GetMap(ctx, e.path("/listings/%s/%s", esc(imageLanguage), esc(imageType)))
			if err != nil {
				return err
			}
			if jsonFlag {
				return printJSON(doc)
			}
			images := doc.Docs("images")
			if len(images) == 0 {
				fmt.Printf("No %s for %s.\n", imageType, imageLanguage)
				return nil
			}
			w := newTable("ID", "SHA256", "URL")
			for _, img := range images {
				fmt.Fprintf(w, "%s\t%s\t%s\n", img.Str("id"), truncate(img.Str("sha256"), 16), img.Str("url"))
			}
			return w.Flush()
		})
	},
}

var imagesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload one or more images",
	Long: `Adds images to a listing. Screenshots accumulate in upload order, so use
"gplay images replace" when you want the listing to contain exactly the files
you pass. Single-image types (icon, featureGraphic, tvBanner) are replaced by
the upload.`,
	Example: `  gplay images upload --language ja --type icon --file icon-512.png
  gplay images upload --language ja --type phoneScreenshots --file 01.png --file 02.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			return uploadImages(ctx, e)
		})
	},
}

var imagesReplaceCmd = &cobra.Command{
	Use:   "replace",
	Short: "Delete every image of a type, then upload the given files",
	Long: `Deletes all existing images of the type and uploads the files in the order
given, so the listing ends up with exactly those images. Both steps happen in
one edit, so the listing is never left empty.`,
	Example: `  gplay images replace --language ja --type phoneScreenshots --file 01.png --file 02.png --file 03.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			if err := e.c.Delete(ctx, e.path("/listings/%s/%s", esc(imageLanguage), esc(imageType))); err != nil {
				return err
			}
			fmt.Printf("Deleted existing %s for %s.\n", imageType, imageLanguage)
			return uploadImages(ctx, e)
		})
	},
}

var imagesDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete one image by id",
	Example: `  gplay images delete --language ja --type phoneScreenshots --id AbCdEf123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			if err := e.c.Delete(ctx, e.path("/listings/%s/%s/%s", esc(imageLanguage), esc(imageType), esc(imageID))); err != nil {
				return err
			}
			fmt.Printf("Image %s deleted.\n", imageID)
			return nil
		})
	},
}

var imagesDeleteAllCmd = &cobra.Command{
	Use:     "delete-all",
	Short:   "Delete every image of one type",
	Example: `  gplay images delete-all --language ja --type phoneScreenshots`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd, func(ctx context.Context, e *edit) error {
			if err := e.c.Delete(ctx, e.path("/listings/%s/%s", esc(imageLanguage), esc(imageType))); err != nil {
				return err
			}
			fmt.Printf("All %s for %s deleted.\n", imageType, imageLanguage)
			return nil
		})
	},
}

func uploadImages(ctx context.Context, e *edit) error {
	for _, file := range imageFiles {
		if !imageSkipChecks {
			if err := validateImage(file, imageType); err != nil {
				return err
			}
		}
		path := e.path("/listings/%s/%s", esc(imageLanguage), esc(imageType))
		if imageAI {
			path += "?aiGeneratedState=aiGeneratedStateAiGeneratedDeveloperAttested"
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		doc, err := e.c.UploadBytes(ctx, path, data, contentTypeFor(file))
		if err != nil {
			return fmt.Errorf("upload %s: %w", file, err)
		}
		fmt.Printf("Uploaded %s as %s (id %s).\n", file, imageType, dash(doc.Doc("image").Str("id")))
	}
	return nil
}

// validateImage rejects files Google Play would reject after the upload, when
// the failure is much harder to attribute.
func validateImage(path, kind string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > 15<<20 {
		return fmt.Errorf("%s is %.1f MiB; the API accepts at most 15 MiB", path, float64(info.Size())/(1<<20))
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("%s: %w (Google Play accepts PNG and JPEG)", path, err)
	}
	spec, ok := imageTypes[kind]
	if !ok {
		return fmt.Errorf("unknown image type %q; expected one of %s", kind, imageTypeList())
	}
	if kind == "icon" && format != "png" {
		return fmt.Errorf("%s is %s; the app icon must be a 32-bit PNG", path, format)
	}
	if spec.width > 0 {
		if cfg.Width != spec.width || cfg.Height != spec.height {
			return fmt.Errorf("%s is %dx%d; %s must be exactly %dx%d",
				path, cfg.Width, cfg.Height, kind, spec.width, spec.height)
		}
		return nil
	}
	for _, side := range []int{cfg.Width, cfg.Height} {
		if side < 320 || side > 3840 {
			return fmt.Errorf("%s is %dx%d; screenshot sides must be between 320 and 3840 px",
				path, cfg.Width, cfg.Height)
		}
	}
	return nil
}

func contentTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "image/png"
	}
}

func imageTypeList() string {
	names := make([]string, 0, len(imageTypes))
	for name := range imageTypes {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func init() {
	subs := []*cobra.Command{imagesListCmd, imagesUploadCmd, imagesReplaceCmd, imagesDeleteCmd, imagesDeleteAllCmd}
	for _, sub := range subs {
		sub.Flags().StringVarP(&imageLanguage, "language", "l", "", "BCP-47 language code, e.g. ja (required)")
		sub.Flags().StringVar(&imageType, "type", "", "image type: "+imageTypeList()+" (required)")
		_ = sub.MarkFlagRequired("language")
		_ = sub.MarkFlagRequired("type")
	}
	for _, sub := range []*cobra.Command{imagesUploadCmd, imagesReplaceCmd} {
		sub.Flags().StringArrayVar(&imageFiles, "file", nil, "image file; repeat for several images, in order (required)")
		sub.Flags().BoolVar(&imageAI, "ai-generated", false, "attest that the images were generated by AI")
		sub.Flags().BoolVar(&imageSkipChecks, "skip-validation", false, "upload without checking format and dimensions")
		_ = sub.MarkFlagRequired("file")
	}
	imagesDeleteCmd.Flags().StringVar(&imageID, "id", "", "image id from \"gplay images list\" (required)")
	_ = imagesDeleteCmd.MarkFlagRequired("id")

	addEditReadFlags(imagesListCmd)
	addEditFlags(imagesUploadCmd, imagesReplaceCmd, imagesDeleteCmd, imagesDeleteAllCmd)

	imagesCmd.AddCommand(subs...)
	rootCmd.AddCommand(imagesCmd)
}
