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

// ytsHTTPClient is a dedicated client for YTS API calls.
var ytsHTTPClient = &http.Client{Timeout: 15 * time.Second}

// ytsTrackers are the standard BitTorrent trackers bundled in YTS magnet links.
var ytsTrackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80",
	"udp://tracker.coppersurfer.tk:6969",
	"udp://glotorrents.pw:6969/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://torrent.gresille.org:80/announce",
	"udp://p4p.arenabg.com:1337",
	"udp://tracker.leechers-paradise.org:6969",
}

// --- internal YTS API types ---

type ytsAPIResponse struct {
	Status string `json:"status"`
	Data   struct {
		Movies []ytsAPIMovie `json:"movies"`
	} `json:"data"`
}

type ytsAPIMovie struct {
	ID              int           `json:"id"`
	Title           string        `json:"title"`
	Year            int           `json:"year"`
	Rating          float32       `json:"rating"`
	Genres          []string      `json:"genres"`
	DescriptionFull string        `json:"description_full"`
	LargeCoverImage string        `json:"large_cover_image"`
	Torrents        []ytsAPITorrent `json:"torrents"`
}

type ytsAPITorrent struct {
	Hash      string `json:"hash"`
	Quality   string `json:"quality"`
	Type      string `json:"type"` // "bluray" or "web"
	SizeBytes int64  `json:"size_bytes"`
}

// --- public bridge types ---

// YTSMovieResult is the JSON-serializable movie result exposed to Android.
type YTSMovieResult struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Year       int      `json:"year"`
	Rating     float32  `json:"rating"`
	Genres     []string `json:"genres"`
	Summary    string   `json:"summary"`
	CoverLarge string   `json:"coverLarge"`
	MagnetURL  string   `json:"magnetUrl"`
	Quality    string   `json:"quality"`
	SizeMB     int64    `json:"sizeMb"`
}

// --- helpers ---

func buildMagnetURL(hash, title string) string {
	var sb strings.Builder
	sb.WriteString("magnet:?xt=urn:btih:")
	sb.WriteString(hash)
	sb.WriteString("&dn=")
	sb.WriteString(url.QueryEscape(title))
	for _, tr := range ytsTrackers {
		sb.WriteString("&tr=")
		sb.WriteString(url.QueryEscape(tr))
	}
	return sb.String()
}

// torrentScore ranks torrents so we always pick the best available quality.
// Scoring: 1080p > 2160p > 720p; bluray adds a small bonus over web.
func torrentScore(t ytsAPITorrent) int {
	s := 0
	switch t.Quality {
	case "1080p":
		s = 30
	case "2160p":
		s = 25
	case "720p":
		s = 10
	}
	if t.Type == "bluray" {
		s++
	}
	return s
}

func bestTorrent(torrents []ytsAPITorrent) *ytsAPITorrent {
	var best *ytsAPITorrent
	for i := range torrents {
		if best == nil || torrentScore(torrents[i]) > torrentScore(*best) {
			best = &torrents[i]
		}
	}
	return best
}

func convertYTSMovie(m ytsAPIMovie) *YTSMovieResult {
	t := bestTorrent(m.Torrents)
	if t == nil {
		return nil
	}
	return &YTSMovieResult{
		ID:         m.ID,
		Title:      m.Title,
		Year:       m.Year,
		Rating:     m.Rating,
		Genres:     m.Genres,
		Summary:    m.DescriptionFull,
		CoverLarge: m.LargeCoverImage,
		MagnetURL:  buildMagnetURL(t.Hash, m.Title),
		Quality:    t.Quality,
		SizeMB:     t.SizeBytes / (1024 * 1024),
	}
}

// --- exported bridge functions ---

// SearchMoviesYTS searches YTS for animation movies matching query.
// genre defaults to "animation" when empty.
// Returns a JSON array of YTSMovieResult, or an error string.
func SearchMoviesYTS(query, genre string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}
	if genre == "" {
		genre = "animation"
	}

	apiURL := fmt.Sprintf(
		"https://yts.mx/api/v2/list_movies.json?query_term=%s&genre=%s&sort_by=seeds&limit=20&minimum_rating=5",
		url.QueryEscape(query),
		url.QueryEscape(genre),
	)

	resp, err := ytsHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("YTS API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("YTS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read YTS response: %w", err)
	}

	var apiResp ytsAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse YTS response: %w", err)
	}
	if apiResp.Status != "ok" {
		return "", fmt.Errorf("YTS API error: status=%s", apiResp.Status)
	}

	results := make([]YTSMovieResult, 0, len(apiResp.Data.Movies))
	for _, m := range apiResp.Data.Movies {
		if r := convertYTSMovie(m); r != nil {
			results = append(results, *r)
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to serialize results: %w", err)
	}
	return string(data), nil
}

// GetMovieYTS returns details (including magnet link) for a single YTS movie by ID.
func GetMovieYTS(movieID int) (string, error) {
	if movieID <= 0 {
		return "", fmt.Errorf("invalid movie ID: %d", movieID)
	}

	apiURL := fmt.Sprintf(
		"https://yts.mx/api/v2/movie_details.json?movie_id=%d",
		movieID,
	)

	resp, err := ytsHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("YTS API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read YTS response: %w", err)
	}

	var raw struct {
		Status string       `json:"status"`
		Data   struct{ Movie ytsAPIMovie `json:"movie"` } `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("failed to parse YTS response: %w", err)
	}
	if raw.Status != "ok" {
		return "", fmt.Errorf("YTS API error: status=%s", raw.Status)
	}

	result := convertYTSMovie(raw.Data.Movie)
	if result == nil {
		return "", fmt.Errorf("movie %d has no downloadable torrents", movieID)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to serialize result: %w", err)
	}
	return string(data), nil
}
