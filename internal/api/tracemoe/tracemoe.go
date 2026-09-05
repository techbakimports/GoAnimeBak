// Package tracemoe provides trace.moe API integration for identifying which
// anime, episode, and timestamp a screenshot was taken from (reverse image
// search against a database of anime frames).
//
// The API is free and keyless for normal use; an optional API key can be
// set via WithAPIKey for a higher search quota.
package tracemoe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// searchURL is the trace.moe search endpoint. anilistInfo asks the API
	// to embed AniList title/ID data in the result so we don't need a
	// second lookup; cutBorders trims black letterbox bars before matching.
	searchURL = "https://api.trace.moe/search?anilistInfo&cutBorders"

	// maxImageBytes caps how much of a local file we read into the upload,
	// as a safety limit (trace.moe itself also enforces a server-side cap).
	maxImageBytes = 10 * 1024 * 1024 // 10MB

	// maxResponseBytes caps how much of the API response we buffer.
	maxResponseBytes = 5 * 1024 * 1024 // 5MB
)

// Client talks to the trace.moe API.
type Client struct {
	client  *http.Client
	apiKey  string
	baseURL string // overridable in tests to point at a mock server
}

// NewClient creates a trace.moe client with the shared SSRF-safe transport.
func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: safeTraceMoeTransport(30 * time.Second),
		},
		baseURL: searchURL,
	}
}

// WithAPIKey sets an optional trace.moe API key (higher search quota) and
// returns the client for chaining.
func (c *Client) WithAPIKey(key string) *Client {
	c.apiKey = key
	return c
}

// Match is one candidate scene match, ranked by Similarity (best first, as
// returned by the API).
type Match struct {
	AnilistID    int
	MalID        int
	TitleRomaji  string
	TitleEnglish string
	TitleNative  string
	Episode      string  // may be empty (e.g. movies) or non-numeric (e.g. "OVA")
	From         float64 // seconds into the episode where the match starts
	To           float64 // seconds into the episode where the match ends
	Similarity   float64 // 0..1
	ImageURL     string  // preview thumbnail
	VideoURL     string  // preview clip
}

// --- trace.moe API response shapes (unexported, internal to this package) ---

type apiResponse struct {
	FrameCount int         `json:"frameCount"`
	Error      string      `json:"error"`
	Result     []apiResult `json:"result"`
}

type apiResult struct {
	Anilist    apiAnilist      `json:"anilist"`
	Filename   string          `json:"filename"`
	Episode    json.RawMessage `json:"episode"`
	From       float64         `json:"from"`
	To         float64         `json:"to"`
	Similarity float64         `json:"similarity"`
	Video      string          `json:"video"`
	Image      string          `json:"image"`
}

type apiAnilist struct {
	ID    int `json:"id"`
	IDMal int `json:"idMal"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
}

// Search uploads the image at imagePath to trace.moe and returns the
// ranked scene matches. imagePath is expected to already be validated by
// the caller (e.g. the CLI flag parser checks os.Stat before this is
// reached).
func (c *Client) Search(ctx context.Context, imagePath string) ([]Match, error) {
	f, err := os.Open(imagePath) // #nosec G304 -- path is a CLI-validated local flag argument, not user-controlled network input
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer func() { _ = f.Close() }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("failed to build upload: %w", err)
	}
	if _, err := io.Copy(part, io.LimitReader(f, maxImageBytes)); err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to build upload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("x-trace-key", c.apiKey)
	}

	resp, err := c.client.Do(req) // #nosec G704 -- baseURL is fixed at construction, not built from request-time input
	if err != nil {
		return nil, fmt.Errorf("trace.moe request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read trace.moe response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusPaymentRequired {
		return nil, fmt.Errorf("trace.moe search quota exceeded (status %d): try again later or set a trace.moe API key", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trace.moe returned status %d", resp.StatusCode)
	}

	var parsed apiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse trace.moe response: %w", err)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("trace.moe error: %s", parsed.Error)
	}
	if len(parsed.Result) == 0 {
		return nil, errors.New("no match found for this image")
	}

	matches := make([]Match, 0, len(parsed.Result))
	for _, r := range parsed.Result {
		matches = append(matches, Match{
			AnilistID:    r.Anilist.ID,
			MalID:        r.Anilist.IDMal,
			TitleRomaji:  r.Anilist.Title.Romaji,
			TitleEnglish: r.Anilist.Title.English,
			TitleNative:  r.Anilist.Title.Native,
			Episode:      formatEpisode(r.Episode),
			From:         r.From,
			To:           r.To,
			Similarity:   r.Similarity,
			ImageURL:     r.Image,
			VideoURL:     r.Video,
		})
	}
	return matches, nil
}

// formatEpisode converts trace.moe's loosely-typed "episode" field (it can
// be a number, a string like "OVA", an array, or null) into a display
// string, defaulting to empty when the value isn't a simple number/string.
func formatEpisode(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var asNum float64
	if err := json.Unmarshal(raw, &asNum); err == nil {
		if asNum == math.Trunc(asNum) {
			return strconv.Itoa(int(asNum))
		}
		return strconv.FormatFloat(asNum, 'f', -1, 64)
	}

	var asStr string
	if err := json.Unmarshal(raw, &asStr); err == nil {
		return asStr
	}

	return strings.Trim(string(raw), `"`)
}

// FormatTimestamp renders a seconds offset as mm:ss for display.
func FormatTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}
