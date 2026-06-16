package types

import (
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFromInternalAnime_Nil(t *testing.T) {
	assert.Nil(t, FromInternalAnime(nil))
}

func TestFromInternalAnime_Basic(t *testing.T) {
	internal := &models.Anime{
		Name:      "Naruto",
		URL:       "https://example.com/naruto",
		ImageURL:  "https://example.com/naruto.jpg",
		AnilistID: 20,
		MalID:     20,
		Source:    "AllAnime",
	}

	got := FromInternalAnime(internal)

	assert.Equal(t, "Naruto", got.Name)
	assert.Equal(t, "https://example.com/naruto", got.URL)
	assert.Equal(t, "https://example.com/naruto.jpg", got.ImageURL)
	assert.Equal(t, 20, got.AnilistID)
	assert.Equal(t, 20, got.MalID)
	assert.Equal(t, "AllAnime", got.Source)
	assert.Empty(t, got.Episodes)
	assert.NotNil(t, got.Details, "Details is always populated, even if empty")
	assert.Nil(t, got.Details.Title, "Title must stay nil when AniList provided no title")
	assert.Nil(t, got.Details.CoverImage, "CoverImage must stay nil when AniList provided no cover")
}

func TestFromInternalAnime_WithEpisodesAndDetails(t *testing.T) {
	internal := &models.Anime{
		Name: "One Piece",
		Episodes: []models.Episode{
			{Number: "1", Num: 1},
			{Number: "2", Num: 2},
		},
	}
	internal.Details.Title.Romaji = "Wan Pīsu"
	internal.Details.Title.English = "One Piece"
	internal.Details.CoverImage.Large = "https://example.com/large.jpg"

	got := FromInternalAnime(internal)

	assert.Len(t, got.Episodes, 2)
	assert.Equal(t, "1", got.Episodes[0].Number)
	assert.Equal(t, "2", got.Episodes[1].Number)

	require := assert.New(t)
	require.NotNil(got.Details.Title)
	require.Equal("Wan Pīsu", got.Details.Title.Romaji)
	require.Equal("One Piece", got.Details.Title.English)
	require.NotNil(got.Details.CoverImage)
	require.Equal("https://example.com/large.jpg", got.Details.CoverImage.Large)
}

func TestFromInternalAnimeList(t *testing.T) {
	internal := []*models.Anime{
		{Name: "A"},
		{Name: "B"},
	}

	got := FromInternalAnimeList(internal)

	assert.Len(t, got, 2)
	assert.Equal(t, "A", got[0].Name)
	assert.Equal(t, "B", got[1].Name)
}

func TestFromInternalAnimeList_Empty(t *testing.T) {
	got := FromInternalAnimeList(nil)
	assert.Empty(t, got)
}

func TestFromInternalEpisode_Nil(t *testing.T) {
	assert.Nil(t, FromInternalEpisode(nil))
}

func TestFromInternalEpisode_Basic(t *testing.T) {
	internal := &models.Episode{
		Number:   "1",
		Num:      1,
		URL:      "https://example.com/ep1",
		Aired:    "2020-01-01",
		Duration: 1440,
		IsFiller: true,
		IsRecap:  false,
		Synopsis: "Episode one",
		SeasonID: "1",
	}

	got := FromInternalEpisode(internal)

	assert.Equal(t, "1", got.Number)
	assert.Equal(t, 1, got.Num)
	assert.Equal(t, "https://example.com/ep1", got.URL)
	assert.Equal(t, "2020-01-01", got.Aired)
	assert.Equal(t, 1440, got.Duration)
	assert.True(t, got.IsFiller)
	assert.False(t, got.IsRecap)
	assert.Equal(t, "Episode one", got.Synopsis)
	assert.Equal(t, "1", got.SeasonID)
	assert.Nil(t, got.Title, "Title must stay nil when no title field is set")
	assert.Nil(t, got.SkipTimes, "SkipTimes must stay nil when no skip data is set")
}

func TestFromInternalEpisode_WithTitle(t *testing.T) {
	internal := &models.Episode{
		Title: models.TitleDetails{English: "The Beginning"},
	}

	got := FromInternalEpisode(internal)

	require := assert.New(t)
	require.NotNil(got.Title)
	require.Equal("The Beginning", got.Title.English)
}

func TestFromInternalEpisode_WithSkipTimes(t *testing.T) {
	internal := &models.Episode{
		SkipTimes: models.SkipTimes{
			Op: models.Skip{Start: 10, End: 100},
			Ed: models.Skip{Start: 1300, End: 1400},
		},
	}

	got := FromInternalEpisode(internal)

	require := assert.New(t)
	require.NotNil(got.SkipTimes)
	require.NotNil(got.SkipTimes.Op)
	require.Equal(10, got.SkipTimes.Op.Start)
	require.Equal(100, got.SkipTimes.Op.End)
	require.NotNil(got.SkipTimes.Ed)
	require.Equal(1300, got.SkipTimes.Ed.Start)
	require.Equal(1400, got.SkipTimes.Ed.End)
}

func TestFromInternalEpisode_ZeroSkipTimesStayNil(t *testing.T) {
	// All-zero skip times are indistinguishable from "not set" — must not
	// synthesize a SkipTimes object in that case.
	internal := &models.Episode{
		SkipTimes: models.SkipTimes{
			Op: models.Skip{Start: 0, End: 0},
			Ed: models.Skip{Start: 0, End: 0},
		},
	}

	got := FromInternalEpisode(internal)
	assert.Nil(t, got.SkipTimes)
}

func TestFromInternalEpisodeList(t *testing.T) {
	internal := []models.Episode{
		{Number: "1"},
		{Number: "2"},
	}

	got := FromInternalEpisodeList(internal)

	assert.Len(t, got, 2)
	assert.Equal(t, "1", got[0].Number)
	assert.Equal(t, "2", got[1].Number)
}

func TestFromInternalEpisodeList_Empty(t *testing.T) {
	got := FromInternalEpisodeList(nil)
	assert.Empty(t, got)
}
