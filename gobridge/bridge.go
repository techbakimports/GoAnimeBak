// Package gobridge provides a JSON-based bridge for gomobile.
// gomobile cannot export slices of structs, maps, or complex interfaces.
// This package wraps pkg/goanime with JSON serialization so that
// Android (Kotlin/Java) can call Go functions with simple string I/O.
package gobridge

import (
	"encoding/json"
	"fmt"
	"sync"

	goanime "github.com/alvarorichard/Goanime/pkg/goanime"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

// client is a lazy-initialized singleton
var (
	client     *goanime.Client
	clientOnce sync.Once
)

func getClient() *goanime.Client {
	clientOnce.Do(func() {
		client = goanime.NewClient()
	})
	return client
}

// --- JSON response types ---

// AnimeResult is the JSON-friendly anime representation for the bridge.
type AnimeResult struct {
	Name      string         `json:"name"`
	URL       string         `json:"url"`
	ImageURL  string         `json:"imageUrl"`
	Source    string         `json:"source"`
	AnilistID int            `json:"anilistId,omitempty"`
	MalID     int            `json:"malId,omitempty"`
	Details   *AnimeDetails  `json:"details,omitempty"`
}

// AnimeDetails contains extended metadata.
type AnimeDetails struct {
	Description  string   `json:"description,omitempty"`
	Genres       []string `json:"genres,omitempty"`
	AverageScore int      `json:"averageScore,omitempty"`
	Episodes     int      `json:"episodes,omitempty"`
	Status       string   `json:"status,omitempty"`
	CoverLarge   string   `json:"coverLarge,omitempty"`
	CoverMedium  string   `json:"coverMedium,omitempty"`
}

// EpisodeResult is the JSON-friendly episode representation.
type EpisodeResult struct {
	Number    string     `json:"number"`
	Num       int        `json:"num"`
	URL       string     `json:"url"`
	Title     string     `json:"title,omitempty"`
	TitleJP   string     `json:"titleJp,omitempty"`
	Aired     string     `json:"aired,omitempty"`
	Duration  int        `json:"duration,omitempty"`
	IsFiller  bool       `json:"isFiller,omitempty"`
	IsRecap   bool       `json:"isRecap,omitempty"`
	Synopsis  string     `json:"synopsis,omitempty"`
	SkipOpStart int      `json:"skipOpStart,omitempty"`
	SkipOpEnd   int      `json:"skipOpEnd,omitempty"`
	SkipEdStart int      `json:"skipEdStart,omitempty"`
	SkipEdEnd   int      `json:"skipEdEnd,omitempty"`
}

// StreamResult is the JSON-friendly stream URL response.
type StreamResult struct {
	URL      string            `json:"url"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SourceResult represents an available source.
type SourceResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- Exported Bridge Functions ---

// SearchAnime searches for anime by query.
// source can be "" (all sources), "AllAnime", or "AnimeFire".
// Returns a JSON array of AnimeResult.
func SearchAnime(query string, source string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	c := getClient()

	var sourcePtr *types.Source
	if source != "" {
		s, err := types.ParseSource(source)
		if err != nil {
			return "", fmt.Errorf("invalid source %q: %w", source, err)
		}
		sourcePtr = &s
	}

	results, err := c.SearchAnime(query, sourcePtr)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	out := make([]AnimeResult, 0, len(results))
	for _, a := range results {
		r := AnimeResult{
			Name:      a.Name,
			URL:       a.URL,
			ImageURL:  a.ImageURL,
			Source:    a.Source,
			AnilistID: a.AnilistID,
			MalID:     a.MalID,
		}
		if a.Details != nil {
			r.Details = &AnimeDetails{
				Description:  a.Details.Description,
				Genres:       a.Details.Genres,
				AverageScore: a.Details.AverageScore,
				Episodes:     a.Details.Episodes,
				Status:       a.Details.Status,
			}
			if a.Details.CoverImage != nil {
				r.Details.CoverLarge = a.Details.CoverImage.Large
				r.Details.CoverMedium = a.Details.CoverImage.Medium
			}
		}
		out = append(out, r)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("json marshal failed: %w", err)
	}
	return string(data), nil
}

// GetEpisodes retrieves episodes for a given anime URL and source.
// Returns a JSON array of EpisodeResult.
func GetEpisodes(animeURL string, source string) (string, error) {
	if animeURL == "" {
		return "", fmt.Errorf("animeURL cannot be empty")
	}
	if source == "" {
		return "", fmt.Errorf("source cannot be empty")
	}

	c := getClient()

	s, err := types.ParseSource(source)
	if err != nil {
		return "", fmt.Errorf("invalid source %q: %w", source, err)
	}

	episodes, err := c.GetAnimeEpisodes(animeURL, s)
	if err != nil {
		return "", fmt.Errorf("get episodes failed: %w", err)
	}

	out := make([]EpisodeResult, 0, len(episodes))
	for _, ep := range episodes {
		r := EpisodeResult{
			Number:   ep.Number,
			Num:      ep.Num,
			URL:      ep.URL,
			Aired:    ep.Aired,
			Duration: ep.Duration,
			IsFiller: ep.IsFiller,
			IsRecap:  ep.IsRecap,
			Synopsis: ep.Synopsis,
		}
		if ep.Title != nil {
			r.Title = ep.Title.English
			if r.Title == "" {
				r.Title = ep.Title.Romaji
			}
			r.TitleJP = ep.Title.Japanese
		}
		if ep.SkipTimes != nil {
			if ep.SkipTimes.Op != nil {
				r.SkipOpStart = ep.SkipTimes.Op.Start
				r.SkipOpEnd = ep.SkipTimes.Op.End
			}
			if ep.SkipTimes.Ed != nil {
				r.SkipEdStart = ep.SkipTimes.Ed.Start
				r.SkipEdEnd = ep.SkipTimes.Ed.End
			}
		}
		out = append(out, r)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("json marshal failed: %w", err)
	}
	return string(data), nil
}

// GetStreamURL retrieves the streaming URL for an episode.
// animeJSON must be a JSON object with at least "url" and "source" fields.
// episodeJSON must be a JSON object with at least "number" and "url" fields.
// quality: "best", "worst", "1080p", "720p", "480p", "360p"
// mode: "sub" or "dub"
// Returns a JSON StreamResult with url and metadata.
func GetStreamURL(animeJSON string, episodeJSON string, quality string, mode string) (string, error) {
	if animeJSON == "" || episodeJSON == "" {
		return "", fmt.Errorf("animeJSON and episodeJSON cannot be empty")
	}

	// Parse anime from JSON
	var animeInput struct {
		URL    string `json:"url"`
		Source string `json:"source"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal([]byte(animeJSON), &animeInput); err != nil {
		return "", fmt.Errorf("invalid animeJSON: %w", err)
	}

	// Parse episode from JSON
	var episodeInput struct {
		Number string `json:"number"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(episodeJSON), &episodeInput); err != nil {
		return "", fmt.Errorf("invalid episodeJSON: %w", err)
	}

	// Reconstruct types for the client
	anime := &types.Anime{
		URL:    animeInput.URL,
		Source: animeInput.Source,
		Name:   animeInput.Name,
	}
	episode := &types.Episode{
		Number: episodeInput.Number,
		URL:    episodeInput.URL,
	}

	opts := &goanime.StreamOptions{
		Quality: quality,
		Mode:    mode,
	}
	if opts.Quality == "" {
		opts.Quality = "best"
	}
	if opts.Mode == "" {
		opts.Mode = "sub"
	}

	c := getClient()
	streamURL, metadata, err := c.GetEpisodeStreamURL(anime, episode, opts)
	if err != nil {
		return "", fmt.Errorf("get stream URL failed: %w", err)
	}

	result := StreamResult{
		URL:      streamURL,
		Metadata: metadata,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("json marshal failed: %w", err)
	}
	return string(data), nil
}

// GetSources returns a JSON array of available sources.
// This function never fails.
func GetSources() string {
	c := getClient()
	sources := c.GetAvailableSources()

	out := make([]SourceResult, 0, len(sources))
	for _, s := range sources {
		out = append(out, SourceResult{
			ID:   s.String(),
			Name: s.String(),
		})
	}

	data, _ := json.Marshal(out)
	return string(data)
}

// Version returns the bridge version for debugging.
func Version() string {
	return "gobridge/1.0.0"
}
