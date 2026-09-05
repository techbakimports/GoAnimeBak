package manga

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestSearchManga_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manga", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "ok",
			"data": [
				{
					"id": "abc-123",
					"type": "manga",
					"attributes": {
						"title": {"en": "One Piece"},
						"description": {"en": "A pirate adventure."},
						"status": "ongoing",
						"year": 1997
					},
					"relationships": [
						{"id": "cover-1", "type": "cover_art", "attributes": {"fileName": "cover.jpg"}}
					]
				}
			]
		}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv.URL)
	results, err := client.SearchManga(context.Background(), "one piece")
	if err != nil {
		t.Fatalf("SearchManga failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.ID != "abc-123" || got.Title != "One Piece" || got.Year != 1997 {
		t.Errorf("unexpected result: %+v", got)
	}
	wantCover := "https://uploads.mangadex.org/covers/abc-123/cover.jpg.256.jpg"
	if got.CoverURL != wantCover {
		t.Errorf("CoverURL = %q, want %q", got.CoverURL, wantCover)
	}
}

func TestGetChapters_FiltersUnhostedChapters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manga/abc-123/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "ok",
			"data": [
				{"id": "ch-1", "type": "chapter", "attributes": {"chapter": "1", "translatedLanguage": "pt-br", "pages": 18, "externalUrl": null}},
				{"id": "ch-2", "type": "chapter", "attributes": {"chapter": "2", "translatedLanguage": "pt-br", "pages": 0, "externalUrl": "https://example.com"}},
				{"id": "ch-3", "type": "chapter", "attributes": {"chapter": "3", "translatedLanguage": "en", "pages": 20, "externalUrl": null}}
			]
		}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv.URL)
	chapters, err := client.GetChapters(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("GetChapters failed: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 hosted chapters, got %d: %+v", len(chapters), chapters)
	}
	if chapters[0].ID != "ch-1" || chapters[1].ID != "ch-3" {
		t.Errorf("unexpected chapters: %+v", chapters)
	}
}

func TestGetChapters_NoneHosted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manga/abc-123/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": "ok", "data": []}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv.URL)
	if _, err := client.GetChapters(context.Background(), "abc-123"); err == nil {
		t.Fatal("expected an error when no chapters are hosted")
	}
}

func TestGetChapterPageURLs_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/at-home/server/ch-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "ok",
			"baseUrl": "https://cdn.example.com",
			"chapter": {"hash": "hash123", "data": ["1.png", "2.png"]}
		}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv.URL)
	pages, err := client.GetChapterPageURLs(context.Background(), "ch-1")
	if err != nil {
		t.Fatalf("GetChapterPageURLs failed: %v", err)
	}
	want := []string{
		"https://cdn.example.com/data/hash123/1.png",
		"https://cdn.example.com/data/hash123/2.png",
	}
	if len(pages) != len(want) {
		t.Fatalf("expected %d pages, got %d", len(want), len(pages))
	}
	for i, p := range pages {
		if p != want[i] {
			t.Errorf("page[%d] = %q, want %q", i, p, want[i])
		}
	}
}

func TestGetChapterPageURLs_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/at-home/server/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv.URL)
	if _, err := client.GetChapterPageURLs(context.Background(), "missing"); err == nil {
		t.Fatal("expected an error for a missing chapter")
	}
}
