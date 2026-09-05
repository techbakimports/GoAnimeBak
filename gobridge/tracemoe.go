package gobridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// traceMoeHTTPClient is a dedicated client for trace.moe API calls.
var traceMoeHTTPClient = &http.Client{Timeout: 30 * time.Second}

// traceMoeSearchURL is a var (not a const) so tests can point it at a mock
// server. anilistInfo embeds AniList title/ID data in the result so a
// second lookup isn't needed; cutBorders trims black letterbox bars.
var traceMoeSearchURL = "https://api.trace.moe/search?anilistInfo&cutBorders"

// --- internal trace.moe API types ---

type traceMoeAPIResponse struct {
	FrameCount int                 `json:"frameCount"`
	Error      string              `json:"error"`
	Result     []traceMoeAPIResult `json:"result"`
}

type traceMoeAPIResult struct {
	Anilist    traceMoeAPIAnilist `json:"anilist"`
	Episode    json.RawMessage    `json:"episode"`
	From       float64            `json:"from"`
	To         float64            `json:"to"`
	Similarity float64            `json:"similarity"`
	Video      string             `json:"video"`
	Image      string             `json:"image"`
}

type traceMoeAPIAnilist struct {
	ID    int `json:"id"`
	IDMal int `json:"idMal"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
}

// --- public bridge type ---

// TraceMoeResult is the JSON-serializable scene match exposed to Android.
type TraceMoeResult struct {
	AnilistID    int     `json:"anilistId"`
	MalID        int     `json:"malId"`
	TitleRomaji  string  `json:"titleRomaji"`
	TitleEnglish string  `json:"titleEnglish"`
	TitleNative  string  `json:"titleNative"`
	Episode      string  `json:"episode"`
	From         float64 `json:"from"`
	To           float64 `json:"to"`
	Similarity   float64 `json:"similarity"`
	ImageURL     string  `json:"imageUrl"`
	VideoURL     string  `json:"videoUrl"`
}

// --- helpers ---

// formatTraceMoeEpisode converts trace.moe's loosely-typed "episode" field
// (number, string like "OVA", or null) into a display string.
func formatTraceMoeEpisode(raw json.RawMessage) string {
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

func convertTraceMoeResult(r traceMoeAPIResult) TraceMoeResult {
	return TraceMoeResult{
		AnilistID:    r.Anilist.ID,
		MalID:        r.Anilist.IDMal,
		TitleRomaji:  r.Anilist.Title.Romaji,
		TitleEnglish: r.Anilist.Title.English,
		TitleNative:  r.Anilist.Title.Native,
		Episode:      formatTraceMoeEpisode(r.Episode),
		From:         r.From,
		To:           r.To,
		Similarity:   r.Similarity,
		ImageURL:     r.Image,
		VideoURL:     r.Video,
	}
}

// --- exported bridge function ---

// IdentifyAnimeFromImage uploads a screenshot (raw image bytes, e.g. picked
// from the device gallery) to trace.moe and returns a JSON array of ranked
// TraceMoeResult matches (best similarity first).
func IdentifyAnimeFromImage(imageBytes []byte) (string, error) {
	if len(imageBytes) == 0 {
		return "", fmt.Errorf("image cannot be empty")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "screenshot.jpg")
	if err != nil {
		return "", fmt.Errorf("failed to build upload: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return "", fmt.Errorf("failed to build upload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to build upload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, traceMoeSearchURL, &body) // #nosec G107 -- traceMoeSearchURL is a fixed internal endpoint, only overridden by tests
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := traceMoeHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("trace.moe request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read trace.moe response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusPaymentRequired {
		return "", fmt.Errorf("trace.moe search quota exceeded (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("trace.moe returned status %d", resp.StatusCode)
	}

	var apiResp traceMoeAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse trace.moe response: %w", err)
	}
	if apiResp.Error != "" {
		return "", fmt.Errorf("trace.moe error: %s", apiResp.Error)
	}

	results := make([]TraceMoeResult, 0, len(apiResp.Result))
	for _, r := range apiResp.Result {
		results = append(results, convertTraceMoeResult(r))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to serialize results: %w", err)
	}
	return string(data), nil
}
