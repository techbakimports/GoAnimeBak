// Package manga provides MangaDex API integration for searching manga,
// listing chapters, and resolving chapter page image URLs.
//
// The MangaDex read API is public and keyless.
package manga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL = "https://api.mangadex.org"

	// userAgent identifies GoAnime to the MangaDex API, per their API usage
	// guidelines (https://api.mangadex.org/docs/).
	userAgent = "GoAnime/1.0 (+https://github.com/alvarorichard/Goanime)"

	maxResponseBytes = 5 * 1024 * 1024 // 5MB
)

// preferredLanguages controls which chapter translations are surfaced,
// in priority order: PT-BR first (GoAnime's primary audience), then English.
var preferredLanguages = []string{"pt-br", "en"}

// Client talks to the MangaDex API.
type Client struct {
	client  *http.Client
	baseURL string // overridable in tests to point at a mock server
}

// NewClient creates a MangaDex client with the shared SSRF-safe transport.
func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout:   20 * time.Second,
			Transport: safeMangaTransport(20 * time.Second),
		},
		baseURL: baseURL,
	}
}

// MangaResult is a manga search hit.
type MangaResult struct {
	ID          string
	Title       string
	Year        int
	Status      string
	Description string
	CoverURL    string
}

// ChapterResult is one entry in a manga's chapter feed.
type ChapterResult struct {
	ID       string
	Chapter  string // e.g. "1050" or "" for a oneshot
	Volume   string
	Title    string
	Language string
	Pages    int
}

// --- MangaDex API response shapes (unexported) ---

type mdSearchResponse struct {
	Data []mdMangaEntry `json:"data"`
}

type mdMangaEntry struct {
	ID         string `json:"id"`
	Attributes struct {
		Title       map[string]string `json:"title"`
		Description map[string]string `json:"description"`
		Status      string            `json:"status"`
		Year        int               `json:"year"`
	} `json:"attributes"`
	Relationships []mdRelationship `json:"relationships"`
}

type mdRelationship struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		FileName string `json:"fileName"`
	} `json:"attributes"`
}

type mdFeedResponse struct {
	Data []mdChapterEntry `json:"data"`
}

type mdChapterEntry struct {
	ID         string `json:"id"`
	Attributes struct {
		Title              string  `json:"title"`
		Volume             string  `json:"volume"`
		Chapter            string  `json:"chapter"`
		TranslatedLanguage string  `json:"translatedLanguage"`
		Pages              int     `json:"pages"`
		ExternalURL        *string `json:"externalUrl"`
	} `json:"attributes"`
}

type mdAtHomeResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"`
	} `json:"chapter"`
}

// SearchManga searches MangaDex by title, excluding erotica/pornographic
// content ratings, ranked by relevance.
func (c *Client) SearchManga(ctx context.Context, title string) ([]MangaResult, error) {
	params := url.Values{}
	params.Set("title", title)
	params.Set("limit", "20")
	params.Add("contentRating[]", "safe")
	params.Add("contentRating[]", "suggestive")
	params.Add("order[relevance]", "desc")
	params.Add("includes[]", "cover_art")

	var parsed mdSearchResponse
	if err := c.getJSON(ctx, "/manga?"+params.Encode(), &parsed); err != nil {
		return nil, fmt.Errorf("manga search failed: %w", err)
	}

	results := make([]MangaResult, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		results = append(results, convertMangaEntry(m))
	}
	return results, nil
}

// GetChapters lists a manga's chapters in the preferred languages
// (PT-BR, then English), ordered ascending, excluding chapters that aren't
// actually hosted on MangaDex (no pages / external-only chapters).
func (c *Client) GetChapters(ctx context.Context, mangaID string) ([]ChapterResult, error) {
	params := url.Values{}
	for _, lang := range preferredLanguages {
		params.Add("translatedLanguage[]", lang)
	}
	params.Set("order[chapter]", "asc")
	params.Set("limit", "500")

	var parsed mdFeedResponse
	if err := c.getJSON(ctx, "/manga/"+url.PathEscape(mangaID)+"/feed?"+params.Encode(), &parsed); err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	results := make([]ChapterResult, 0, len(parsed.Data))
	for _, ch := range parsed.Data {
		if ch.Attributes.Pages == 0 || ch.Attributes.ExternalURL != nil {
			continue // not hosted on MangaDex
		}
		results = append(results, ChapterResult{
			ID:       ch.ID,
			Chapter:  ch.Attributes.Chapter,
			Volume:   ch.Attributes.Volume,
			Title:    ch.Attributes.Title,
			Language: ch.Attributes.TranslatedLanguage,
			Pages:    ch.Attributes.Pages,
		})
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no downloadable chapters found in %v", preferredLanguages)
	}
	return results, nil
}

// GetChapterPageURLs resolves the full-quality page image URLs for a
// chapter via MangaDex's at-home server endpoint.
func (c *Client) GetChapterPageURLs(ctx context.Context, chapterID string) ([]string, error) {
	var parsed mdAtHomeResponse
	if err := c.getJSON(ctx, "/at-home/server/"+url.PathEscape(chapterID), &parsed); err != nil {
		return nil, fmt.Errorf("failed to resolve chapter pages: %w", err)
	}
	if parsed.BaseURL == "" || parsed.Chapter.Hash == "" {
		return nil, fmt.Errorf("chapter %s has no available pages", chapterID)
	}

	pages := make([]string, 0, len(parsed.Chapter.Data))
	for _, filename := range parsed.Chapter.Data {
		pages = append(pages, parsed.BaseURL+"/data/"+parsed.Chapter.Hash+"/"+filename)
	}
	return pages, nil
}

// getJSON performs a GET against the MangaDex API and decodes the JSON
// response into v.
func (c *Client) getJSON(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req) // #nosec G704 -- baseURL is fixed at construction, path segments are URL-escaped
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MangaDex returned status %d", resp.StatusCode)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

func convertMangaEntry(m mdMangaEntry) MangaResult {
	title := firstNonEmpty(m.Attributes.Title, "en", "ja-ro", "ja")
	description := firstNonEmpty(m.Attributes.Description, "en")

	coverURL := ""
	for _, rel := range m.Relationships {
		if rel.Type == "cover_art" && rel.Attributes.FileName != "" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", m.ID, rel.Attributes.FileName)
			break
		}
	}

	return MangaResult{
		ID:          m.ID,
		Title:       title,
		Year:        m.Attributes.Year,
		Status:      m.Attributes.Status,
		Description: description,
		CoverURL:    coverURL,
	}
}

// firstNonEmpty returns the first non-empty value among the given keys, or
// any single value in the map as a last resort.
func firstNonEmpty(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := m[k]; v != "" {
			return v
		}
	}
	for _, v := range m {
		if v != "" {
			return v
		}
	}
	return ""
}
