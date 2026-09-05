package manga

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// useLoopbackHTTPClient bypasses the SSRF-safe transport (which rejects
// loopback addresses) for the duration of a test, so it can talk to an
// httptest server. Restored automatically via t.Cleanup.
func useLoopbackHTTPClient(t *testing.T) {
	t.Helper()
	original := newPageHTTPClient
	newPageHTTPClient = func() *http.Client { return &http.Client{} }
	t.Cleanup(func() { newPageHTTPClient = original })
}

func TestDownloadPages_Success(t *testing.T) {
	useLoopbackHTTPClient(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake-image-bytes:" + r.URL.Path))
	}))
	defer srv.Close()

	urls := []string{
		srv.URL + "/data/hash/1.png",
		srv.URL + "/data/hash/2.png",
		srv.URL + "/data/hash/3.png",
	}

	destDir := t.TempDir()

	var mu sync.Mutex
	var progressCalls []int
	onProgress := func(done, total int) {
		mu.Lock()
		defer mu.Unlock()
		progressCalls = append(progressCalls, done)
		if total != len(urls) {
			t.Errorf("progress total = %d, want %d", total, len(urls))
		}
	}

	paths, err := DownloadPages(context.Background(), urls, destDir, onProgress)
	if err != nil {
		t.Fatalf("DownloadPages failed: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 downloaded files, got %d", len(paths))
	}
	if len(progressCalls) != 3 {
		t.Errorf("expected 3 progress callbacks, got %d", len(progressCalls))
	}

	// Page order must be preserved regardless of concurrent completion order.
	for i, p := range paths {
		if filepath.Base(p) != "000"+string(rune('1'+i))+".png" {
			t.Errorf("path[%d] = %q, expected page %d's file", i, p, i+1)
		}
		data, err := os.ReadFile(p) // #nosec G304 -- path is one this test just downloaded into its own temp dir
		if err != nil {
			t.Fatalf("failed to read downloaded page %d: %v", i+1, err)
		}
		if !strings.Contains(string(data), "fake-image-bytes") {
			t.Errorf("unexpected content for page %d: %q", i+1, data)
		}
	}
}

func TestDownloadPages_NoURLs(t *testing.T) {
	if _, err := DownloadPages(context.Background(), nil, t.TempDir(), nil); err == nil {
		t.Fatal("expected an error when there are no URLs")
	}
}

func TestDownloadPages_ServerError(t *testing.T) {
	useLoopbackHTTPClient(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := DownloadPages(context.Background(), []string{srv.URL + "/1.png"}, t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected an error when the server fails")
	}
}
