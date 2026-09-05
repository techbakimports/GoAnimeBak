package gobridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// mangaDexHTTPClient is a dedicated client for MangaDex API calls.
var mangaDexHTTPClient = &http.Client{Timeout: 20 * time.Second}

// mangaDexBaseURL is a var (not a const) so tests can point it at a mock
// server.
var mangaDexBaseURL = "https://api.mangadex.org"

// mangaDexUserAgent identifies GoAnime to the MangaDex API, per their API
// usage guidelines (https://api.mangadex.org/docs/).
const mangaDexUserAgent = "GoAnime-Android/1.0 (+https://github.com/alvarorichard/Goanime)"

// mangaPreferredLanguages controls which chapter translations are
// surfaced, in priority order: PT-BR first, then English.
var mangaPreferredLanguages = []string{"pt-br", "en"}

// --- internal MangaDex API types ---

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

// --- public bridge types ---

// MangaResult is a manga search hit exposed to Android.
type MangaResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Status      string `json:"status"`
	Description string `json:"description"`
	CoverURL    string `json:"coverUrl"`
}

// ChapterResult is one chapter entry exposed to Android.
type ChapterResult struct {
	ID       string `json:"id"`
	Chapter  string `json:"chapter"`
	Volume   string `json:"volume"`
	Title    string `json:"title"`
	Language string `json:"language"`
	Pages    int    `json:"pages"`
}

// --- helpers ---

func mangaDexGetJSON(path string, v any) error {
	req, err := http.NewRequest(http.MethodGet, mangaDexBaseURL+path, nil) // #nosec G107 -- mangaDexBaseURL is fixed at package init (only overridden by tests); path segments are URL-escaped by callers
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", mangaDexUserAgent)

	resp, err := mangaDexHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
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
	title := firstNonEmptyMangaField(m.Attributes.Title, "en", "ja-ro", "ja")
	description := firstNonEmptyMangaField(m.Attributes.Description, "en")

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

func firstNonEmptyMangaField(m map[string]string, keys ...string) string {
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

// --- exported bridge functions ---

// SearchManga searches MangaDex by title, excluding erotica/pornographic
// content ratings, ranked by relevance. Returns a JSON array of
// MangaResult.
func SearchManga(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", "20")
	params.Add("contentRating[]", "safe")
	params.Add("contentRating[]", "suggestive")
	params.Add("order[relevance]", "desc")
	params.Add("includes[]", "cover_art")

	var parsed mdSearchResponse
	if err := mangaDexGetJSON("/manga?"+params.Encode(), &parsed); err != nil {
		return "", fmt.Errorf("manga search failed: %w", err)
	}

	results := make([]MangaResult, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		results = append(results, convertMangaEntry(m))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to serialize results: %w", err)
	}
	return string(data), nil
}

// GetMangaChapters lists a manga's chapters in the preferred languages
// (PT-BR, then English), ordered ascending, excluding chapters that aren't
// actually hosted on MangaDex. Returns a JSON array of ChapterResult.
func GetMangaChapters(mangaID string) (string, error) {
	if mangaID == "" {
		return "", fmt.Errorf("mangaID cannot be empty")
	}

	params := url.Values{}
	for _, lang := range mangaPreferredLanguages {
		params.Add("translatedLanguage[]", lang)
	}
	params.Set("order[chapter]", "asc")
	params.Set("limit", "500")

	var parsed mdFeedResponse
	if err := mangaDexGetJSON("/manga/"+url.PathEscape(mangaID)+"/feed?"+params.Encode(), &parsed); err != nil {
		return "", fmt.Errorf("failed to list chapters: %w", err)
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

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to serialize results: %w", err)
	}
	return string(data), nil
}

// GetChapterPageURLs resolves the full-quality page image URLs for a
// chapter via MangaDex's at-home server endpoint. Returns a JSON array of
// strings. The caller (Android) downloads these directly and assembles the
// CBZ/PDF natively.
func GetChapterPageURLs(chapterID string) (string, error) {
	if chapterID == "" {
		return "", fmt.Errorf("chapterID cannot be empty")
	}

	var parsed mdAtHomeResponse
	if err := mangaDexGetJSON("/at-home/server/"+url.PathEscape(chapterID), &parsed); err != nil {
		return "", fmt.Errorf("failed to resolve chapter pages: %w", err)
	}
	if parsed.BaseURL == "" || parsed.Chapter.Hash == "" {
		return "", fmt.Errorf("chapter %s has no available pages", chapterID)
	}

	pages := make([]string, 0, len(parsed.Chapter.Data))
	for _, filename := range parsed.Chapter.Data {
		pages = append(pages, parsed.BaseURL+"/data/"+parsed.Chapter.Hash+"/"+filename)
	}

	data, err := json.Marshal(pages)
	if err != nil {
		return "", fmt.Errorf("failed to serialize page URLs: %w", err)
	}
	return string(data), nil
}
