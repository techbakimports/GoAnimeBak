package manga

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/alvarorichard/Goanime/internal/util"
)

// maxPageBytes caps how much of a single page image is read, as a safety
// limit against a misbehaving/compromised CDN node.
const maxPageBytes = 20 * 1024 * 1024 // 20MB

// pageDownloadWorkers bounds how many pages are fetched concurrently.
const pageDownloadWorkers = 6

// newPageHTTPClient builds the client used to fetch page images. It's a var
// (not inlined) so tests can swap in a plain client, bypassing the SSRF-safe
// transport, which rejects the loopback addresses httptest servers use.
var newPageHTTPClient = func() *http.Client {
	return &http.Client{Transport: safeMangaTransport(2 * time.Minute)}
}

// DownloadPages downloads each page URL into destDir (created if it
// doesn't exist), naming files 0001.<ext>, 0002.<ext>, ... to preserve page
// order, and returns the local file paths in page order.
//
// onProgress, if non-nil, is called after each page finishes (success or
// failure) with how many pages have completed so far.
func DownloadPages(ctx context.Context, urls []string, destDir string, onProgress func(done, total int)) ([]string, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no pages to download")
	}
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	client := newPageHTTPClient()

	paths := make([]string, len(urls))
	errs := make([]error, len(urls))
	var completed int32

	tasks := make([]func(), len(urls))
	for i, pageURL := range urls {
		i, pageURL := i, pageURL
		tasks[i] = func() {
			path, err := downloadPage(ctx, client, pageURL, destDir, i)
			paths[i] = path
			errs[i] = err
			done := int(atomic.AddInt32(&completed, 1))
			if onProgress != nil {
				onProgress(done, len(urls))
			}
		}
	}

	util.ParallelExecute(pageDownloadWorkers, tasks...)

	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("failed to download page %d: %w", i+1, err)
		}
	}
	return paths, nil
}

// downloadPage fetches a single page image to destDir/NNNN.ext.
func downloadPage(ctx context.Context, client *http.Client, pageURL, destDir string, index int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req) // #nosec G704 -- pageURL comes from MangaDex's own at-home response, not raw user input
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	ext := filepath.Ext(pageURL)
	if ext == "" {
		ext = ".jpg"
	}
	dest := filepath.Join(destDir, fmt.Sprintf("%04d%s", index+1, ext))

	f, err := os.Create(dest) // #nosec G304 -- dest is built from a sequential index and a caller-controlled local directory
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxPageBytes)); err != nil {
		return "", err
	}
	return dest, nil
}
