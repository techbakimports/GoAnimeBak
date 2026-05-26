package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/api/providers/metadata"
	"github.com/alvarorichard/Goanime/internal/appflow"
	"github.com/alvarorichard/Goanime/internal/downloader"
	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/internal/util"
)

// HandleDownloadRequest processes download requests
func HandleDownloadRequest() error {
	util.InitLogger()

	if util.GlobalDownloadRequest == nil {
		return fmt.Errorf("download request is nil")
	}

	request := util.GlobalDownloadRequest
	source := request.Source
	quality := request.Quality
	if quality == "" {
		quality = "best"
	}

	util.Infof("Using source: %s, quality: %s", source, quality)

	anime, err := appflow.SearchAnimeWithRetry(request.AnimeName)
	if err != nil {
		return fmt.Errorf("failed to search for anime: %w", err)
	}

	season := 1
	if request.SeasonNum > 0 {
		season = request.SeasonNum
	}
	player.SetAnimeName(anime.Name, season)
	player.SetExactMediaType(string(anime.MediaType))

	player.SetMediaMeta(&util.MediaMeta{
		OfficialTitle: anime.OfficialTitle(),
		Year:          anime.Year,
		TMDBID:        anime.TMDBID,
		IMDBID:        anime.IMDBID,
		AnilistID:     anime.AnilistID,
		MalID:         anime.MalID,
	})

	enricher := metadata.NewEnricher()
	seasonMap, _ := enricher.EnrichAnime(context.Background(), anime)
	player.SetSeasonMap(seasonMap)

	player.SetMediaMeta(&util.MediaMeta{
		OfficialTitle: anime.OfficialTitle(),
		Year:          anime.Year,
		TMDBID:        anime.TMDBID,
		IMDBID:        anime.IMDBID,
		AnilistID:     anime.AnilistID,
		MalID:         anime.MalID,
	})

	if request.IsAll {
		util.Infof("Downloading ALL episodes of %s", anime.Name)
		eps, err := api.GetAnimeEpisodesEnhanced(anime)
		if err == nil && len(eps) > 0 {
			dlErr := player.HandleBatchDownload(eps, anime)
			if dlErr == nil || errors.Is(dlErr, player.ErrUserQuit) {
				return nil
			}
			util.Infof("Batch download path failed, falling back to legacy: %v", dlErr)
		}
		episodes, legacyErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
		if legacyErr != nil {
			return fmt.Errorf("failed to fetch episodes: %w", legacyErr)
		}
		dl := downloader.NewEpisodeDownloaderWithAnime(episodes, anime.URL, anime)
		return dl.DownloadAllEpisodes()
	}

	if request.IsRange {
		util.Infof("Downloading episodes %d-%d of %s", request.StartEpisode, request.EndEpisode, anime.Name)

		if request.AllAnimeSmart && (anime.Source == "AllAnime" || source == "allanime" || source == "AllAnime") {
			eps, err := api.GetAnimeEpisodesEnhanced(anime)
			if err == nil && len(eps) > 0 {
				dlErr := player.HandleBatchDownloadRange(eps, anime, request.StartEpisode, request.EndEpisode)
				if dlErr == nil || errors.Is(dlErr, player.ErrUserQuit) {
					return nil
				}
			}
			if err := api.DownloadAllAnimeSmartRange(anime, request.StartEpisode, request.EndEpisode, quality); err != nil {
				if err := api.DownloadEpisodeRangeEnhanced(anime, request.StartEpisode, request.EndEpisode, quality); err != nil {
					episodes, legacyErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
					if legacyErr != nil {
						return fmt.Errorf("legacy episode fetch also failed: %w", legacyErr)
					}
					dl := downloader.NewEpisodeDownloaderWithAnime(episodes, anime.URL, anime)
					return dl.DownloadEpisodeRange(request.StartEpisode, request.EndEpisode)
				}
				return nil
			}
			return nil
		}

		eps, err := api.GetAnimeEpisodesEnhanced(anime)
		if err == nil && len(eps) > 0 {
			dlErr := player.HandleBatchDownloadRange(eps, anime, request.StartEpisode, request.EndEpisode)
			if dlErr == nil || errors.Is(dlErr, player.ErrUserQuit) {
				return nil
			}
		}
		episodes, legacyErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
		if legacyErr != nil {
			return fmt.Errorf("failed to fetch episodes: %w", legacyErr)
		}
		dl := downloader.NewEpisodeDownloaderWithAnime(episodes, anime.URL, anime)
		return dl.DownloadEpisodeRange(request.StartEpisode, request.EndEpisode)
	}

	util.Infof("Downloading episode %d of %s", request.EpisodeNum, anime.Name)
	episodes, legacyErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
	if legacyErr != nil {
		return fmt.Errorf("failed to fetch episodes: %w", legacyErr)
	}
	dl := downloader.NewEpisodeDownloaderWithAnime(episodes, anime.URL, anime)
	return dl.DownloadSingleEpisode(request.EpisodeNum)
}
