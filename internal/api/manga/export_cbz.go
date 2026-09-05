package manga

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BuildCBZ bundles pageFiles (in order) into a CBZ (a plain zip archive,
// the de facto standard for comic/manga readers) at outPath.
func BuildCBZ(pageFiles []string, outPath string) error {
	if len(pageFiles) == 0 {
		return fmt.Errorf("no pages to bundle")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	out, err := os.Create(outPath) // #nosec G304 -- outPath is built by the caller from a sanitized manga/chapter title
	if err != nil {
		return fmt.Errorf("failed to create CBZ file: %w", err)
	}
	defer func() { _ = out.Close() }()

	zw := zip.NewWriter(out)
	defer func() { _ = zw.Close() }()

	for i, pagePath := range pageFiles {
		if err := addPageToZip(zw, pagePath, i); err != nil {
			return fmt.Errorf("failed to add page %d to CBZ: %w", i+1, err)
		}
	}

	return zw.Close()
}

func addPageToZip(zw *zip.Writer, pagePath string, index int) error {
	src, err := os.Open(pagePath) // #nosec G304 -- pagePath comes from files this package itself just downloaded
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	entryName := fmt.Sprintf("%04d%s", index+1, filepath.Ext(pagePath))
	w, err := zw.Create(entryName)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}
