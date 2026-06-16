package types

import (
	"testing"

	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/stretchr/testify/assert"
)

func TestSourceString_AllValues(t *testing.T) {
	tests := []struct {
		source   Source
		expected string
	}{
		{SourceAllAnime, "AllAnime"},
		{SourceAnimeFire, "AnimeFire"},
		{SourceGoyabu, "Goyabu"},
		{SourceGogoAnime, "GogoAnime"},
		{SourceAnimesOnlineCC, "AnimesOnlineCC"},
		{SourceAnimeHeaven, "AnimeHeaven"},
		{Source(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}

func TestSourceToScraperType_AllValues(t *testing.T) {
	tests := []struct {
		source   Source
		expected scraper.ScraperType
	}{
		{SourceAllAnime, scraper.AllAnimeType},
		{SourceAnimeFire, scraper.AnimefireType},
		{SourceGoyabu, scraper.GoyabuType},
		{SourceGogoAnime, scraper.GogoAnimeType},
		{SourceAnimesOnlineCC, scraper.AnimesOnlineCCType},
		{SourceAnimeHeaven, scraper.AnimeHeavenType},
		{Source(999), scraper.AllAnimeType}, // unknown falls back to AllAnime
	}

	for _, tt := range tests {
		t.Run(tt.source.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.ToScraperType())
		})
	}
}

func TestParseSource_AllAliases(t *testing.T) {
	tests := []struct {
		input    string
		expected Source
	}{
		{"AllAnime", SourceAllAnime},
		{"allanime", SourceAllAnime},
		{"all", SourceAllAnime},
		{"AnimeFire", SourceAnimeFire},
		{"animefire", SourceAnimeFire},
		{"fire", SourceAnimeFire},
		{"animefire.io", SourceAnimeFire},
		{"Goyabu", SourceGoyabu},
		{"goyabu", SourceGoyabu},
		{"GogoAnime", SourceGogoAnime},
		{"gogoanime", SourceGogoAnime},
		{"AnimesOnlineCC", SourceAnimesOnlineCC},
		{"animesonlinecc", SourceAnimesOnlineCC},
		{"animescc", SourceAnimesOnlineCC},
		{"AnimeHeaven", SourceAnimeHeaven},
		{"animeheaven", SourceAnimeHeaven},
		{"heaven", SourceAnimeHeaven},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSource(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseSource_Invalid(t *testing.T) {
	got, err := ParseSource("not-a-real-source")
	assert.Error(t, err)
	assert.Equal(t, SourceAllAnime, got, "invalid input still returns the AllAnime zero value alongside the error")
}
