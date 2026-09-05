package manga

import (
	"fmt"
	"image"
	_ "image/gif"  // register GIF decoding for image.DecodeConfig
	_ "image/jpeg" // register JPEG decoding for image.DecodeConfig
	_ "image/png"  // register PNG decoding for image.DecodeConfig
	"os"
	"path/filepath"

	"github.com/signintech/gopdf"
)

// BuildPDF bundles pageFiles (in order) into a single PDF at outPath, one
// page per image, each page sized to match that image's own pixel
// dimensions (1px = 1pt) so pages aren't stretched or letterboxed.
func BuildPDF(pageFiles []string, outPath string) error {
	if len(pageFiles) == 0 {
		return fmt.Errorf("no pages to bundle")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	for i, pagePath := range pageFiles {
		width, height, err := imageDimensions(pagePath)
		if err != nil {
			return fmt.Errorf("failed to read page %d: %w", i+1, err)
		}

		pageSize := &gopdf.Rect{W: width, H: height}
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: pageSize})
		if err := pdf.Image(pagePath, 0, 0, pageSize); err != nil {
			return fmt.Errorf("failed to add page %d: %w", i+1, err)
		}
	}

	if err := pdf.WritePdf(outPath); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}
	return nil
}

// imageDimensions reads just the header of a local image file to get its
// pixel dimensions, without decoding the full image.
func imageDimensions(path string) (width, height float64, err error) {
	f, err := os.Open(path) // #nosec G304 -- path comes from files this package itself just downloaded
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = f.Close() }()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return float64(cfg.Width), float64(cfg.Height), nil
}
