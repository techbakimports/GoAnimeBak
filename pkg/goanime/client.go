// Package goanime provides a public API for anime scraping and searching functionality.
// This package can be used as a library in other Go projects.
package goanime

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/alvarorichard/Goanime/pkg/goanime/types"
)

// Client is the main client for interacting with anime sources
type Client struct {
	manager *scraper.ScraperManager
}

// NewClient creates a new GoAnime client with all available scrapers
func NewClient() *Client {
	return &Client{
		manager: scraper.NewScraperManager(),
	}
}

// SearchAnime searches for anime across all sources or a specific source.
// If source is nil, searches all available sources.
// Returns a list of anime results or an error.
func (c *Client) SearchAnime(query string, source *types.Source) ([]*types.Anime, error) {
	var scraperType *scraper.ScraperType
	if source != nil {
		st := source.ToScraperType()
		scraperType = &st
	}

	results, err := c.manager.SearchAnime(query, scraperType)
	if err != nil {
		return nil, err
	}

	// Convert internal models to public types
	return types.FromInternalAnimeList(results), nil
}

// GetAnimeEpisodes retrieves all episodes for a specific anime.
// The animeURL should be obtained from a SearchAnime result.
// For SuperFlix TV shows, returns all episodes from all seasons (S01E01 format).
func (c *Client) GetAnimeEpisodes(animeURL string, source types.Source) ([]*types.Episode, error) {
	// SuperFlix requires special handling — the unified scraper adapter
	// cannot fetch episodes directly (it needs TMDB ID + season logic).
	if source == types.SourceSuperFlix {
		return c.getSuperFlixEpisodesAll(animeURL)
	}

	scr, err := c.manager.GetScraper(source.ToScraperType())
	if err != nil {
		return nil, err
	}

	episodes, err := scr.GetAnimeEpisodes(animeURL)
	if err != nil {
		return nil, err
	}

	// For AllAnime, we need to store the anime ID in episodes for later stream URL retrieval
	if source == types.SourceAllAnime {
		for i := range episodes {
			episodes[i].URL = animeURL // Store anime ID in URL field
		}
	}

	return types.FromInternalEpisodeList(episodes), nil
}

// getSuperFlixEpisodesAll fetches all episodes from all seasons for a SuperFlix
// TV show without requiring TUI interaction (for library/Android usage).
// For movies, returns a single episode representing the film.
func (c *Client) getSuperFlixEpisodesAll(tmdbID string) ([]*types.Episode, error) {
	sfClient := scraper.NewSuperFlixClient()

	// Try TV show first: fetch all seasons/episodes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allEpisodes, err := sfClient.GetEpisodes(ctx, tmdbID)
	if err != nil || len(allEpisodes) == 0 {
		// Likely a movie — return a single "episode"
		return []*types.Episode{
			{
				Number: "1",
				Num:    1,
				URL:    tmdbID,
			},
		}, nil
	}

	// Sort seasons numerically
	var seasonNums []string
	for k := range allEpisodes {
		seasonNums = append(seasonNums, k)
	}
	sort.Strings(seasonNums)

	// Flatten all seasons into a single episode list with S01E01 format
	var episodes []models.Episode
	globalNum := 1
	for _, season := range seasonNums {
		epList := allEpisodes[season]
		for _, ep := range epList {
			epNum := ep.EpiNum.String()
			num := 0
			if n, err2 := ep.EpiNum.Int64(); err2 == nil {
				num = int(n)
			}

			episodes = append(episodes, models.Episode{
				Number:   fmt.Sprintf("S%sE%s", season, epNum),
				Num:      globalNum,
				URL:      tmdbID,
				SeasonID: season,
				Title: models.TitleDetails{
					English: ep.Title,
					Romaji:  ep.Title,
				},
				Aired: ep.AirDate,
			})
			globalNum++
			_ = num
		}
	}

	return types.FromInternalEpisodeList(episodes), nil
}

// GetStreamURL retrieves the streaming URL and headers for a specific episode.
// The episodeURL should be obtained from GetAnimeEpisodes.
//
// Deprecated: Use GetEpisodeStreamURL instead for better control over quality and mode.
func (c *Client) GetStreamURL(episodeURL string, source types.Source, options ...any) (string, map[string]string, error) {
	scr, err := c.manager.GetScraper(source.ToScraperType())
	if err != nil {
		return "", nil, err
	}

	return scr.GetStreamURL(episodeURL, options...)
}

// StreamOptions contains options for retrieving stream URLs
type StreamOptions struct {
	// Quality can be "best", "worst", "1080p", "720p", "480p", "360p"
	Quality string
	// Mode can be "sub" (subtitled) or "dub" (dubbed)
	Mode string
}

// DefaultStreamOptions returns default stream options
func DefaultStreamOptions() StreamOptions {
	return StreamOptions{
		Quality: "best",
		Mode:    "sub",
	}
}

// GetEpisodeStreamURL retrieves the streaming URL for a specific episode with full control.
// This is the recommended method to get playback URLs.
//
// Parameters:
//   - anime: The anime object from SearchAnime
//   - episode: The episode object from GetAnimeEpisodes
//   - options: Optional StreamOptions (uses defaults if nil)
//
// Returns:
//   - streamURL: Direct URL for video playback
//   - metadata: Additional info like quality, source, etc.
//   - error: Any error that occurred
func (c *Client) GetEpisodeStreamURL(anime *types.Anime, episode *types.Episode, options *StreamOptions) (string, map[string]string, error) {
	source, err := types.ParseSource(anime.Source)
	if err != nil {
		return "", nil, err
	}

	scr, err := c.manager.GetScraper(source.ToScraperType())
	if err != nil {
		return "", nil, err
	}

	// Set default options if not provided
	opts := DefaultStreamOptions()
	if options != nil {
		if options.Quality != "" {
			opts.Quality = options.Quality
		}
		if options.Mode != "" {
			opts.Mode = options.Mode
		}
	}

	// For AllAnime, we need to pass: animeID (URL), episodeNumber, quality, mode
	if source == types.SourceAllAnime {
		return scr.GetStreamURL(anime.URL, episode.Number, opts.Quality, opts.Mode)
	}

	// For SuperFlix, we need: tmdbID, mediaType, season, episodeNumber
	if source == types.SourceSuperFlix {
		tmdbID := episode.URL
		if tmdbID == "" {
			tmdbID = anime.URL
		}
		// Determine if movie or serie based on SeasonID presence
		if episode.SeasonID == "" {
			// Movie
			return scr.GetStreamURL(tmdbID, "filme", "", "")
		}
		// TV show — extract episode number from "S01E03" format or use raw number
		epNum := episode.Number
		if len(epNum) > 4 && epNum[0] == 'S' {
			// Parse S01E03 → season="1", episode="3"
			for i := 1; i < len(epNum); i++ {
				if epNum[i] == 'E' {
					epNum = epNum[i+1:]
					break
				}
			}
		}
		return scr.GetStreamURL(tmdbID, "serie", episode.SeasonID, epNum)
	}

	// All other sources use the episode URL directly
	return scr.GetStreamURL(episode.URL)
}

// GetAvailableSources returns a list of all available scraper sources.
func (c *Client) GetAvailableSources() []types.Source {
	return []types.Source{
		types.SourceAllAnime,
		types.SourceAnimeFire,
		types.SourceGoyabu,
		types.SourceSuperFlix,
		types.SourceHiAnime,
		types.SourceGogoAnime,
		types.SourceAniNeko,
	}
}
