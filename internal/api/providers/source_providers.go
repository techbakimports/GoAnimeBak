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

// --- AnimesOnlineCC Provider ---

type animesOnlineCCProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.AnimesOnlineCC, func(sm *scraper.ScraperManager) Provider {
		return &animesOnlineCCProvider{sm: sm}
	})
}

func (p *animesOnlineCCProvider) Kind() source.SourceKind { return source.AnimesOnlineCC }
func (p *animesOnlineCCProvider) HasSeasons() bool        { return false }

func (p *animesOnlineCCProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimesOnlineCCType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *animesOnlineCCProvider) FetchStreamURL(_ context.Context, episode *models.Episode, _ *models.Anime, _ string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimesOnlineCCType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("animesonlinecc stream: %w", err)
	}
	return url, nil
}

// --- AnimeHeaven Provider ---

type animeHeavenProvider struct {
	sm *scraper.ScraperManager
}

func init() {
	RegisterProvider(source.AnimeHeaven, func(sm *scraper.ScraperManager) Provider {
		return &animeHeavenProvider{sm: sm}
	})
}

func (p *animeHeavenProvider) Kind() source.SourceKind { return source.AnimeHeaven }
func (p *animeHeavenProvider) HasSeasons() bool        { return false }

func (p *animeHeavenProvider) FetchEpisodes(_ context.Context, anime *models.Anime) ([]models.Episode, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimeHeavenType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAnimeEpisodes(anime.URL)
}

func (p *animeHeavenProvider) FetchStreamURL(_ context.Context, episode *models.Episode, _ *models.Anime, _ string) (string, error) {
	adapter, err := p.sm.GetScraper(scraper.AnimeHeavenType)
	if err != nil {
		return "", err
	}
	url, _, err := adapter.GetStreamURL(episode.URL)
	if err != nil {
		return "", fmt.Errorf("animeheaven stream: %w", err)
	}
	return url, nil
}
