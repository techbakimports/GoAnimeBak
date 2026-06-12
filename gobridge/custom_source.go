package gobridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var customHTTPClient = &http.Client{Timeout: 20 * time.Second}

// SearchCustomSource calls {baseURL}/search?q={query} and returns a JSON array
// of AnimeResult objects. The remote server must implement the AniNex source schema.
func SearchCustomSource(baseURL string, query string) (string, error) {
	if baseURL == "" || query == "" {
		return "", fmt.Errorf("baseURL and query cannot be empty")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/search?q=" + url.QueryEscape(query)
	body, err := fetchJSON(endpoint)
	if err != nil {
		return "", fmt.Errorf("custom source search failed: %w", err)
	}
	return body, nil
}

// GetEpisodesCustom calls {baseURL}/episodes?url={animeURL} and returns a JSON
// array of EpisodeResult objects.
func GetEpisodesCustom(baseURL string, animeURL string) (string, error) {
	if baseURL == "" || animeURL == "" {
		return "", fmt.Errorf("baseURL and animeURL cannot be empty")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/episodes?url=" + url.QueryEscape(animeURL)
	body, err := fetchJSON(endpoint)
	if err != nil {
		return "", fmt.Errorf("custom source episodes failed: %w", err)
	}
	return body, nil
}

// GetStreamCustom calls {baseURL}/stream?url={episodeURL} and returns a JSON
// StreamResult with the direct video URL.
func GetStreamCustom(baseURL string, episodeURL string) (string, error) {
	if baseURL == "" || episodeURL == "" {
		return "", fmt.Errorf("baseURL and episodeURL cannot be empty")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/stream?url=" + url.QueryEscape(episodeURL)
	body, err := fetchJSON(endpoint)
	if err != nil {
		return "", fmt.Errorf("custom source stream failed: %w", err)
	}

	// Validate that the response is a StreamResult
	var result StreamResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return "", fmt.Errorf("custom source returned invalid stream JSON: %w", err)
	}
	if result.URL == "" {
		return "", fmt.Errorf("custom source returned empty stream URL")
	}
	return body, nil
}

func fetchJSON(endpoint string) (string, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AniNex/1.0")

	resp, err := customHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
