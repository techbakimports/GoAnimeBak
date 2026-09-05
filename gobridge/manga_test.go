package gobridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// withMockMangaDexServer points mangaDexBaseURL at a mock server for the
// duration of the test and restores it afterward.
func withMockMangaDexServer(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	original := mangaDexBaseURL
	mangaDexBaseURL = srv.URL
	t.Cleanup(func() { mangaDexBaseURL = original })
}

func TestSearchManga_EmptyQuery(t *testing.T) {
	if _, err := SearchManga(""); err == nil {
		t.Fatal("expected an error for an empty query")
	}
}

func TestSearchManga_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manga", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "abc-123",
					"attributes": {"title": {"en": "One Piece"}, "status": "ongoing", "year": 1997},
					"relationships": [{"type": "cover_art", "attributes": {"fileName": "cover.jpg"}}]
				}
			]
		}`))
	})
	withMockMangaDexServer(t, mux)

	raw, err := SearchManga("one piece")
	if err != nil {
		t.Fatalf("SearchManga failed: %v", err)
	}

	var results []MangaResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	if len(results) != 1 || results[0].Title != "One Piece" {
		t.Errorf("unexpected results: %+v", results)
	}
	wantCover := "https://uploads.mangadex.org/covers/abc-123/cover.jpg.256.jpg"
	if results[0].CoverURL != wantCover {
		t.Errorf("CoverURL = %q, want %q", results[0].CoverURL, wantCover)
	}
}

func TestGetMangaChapters_EmptyID(t *testing.T) {
	if _, err := GetMangaChapters(""); err == nil {
		t.Fatal("expected an error for an empty manga ID")
	}
}

func TestGetMangaChapters_FiltersUnhosted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manga/abc-123/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "ch-1", "attributes": {"chapter": "1", "translatedLanguage": "pt-br", "pages": 18, "externalUrl": null}},
				{"id": "ch-2", "attributes": {"chapter": "2", "translatedLanguage": "pt-br", "pages": 0, "externalUrl": "https://example.com"}}
			]
		}`))
	})
	withMockMangaDexServer(t, mux)

	raw, err := GetMangaChapters("abc-123")
	if err != nil {
		t.Fatalf("GetMangaChapters failed: %v", err)
	}

	var results []ChapterResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	if len(results) != 1 || results[0].ID != "ch-1" {
		t.Errorf("unexpected results: %+v", results)
	}
}

func TestGetChapterPageURLs_EmptyID(t *testing.T) {
	if _, err := GetChapterPageURLs(""); err == nil {
		t.Fatal("expected an error for an empty chapter ID")
	}
}

func TestGetChapterPageURLs_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/at-home/server/ch-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"baseUrl": "https://cdn.example.com", "chapter": {"hash": "hash123", "data": ["1.png", "2.png"]}}`))
	})
	withMockMangaDexServer(t, mux)

	raw, err := GetChapterPageURLs("ch-1")
	if err != nil {
		t.Fatalf("GetChapterPageURLs failed: %v", err)
	}

	var pages []string
	if err := json.Unmarshal([]byte(raw), &pages); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}
	want := []string{
		"https://cdn.example.com/data/hash123/1.png",
		"https://cdn.example.com/data/hash123/2.png",
	}
	if len(pages) != len(want) {
		t.Fatalf("expected %d pages, got %d", len(want), len(pages))
	}
	for i := range want {
		if pages[i] != want[i] {
			t.Errorf("page[%d] = %q, want %q", i, pages[i], want[i])
		}
	}
}
