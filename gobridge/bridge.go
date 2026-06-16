// Package gobridge provides a JSON-based bridge for gomobile.
// gomobile cannot export slices of structs, maps, or complex interfaces.
// This package wraps pkg/goanime with JSON serialization so that
// Android (Kotlin/Java) can call Go functions with simple string I/O.
package gobridge

import (
	"encoding/json"
	"fmt"
	"strings"
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
	SeasonID  string     `json:"seasonId,omitempty"`
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

// FallbackEpisodesResult is returned by GetEpisodesWithFallback.
type FallbackEpisodesResult struct {
	Episodes  []EpisodeResult `json:"episodes"`
	Source    string          `json:"source"`
	AnimeURL  string          `json:"animeUrl"`
	AnimeName string          `json:"animeName"`
}

// defaultSourceOrder is the priority used when no explicit list is given.
var defaultSourceOrder = []types.Source{
	types.SourceAllAnime,
	types.SourceAnimeFire,
	types.SourceGoyabu,
	types.SourceGogoAnime,
	types.SourceAnimesOnlineCC,
	types.SourceAnimeHeaven,
}

// convertEpisode maps a types.Episode to the bridge EpisodeResult.
func convertEpisode(ep *types.Episode) EpisodeResult {
	r := EpisodeResult{
		Number:   ep.Number,
		Num:      ep.Num,
		URL:      ep.URL,
		Aired:    ep.Aired,
		Duration: ep.Duration,
		IsFiller: ep.IsFiller,
		IsRecap:  ep.IsRecap,
		Synopsis: ep.Synopsis,
		SeasonID: ep.SeasonID,
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
	return r
}

// parseSources parses a JSON array of source name strings into []types.Source.
// Returns defaultSourceOrder on empty input or parse failure.
func parseSources(sourcesJSON string) []types.Source {
	if sourcesJSON == "" || sourcesJSON == "[]" || sourcesJSON == "null" {
		return defaultSourceOrder
	}
	var names []string
	if err := json.Unmarshal([]byte(sourcesJSON), &names); err != nil || len(names) == 0 {
		return defaultSourceOrder
	}
	result := make([]types.Source, 0, len(names))
	for _, n := range names {
		if s, err := types.ParseSource(n); err == nil {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return defaultSourceOrder
	}
	return result
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

	out := make([]EpisodeResult, len(episodes))
	for i, ep := range episodes {
		out[i] = convertEpisode(ep)
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
		Number   string `json:"number"`
		URL      string `json:"url"`
		SeasonID string `json:"seasonId"`
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
		Number:   episodeInput.Number,
		URL:      episodeInput.URL,
		SeasonID: episodeInput.SeasonID,
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

// GetEpisodesWithFallback searches for animeName across sources in priority order
// and returns episodes from the first source that delivers results.
// sourcesJSON: JSON array like ["AnimeFire","Goyabu","AllAnime"], or "" for default order.
// Returns a JSON FallbackEpisodesResult that includes which source was used.
func GetEpisodesWithFallback(animeName string, sourcesJSON string) (string, error) {
	if animeName == "" {
		return "", fmt.Errorf("animeName cannot be empty")
	}

	sources := parseSources(sourcesJSON)
	c := getClient()
	var errs []string

	for _, src := range sources {
		srcPtr := src
		results, err := c.SearchAnime(animeName, &srcPtr)
		if err != nil || len(results) == 0 {
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s search: %v", src, err))
			} else {
				errs = append(errs, fmt.Sprintf("%s: no results", src))
			}
			continue
		}

		anime := results[0]
		episodes, err := c.GetAnimeEpisodes(anime.URL, src)
		if err != nil || len(episodes) == 0 {
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s episodes: %v", src, err))
			} else {
				errs = append(errs, fmt.Sprintf("%s: no episodes", src))
			}
			continue
		}

		out := make([]EpisodeResult, len(episodes))
		for i, ep := range episodes {
			out[i] = convertEpisode(ep)
		}

		result := FallbackEpisodesResult{
			Episodes:  out,
			Source:    src.String(),
			AnimeURL:  anime.URL,
			AnimeName: anime.Name,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("json marshal failed: %w", err)
		}
		return string(data), nil
	}

	return "", fmt.Errorf("all sources failed: %s", strings.Join(errs, "; "))
}

// GetStreamURLWithFallback tries to get a stream URL from the primary source
// (encoded in animeJSON) and falls back through fallbackSourcesJSON on failure.
// animeJSON: {"url":"...", "source":"...", "name":"..."}
// episodeJSON: {"number":"...", "url":"..."}
// fallbackSourcesJSON: JSON array like ["AnimeFire","Goyabu"] tried in order after primary fails.
// Returns a JSON StreamResult.
func GetStreamURLWithFallback(animeJSON, episodeJSON, quality, mode, fallbackSourcesJSON string) (string, error) {
	// Try primary first
	if result, err := GetStreamURL(animeJSON, episodeJSON, quality, mode); err == nil {
		return result, nil
	}

	var animeInput struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	}
	var episodeInput struct {
		Number string `json:"number"`
	}
	if err := json.Unmarshal([]byte(animeJSON), &animeInput); err != nil || animeInput.Name == "" {
		return "", fmt.Errorf("primary source failed and animeJSON is missing name field")
	}
	if err := json.Unmarshal([]byte(episodeJSON), &episodeInput); err != nil || episodeInput.Number == "" {
		return "", fmt.Errorf("primary source failed and episodeJSON is missing number field")
	}

	fallbackSources := parseSources(fallbackSourcesJSON)
	c := getClient()
	var errs []string

	for _, src := range fallbackSources {
		if src.String() == animeInput.Source {
			continue // already failed
		}

		srcPtr := src
		searchResults, err := c.SearchAnime(animeInput.Name, &srcPtr)
		if err != nil || len(searchResults) == 0 {
			errs = append(errs, fmt.Sprintf("%s: search failed", src))
			continue
		}

		fallbackAnime := searchResults[0]
		episodes, err := c.GetAnimeEpisodes(fallbackAnime.URL, src)
		if err != nil || len(episodes) == 0 {
			errs = append(errs, fmt.Sprintf("%s: no episodes", src))
			continue
		}

		var matchedEp *types.Episode
		for _, ep := range episodes {
			if ep.Number == episodeInput.Number {
				matchedEp = ep
				break
			}
		}
		if matchedEp == nil {
			errs = append(errs, fmt.Sprintf("%s: episode %s not found", src, episodeInput.Number))
			continue
		}

		fbAnimeBytes, err := json.Marshal(map[string]string{
			"url":    fallbackAnime.URL,
			"source": src.String(),
			"name":   fallbackAnime.Name,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: marshal anime: %v", src, err))
			continue
		}
		fbEpBytes, err := json.Marshal(map[string]string{
			"number":   matchedEp.Number,
			"url":      matchedEp.URL,
			"seasonId": matchedEp.SeasonID,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: marshal episode: %v", src, err))
			continue
		}

		if result, err := GetStreamURL(string(fbAnimeBytes), string(fbEpBytes), quality, mode); err == nil {
			return result, nil
		} else {
			errs = append(errs, fmt.Sprintf("%s: stream: %v", src, err))
		}
	}

	return "", fmt.Errorf("all fallback sources failed: %s", strings.Join(errs, "; "))
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

	data, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// Version returns the bridge version for debugging.
func Version() string {
	return "gobridge/1.0.0"
}
