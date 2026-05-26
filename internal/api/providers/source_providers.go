package providers

import (
	"context"
	"fmt"

	"github.com/alvarorichard/Goanime/internal/api/source"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
)

// EpisodeNumber extracts the episode number string from an Episode model.
// Returns "" if indeterminate — caller must decide how to handle.
func EpisodeNumber(ep *models.Episode) string {
	if ep == nil {
		return ""
	}
	if ep.Number != "" {
		return ep.Number
	}
	if ep.Num > 0 {
		return fmt.Sprintf("%d", ep.Num)
	}
	return ""
}

// --- AllAnime Provider ---

type allAnimeProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.AllAnime, func(sm *scraper.ScraperManager) Provider {
		return &allAnimeProvider{sm: sm}
	})
}

func (p *allAnimeProvider) Kind() source.SourceKind { return source.AllAnime }
func (p *allAnimeProvider) HasSeasons() bool        { return false }

func (p *allAnimeProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.AllAnimeType)
	if err != nil {
		return nil, err
	}
	animeID := source.ExtractAllAnimeID(anime.URL)
	return adapter.GetAnimeEpisodes(animeID)
}

func (p *allAnimeProvider) FetchStreamURL(_ context.Context, episode *models.Episode, anime *models.Anime, quality string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.AllAnimeType)
	if err != nil {
		return "", err
	}
	animeID := source.ExtractAllAnimeID(anime.URL)
	epNum := EpisodeNumber(episode)
	if quality == "" {
		quality = "best"
	}
	url, _, err := adapter.GetStreamURL(animeID, epNum, quality)
	if err != nil {
		return "", fmt.Errorf("allAnime stream: %w", err)
	}
	return url, nil
}

// --- AnimeFire Provider ---

type animeFireProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.AnimeFire, func(sm *scraper.ScraperManager) Provider {
		return &animeFireProvider{sm: sm}
	})
}

func (p *animeFireProvider) Kind() source.SourceKind { return source.AnimeFire }
func (p *animeFireProvider) HasSeasons() bool        { return false }

func (p *animeFireProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimefireType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *animeFireProvider) FetchStreamURL(_ context.Context, episode *models.Episode, anime *models.Anime, quality string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimefireType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("animeFire stream: %w", err)
	}
	return url, nil
}

// --- Goyabu Provider ---

type goyabuProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.Goyabu, func(sm *scraper.ScraperManager) Provider {
		return &goyabuProvider{sm: sm}
	})
}

func (p *goyabuProvider) Kind() source.SourceKind { return source.Goyabu }
func (p *goyabuProvider) HasSeasons() bool        { return false }

func (p *goyabuProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.GoyabuType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *goyabuProvider) FetchStreamURL(_ context.Context, episode *models.Episode, anime *models.Anime, quality string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.GoyabuType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("goyabu stream: %w", err)
	}
	return url, nil
}

// --- HiAnime Provider ---

type hiAnimeProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.HiAnime, func(sm *scraper.ScraperManager) Provider {
		return &hiAnimeProvider{sm: sm}
	})
}

func (p *hiAnimeProvider) Kind() source.SourceKind { return source.HiAnime }
func (p *hiAnimeProvider) HasSeasons() bool        { return false }

func (p *hiAnimeProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.HiAnimeType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *hiAnimeProvider) FetchStreamURL(_ context.Context, episode *models.Episode, _ *models.Anime, _ string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.HiAnimeType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("hianime stream: %w", err)
	}
	return url, nil
}

// --- GogoAnime Provider ---

type gogoAnimeProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.GogoAnime, func(sm *scraper.ScraperManager) Provider {
		return &gogoAnimeProvider{sm: sm}
	})
}

func (p *gogoAnimeProvider) Kind() source.SourceKind { return source.GogoAnime }
func (p *gogoAnimeProvider) HasSeasons() bool        { return false }

func (p *gogoAnimeProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.GogoAnimeType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *gogoAnimeProvider) FetchStreamURL(_ context.Context, episode *models.Episode, _ *models.Anime, _ string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.GogoAnimeType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("gogoanime stream: %w", err)
	}
	return url, nil
}

// --- AniNeko Provider ---

type aniNekoProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.AniNeko, func(sm *scraper.ScraperManager) Provider {
		return &aniNekoProvider{sm: sm}
	})
}

func (p *aniNekoProvider) Kind() source.SourceKind { return source.AniNeko }
func (p *aniNekoProvider) HasSeasons() bool        { return false }

func (p *aniNekoProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.AniNekoType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *aniNekoProvider) FetchStreamURL(_ context.Context, episode *models.Episode, _ *models.Anime, _ string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.AniNekoType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("anineko stream: %w", err)
	}
	return url, nil
}

// --- SuperFlix Provider ---

type superFlixProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.SuperFlix, func(sm *scraper.ScraperManager) Provider {
		return &superFlixProvider{sm: sm}
	})
}

func (p *superFlixProvider) Kind() source.SourceKind { return source.SuperFlix }
func (p *superFlixProvider) HasSeasons() bool        { return true }

func (p *superFlixProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.SuperFlixType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *superFlixProvider) FetchStreamURL(_ context.Context, episode *models.Episode, anime *models.Anime, quality string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.SuperFlixType)
	if err != nil {
		return "", err
	}
	epNum := EpisodeNumber(episode)
	mediaType := "serie"
	if anime.MediaType == models.MediaTypeMovie {
		mediaType = "filme"
	}
	season := "1"
	if anime.CurrentSeason > 0 {
		season = fmt.Sprintf("%d", anime.CurrentSeason)
	}
	url, _, err := adapter.GetStreamURL(episode.URL, mediaType, season, epNum)
	if err != nil {
		return "", fmt.Errorf("superFlix stream: %w", err)
	}
	return url, nil
}

