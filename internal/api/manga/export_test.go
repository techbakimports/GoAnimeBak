package manga

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestPNG writes a tiny solid-color PNG so BuildPDF's image.DecodeConfig
// call has something real to read.
func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}
}

func TestBuildCBZ(t *testing.T) {
	dir := t.TempDir()
	pages := []string{
		filepath.Join(dir, "0001.png"),
		filepath.Join(dir, "0002.png"),
	}
	for _, p := range pages {
		writeTestPNG(t, p, 10, 10)
	}

	outPath := filepath.Join(dir, "chapter.cbz")
	if err := BuildCBZ(pages, outPath); err != nil {
		t.Fatalf("BuildCBZ failed: %v", err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("failed to open CBZ as zip: %v", err)
	}
	defer func() { _ = r.Close() }()

	if len(r.File) != 2 {
		t.Fatalf("expected 2 entries in CBZ, got %d", len(r.File))
	}
	if r.File[0].Name != "0001.png" || r.File[1].Name != "0002.png" {
		t.Errorf("unexpected entry names: %s, %s", r.File[0].Name, r.File[1].Name)
	}
}

func TestBuildCBZ_NoPages(t *testing.T) {
	if err := BuildCBZ(nil, filepath.Join(t.TempDir(), "out.cbz")); err == nil {
		t.Fatal("expected an error when there are no pages")
	}
}

func TestBuildPDF(t *testing.T) {
	dir := t.TempDir()
	pages := []string{
		filepath.Join(dir, "0001.png"),
		filepath.Join(dir, "0002.png"),
	}
	writeTestPNG(t, pages[0], 20, 30)
	writeTestPNG(t, pages[1], 20, 30)

	outPath := filepath.Join(dir, "chapter.pdf")
	if err := BuildPDF(pages, outPath); err != nil {
		t.Fatalf("BuildPDF failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("expected output PDF to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty PDF file")
	}
}

func TestBuildPDF_NoPages(t *testing.T) {
	if err := BuildPDF(nil, filepath.Join(t.TempDir(), "out.pdf")); err == nil {
		t.Fatal("expected an error when there are no pages")
	}
}
