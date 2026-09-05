package tracemoe

import (
	"context"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestClient builds a Client pointed at a mock server, bypassing the
// SSRF-safe transport (which rejects loopback addresses) the same way
// internal/api/movie's httptest-based tests do.
func newTestClient(serverURL string) *Client {
	return &Client{
		client:  &http.Client{},
		baseURL: serverURL,
	}
}

// tempImageFile writes a tiny fake image file for Search() to upload.
func tempImageFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "screenshot.png")
	if err := os.WriteFile(path, []byte("not a real png, just bytes"), 0o600); err != nil {
		t.Fatalf("failed to write temp image: %v", err)
	}
	return path
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if _, params, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || params["boundary"] == "" {
			t.Errorf("expected multipart content type with boundary, got %q", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"frameCount": 123,
			"error": "",
			"result": [
				{
					"anilist": {"id": 21, "idMal": 21, "title": {"romaji": "One Piece", "english": "One Piece", "native": "ワンピース"}},
					"filename": "1.jpg",
					"episode": 1050,
					"from": 733.7,
					"to": 737.7,
					"similarity": 0.9805154,
					"video": "https://media.trace.moe/video/1",
					"image": "https://media.trace.moe/image/1"
				},
				{
					"anilist": {"id": 20, "idMal": 20, "title": {"romaji": "Naruto"}},
					"filename": "2.jpg",
					"episode": null,
					"from": 10,
					"to": 12,
					"similarity": 0.5,
					"video": "",
					"image": ""
				}
			]
		}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	matches, err := client.Search(context.Background(), tempImageFile(t))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	best := matches[0]
	if best.AnilistID != 21 || best.TitleEnglish != "One Piece" || best.Episode != "1050" {
		t.Errorf("unexpected best match: %+v", best)
	}
	if got := FormatTimestamp(best.From); got != "12:13" {
		t.Errorf("FormatTimestamp(733.7) = %q, want 12:13", got)
	}

	if matches[1].Episode != "" {
		t.Errorf("expected empty episode for null value, got %q", matches[1].Episode)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"frameCount": 0, "error": "", "result": []}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	if _, err := client.Search(context.Background(), tempImageFile(t)); err == nil {
		t.Fatal("expected an error when no match is found")
	}
}

func TestSearch_QuotaExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.Search(context.Background(), tempImageFile(t))
	if err == nil {
		t.Fatal("expected a quota error")
	}
}

func TestSearch_ImageNotFound(t *testing.T) {
	client := newTestClient("http://example.invalid")
	if _, err := client.Search(context.Background(), filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Fatal("expected an error for a missing image file")
	}
}

func TestFormatTimestamp(t *testing.T) {
	cases := map[float64]string{
		0:      "00:00",
		5:      "00:05",
		65:     "01:05",
		-3:     "00:00",
		3661.9: "61:01",
	}
	for in, want := range cases {
		if got := FormatTimestamp(in); got != want {
			t.Errorf("FormatTimestamp(%v) = %q, want %q", in, got, want)
		}
	}
}
