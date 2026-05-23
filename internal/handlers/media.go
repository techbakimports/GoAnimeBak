// Package handlers provides HTTP handlers and flow controllers for media playback
package handlers

import (
	"context"
	"fmt"
	"strings"

	"strconv"

	"charm.land/huh/v2"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/alvarorichard/Goanime/internal/tui"
	"github.com/alvarorichard/Goanime/internal/util"
)

// MediaHandler handles media selection and playback operations
type MediaHandler struct {
	mediaManager *scraper.MediaManager
	provider     string
	quality      scraper.Quality
	subsLanguage string
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler() *MediaHandler {
	return &MediaHandler{
		mediaManager: scraper.NewMediaManager(),
		provider:     "Vidcloud",
		quality:      scraper.Quality1080,
		subsLanguage: "english",
	}
}

// SetOptions sets playback options
func (mh *MediaHandler) SetOptions(provider, quality, subsLanguage string) {
	if provider != "" {
		mh.provider = provider
	}
	if quality != "" {
		mh.quality = scraper.Quality(quality)
	}
	if subsLanguage != "" {
		mh.subsLanguage = subsLanguage
	}
}

// SearchMedia searches for media based on content type
func (mh *MediaHandler) SearchMedia(query string, contentType models.MediaType) ([]*models.Anime, error) {
	switch contentType {
	case models.MediaTypeAnime:
		return mh.mediaManager.SearchAnimeOnly(query)
	case models.MediaTypeMovie, models.MediaTypeTV:
		media, err := mh.mediaManager.SearchMoviesAndTV(query)
		if err != nil {
			return nil, err
		}
		return scraper.ConvertFlixHQToAnime(media), nil
	default:
		return mh.mediaManager.SearchAll(query)
	}
}

// SelectMediaType prompts user to select media type
func (mh *MediaHandler) SelectMediaType() (models.MediaType, error) {
	items := []tui.MenuItem{
		{Label: "Anime", Value: "anime"},
		{Label: "Movies", Value: "movies"},
		{Label: "TV Shows", Value: "tv"},
		{Label: "Search All", Value: "all"},
	}
	switch tui.RunMenu("Select content type", items) {
	case "anime":
		return models.MediaTypeAnime, nil
	case "movies":
		return models.MediaTypeMovie, nil
	case "tv":
		return models.MediaTypeTV, nil
	case "all":
		return "", nil
	default:
		return "", fmt.Errorf("selection cancelled")
	}
}

// SelectMedia prompts user to select from search results
func (mh *MediaHandler) SelectMedia(results []*models.Anime) (*models.Anime, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to select from")
	}

	items := make([]tui.MenuItem, len(results))
	for i, r := range results {
		typeTag := ""
		switch r.MediaType {
		case models.MediaTypeMovie:
			typeTag = "[Movie]"
		case models.MediaTypeTV:
			typeTag = "[TV]"
		case models.MediaTypeAnime:
			typeTag = "[Anime]"
		}
		year := ""
		if r.Year != "" {
			year = fmt.Sprintf(" (%s)", r.Year)
		}
		items[i] = tui.MenuItem{
			Label: fmt.Sprintf("%s %s%s - %s", typeTag, r.Name, year, r.Source),
			Value: strconv.Itoa(i),
		}
	}

	choice := tui.RunMenu("Select media", items)
	if choice == "q" || choice == "back" {
		return nil, fmt.Errorf("selection cancelled")
	}
	idx, _ := strconv.Atoi(choice)
	return results[idx], nil
}

// SelectSeason prompts user to select a TV season
func (mh *MediaHandler) SelectSeason(mediaID string) (*scraper.FlixHQSeason, error) {
	seasons, err := mh.mediaManager.GetTVSeasons(mediaID)
	if err != nil {
		return nil, err
	}

	if len(seasons) == 0 {
		return nil, fmt.Errorf("no seasons found")
	}

	seasonItems := make([]tui.MenuItem, len(seasons))
	for i, s := range seasons {
		seasonItems[i] = tui.MenuItem{Label: s.Title, Value: strconv.Itoa(i)}
	}
	choice := tui.RunMenu("Select season", seasonItems)
	if choice == "q" || choice == "back" {
		return nil, fmt.Errorf("selection cancelled")
	}
	idx, _ := strconv.Atoi(choice)
	return &seasons[idx], nil
}

// SelectEpisode prompts user to select a TV episode
func (mh *MediaHandler) SelectEpisode(seasonID string) (*scraper.FlixHQEpisode, error) {
	episodes, err := mh.mediaManager.GetTVEpisodes(seasonID)
	if err != nil {
		return nil, err
	}

	if len(episodes) == 0 {
		return nil, fmt.Errorf("no episodes found")
	}

	epItems := make([]tui.MenuItem, len(episodes))
	for i, ep := range episodes {
		epItems[i] = tui.MenuItem{
			Label: fmt.Sprintf("Episode %d: %s", ep.Number, ep.Title),
			Value: strconv.Itoa(i),
		}
	}
	epChoice := tui.RunMenu("Select episode", epItems)
	if epChoice == "q" || epChoice == "back" {
		return nil, fmt.Errorf("selection cancelled")
	}
	idx, _ := strconv.Atoi(epChoice)
	return &episodes[idx], nil
}

// GetStreamInfo gets streaming information for selected media
func (mh *MediaHandler) GetStreamInfo(media *models.Anime, episode *scraper.FlixHQEpisode) (*scraper.FlixHQStreamInfo, error) {
	return mh.GetStreamInfoWithContext(context.Background(), media, episode)
}

// GetStreamInfoWithContext gets streaming information with context support
func (mh *MediaHandler) GetStreamInfoWithContext(ctx context.Context, media *models.Anime, episode *scraper.FlixHQEpisode) (*scraper.FlixHQStreamInfo, error) {
	source := strings.ToLower(media.Source)

	if !strings.Contains(source, "flixhq") {
		return nil, fmt.Errorf("media source %s does not support FlixHQ streaming", media.Source)
	}

	// Extract media ID from URL
	mediaID := extractIDFromURL(media.URL)
	if mediaID == "" {
		return nil, fmt.Errorf("could not extract media ID from URL: %s", media.URL)
	}

	if media.MediaType == models.MediaTypeMovie {
		return mh.mediaManager.GetMovieStreamInfo(mediaID, mh.provider, string(mh.quality), mh.subsLanguage)
	}

	if episode == nil {
		return nil, fmt.Errorf("episode is required for TV shows")
	}

	return mh.mediaManager.GetTVEpisodeStreamInfo(episode.DataID, mh.provider, string(mh.quality), mh.subsLanguage)
}

// GetStreamWithQuality gets stream info with quality selection
func (mh *MediaHandler) GetStreamWithQuality(episodeID string, isMovie bool) (*scraper.FlixHQStreamInfo, error) {
	return mh.mediaManager.GetStreamWithQuality(episodeID, isMovie, mh.quality, mh.subsLanguage)
}

// GetStreamWithQualityContext gets stream info with quality selection and context
func (mh *MediaHandler) GetStreamWithQualityContext(ctx context.Context, episodeID string, isMovie bool) (*scraper.FlixHQStreamInfo, error) {
	return mh.mediaManager.GetStreamWithQualityWithContext(ctx, episodeID, isMovie, mh.quality, mh.subsLanguage)
}

// SelectQuality prompts user to select video quality
func (mh *MediaHandler) SelectQuality(episodeID string, isMovie bool) (scraper.Quality, error) {
	qualities, err := mh.mediaManager.GetAvailableQualities(episodeID, isMovie)
	if err != nil {
		return scraper.QualityAuto, err
	}

	if len(qualities) == 0 {
		return scraper.QualityAuto, nil
	}

	qualItems := make([]tui.MenuItem, len(qualities))
	for i, q := range qualities {
		qualItems[i] = tui.MenuItem{Label: string(q), Value: strconv.Itoa(i)}
	}
	qChoice := tui.RunMenu("Select quality", qualItems)
	if qChoice == "q" || qChoice == "back" {
		return scraper.QualityAuto, fmt.Errorf("selection cancelled")
	}
	idx, _ := strconv.Atoi(qChoice)
	return qualities[idx], nil
}

// GetAvailableQualities returns available video qualities
func (mh *MediaHandler) GetAvailableQualities(episodeID string, isMovie bool) ([]scraper.Quality, error) {
	return mh.mediaManager.GetAvailableQualities(episodeID, isMovie)
}

// GetAnimeStreamURL gets stream URL for anime content
func (mh *MediaHandler) GetAnimeStreamURL(anime *models.Anime, episodeNum string, mode string) (string, map[string]string, error) {
	return mh.mediaManager.GetAnimeStreamURL(anime, episodeNum, string(mh.quality), mode)
}

// InteractiveMediaFlow runs an interactive media selection and playback flow
func (mh *MediaHandler) InteractiveMediaFlow(query string) (*PlaybackInfo, error) {
	// Select media type if not already searching
	var contentType models.MediaType
	if query == "" {
		var err error
		contentType, err = mh.SelectMediaType()
		if err != nil {
			return nil, err
		}
	}

	// Get search query if not provided
	if query == "" {
		var searchQuery string
		prompt := huh.NewInput().
			Title("Search").
			Value(&searchQuery)
		if err := tui.RunClean(prompt.Run); err != nil {
			return nil, err
		}
		query = searchQuery
	}

	// Search for media
	results, err := mh.SearchMedia(query, contentType)
	if err != nil {
		return nil, err
	}

	util.Debug("Search results", "count", len(results))

	// Select media
	selected, err := mh.SelectMedia(results)
	if err != nil {
		return nil, err
	}

	playbackInfo := &PlaybackInfo{
		Title:     selected.Name,
		MediaType: selected.MediaType,
		Source:    selected.Source,
		ImageURL:  selected.ImageURL,
	}

	// Handle based on media type and source
	if strings.Contains(strings.ToLower(selected.Source), "flixhq") {
		return mh.handleFlixHQPlayback(selected, playbackInfo)
	}

	// Handle anime sources
	return mh.handleAnimePlayback(selected, playbackInfo)
}

func (mh *MediaHandler) handleFlixHQPlayback(media *models.Anime, info *PlaybackInfo) (*PlaybackInfo, error) {
	mediaID := extractIDFromURL(media.URL)

	if media.MediaType == models.MediaTypeMovie {
		// Get available qualities for the movie
		qualities, err := mh.mediaManager.GetMovieQualities(mediaID)
		if err != nil {
			util.Debug("Could not fetch movie qualities", "error", err)
			// Fall back to default quality
			streamInfo, err := mh.mediaManager.GetMovieStreamInfo(mediaID, mh.provider, string(mh.quality), mh.subsLanguage)
			if err != nil {
				return nil, err
			}
			info.StreamURL = streamInfo.VideoURL
			info.Subtitles = convertSubtitles(streamInfo.Subtitles)
			return info, nil
		}

		// If qualities are available, let user select
		if len(qualities) > 0 {
			selectedQuality, err := mh.selectMovieQuality(qualities)
			if err != nil {
				util.Debug("Quality selection cancelled, using default", "error", err)
				// Keep mh.quality as default
			} else {
				mh.quality = selectedQuality
			}
		}

		streamInfo, err := mh.mediaManager.GetMovieStreamWithQuality(mediaID, mh.quality, mh.subsLanguage)
		if err != nil {
			return nil, err
		}
		info.StreamURL = streamInfo.VideoURL
		info.Subtitles = convertSubtitles(streamInfo.Subtitles)
		info.Quality = string(mh.quality)
		return info, nil
	}

	// TV Show flow
	season, err := mh.SelectSeason(mediaID)
	if err != nil {
		return nil, err
	}
	info.Season = season.Title

	episode, err := mh.SelectEpisode(season.ID)
	if err != nil {
		return nil, err
	}
	info.Episode = episode.Title
	info.EpisodeNum = episode.Number

	// Get available qualities for the episode
	qualities, err := mh.mediaManager.GetEpisodeQualities(episode.DataID)
	if err == nil && len(qualities) > 0 {
		selectedQuality, err := mh.selectMovieQuality(qualities)
		if err == nil {
			mh.quality = selectedQuality
		}
	}

	streamInfo, err := mh.mediaManager.GetTVEpisodeStreamInfo(episode.DataID, mh.provider, string(mh.quality), mh.subsLanguage)
	if err != nil {
		return nil, err
	}
	info.StreamURL = streamInfo.VideoURL
	info.Subtitles = convertSubtitles(streamInfo.Subtitles)
	info.Quality = string(mh.quality)

	return info, nil
}

// selectMovieQuality prompts user to select video quality from available options
func (mh *MediaHandler) selectMovieQuality(qualities []scraper.QualityOption) (scraper.Quality, error) {
	if len(qualities) == 0 {
		return mh.quality, nil
	}

	mqItems := make([]tui.MenuItem, len(qualities))
	for i, q := range qualities {
		mqItems[i] = tui.MenuItem{Label: q.Label, Value: strconv.Itoa(i)}
	}
	mqChoice := tui.RunMenu("Select video quality", mqItems)
	if mqChoice == "q" || mqChoice == "back" {
		return mh.quality, nil
	}
	idx, _ := strconv.Atoi(mqChoice)
	return qualities[idx].Quality, nil
}

func (mh *MediaHandler) handleAnimePlayback(anime *models.Anime, info *PlaybackInfo) (*PlaybackInfo, error) {
	// For anime, we need to select an episode
	var episodeNum string
	prompt := huh.NewInput().
		Title("Episode number").
		Value(&episodeNum).
		Validate(func(v string) error {
			if len(v) == 0 {
				return fmt.Errorf("episode number is required")
			}
			return nil
		})

	if err := tui.RunClean(prompt.Run); err != nil {
		return nil, err
	}
	if episodeNum == "" {
		episodeNum = "1"
	}

	modeItems := []tui.MenuItem{
		{Label: "Sub (Subtitled)", Value: "sub"},
		{Label: "Dub (English Dubbed)", Value: "dub"},
	}
	mode := tui.RunMenu("Select audio", modeItems)
	if mode == "q" || mode == "back" {
		return nil, fmt.Errorf("selection cancelled")
	}

	streamURL, metadata, err := mh.GetAnimeStreamURL(anime, episodeNum, mode)
	if err != nil {
		return nil, err
	}

	info.StreamURL = streamURL
	info.Episode = fmt.Sprintf("Episode %s", episodeNum)
	info.Metadata = metadata

	return info, nil
}

// PlaybackInfo contains all information needed for playback
type PlaybackInfo struct {
	Title      string
	MediaType  models.MediaType
	Source     string
	Season     string
	Episode    string
	EpisodeNum int
	StreamURL  string
	Quality    string
	Subtitles  []models.Subtitle
	Referer    string
	ImageURL   string
	Metadata   map[string]string
}

// Helper functions

func extractIDFromURL(urlStr string) string {
	// Extract ID from URL like /movie/watch-movie-name-12345 or /tv/watch-show-name-12345
	parts := strings.Split(urlStr, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func convertSubtitles(flixSubs []scraper.FlixHQSubtitle) []models.Subtitle {
	var subs []models.Subtitle
	for _, fs := range flixSubs {
		subs = append(subs, models.Subtitle{
			URL:      fs.URL,
			Language: fs.Language,
			Label:    fs.Label,
		})
	}
	return subs
}
