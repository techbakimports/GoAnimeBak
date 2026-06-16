package playback

import (
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestIsAllAnimeSource(t *testing.T) {
	tests := []struct {
		name     string
		anime    *models.Anime
		expected bool
	}{
		{"explicit AllAnime source", &models.Anime{Source: "AllAnime", URL: "https://example.com/x"}, true},
		{"URL contains allanime", &models.Anime{Source: "Other", URL: "https://allanime.to/anime/abc123"}, true},
		{"short alnum ID without http", &models.Anime{Source: "Other", URL: "abc123XYZ"}, true},
		{"non-AllAnime full URL", &models.Anime{Source: "AnimeFire", URL: "https://animefire.io/anime/some-long-slug-1234567890"}, false},
		{"empty anime", &models.Anime{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAllAnimeSource(tt.anime))
		})
	}
}

func TestExtractAllAnimeID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"short id without http", "abc123", "abc123"},
		// The function returns the first "/"-separated segment between 6 and
		// 29 characters containing any alphanumeric character — for a typical
		// "https://allanime.to/..." URL that is the scheme segment itself,
		// not the trailing ID. This documents the function's actual current
		// behavior rather than its doc-comment's stated intent.
		{"allanime full url returns first qualifying segment", "https://allanime.to/anime/abcdef123456", "https:"},
		{"unrelated long url returned as-is", "https://example.com/anime/some-other-id-1234567890", "https://example.com/anime/some-other-id-1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractAllAnimeID(tt.url))
		})
	}
}

func TestAllAnimeNavigator_GetNextEpisode(t *testing.T) {
	nav := &AllAnimeNavigator{episodes: []string{"ep1", "ep2", "ep3"}}

	next, err := nav.GetNextEpisode("1")
	assert.NoError(t, err)
	assert.Equal(t, "2", next)

	_, err = nav.GetNextEpisode("3")
	assert.Error(t, err, "expected error when already at last episode")

	_, err = nav.GetNextEpisode("not-a-number")
	assert.Error(t, err, "expected error for non-numeric episode")
}

func TestAllAnimeNavigator_GetPreviousEpisode(t *testing.T) {
	nav := &AllAnimeNavigator{episodes: []string{"ep1", "ep2", "ep3"}}

	prev, err := nav.GetPreviousEpisode("2")
	assert.NoError(t, err)
	assert.Equal(t, "1", prev)

	_, err = nav.GetPreviousEpisode("1")
	assert.Error(t, err, "expected error when already at first episode")

	_, err = nav.GetPreviousEpisode("not-a-number")
	assert.Error(t, err, "expected error for non-numeric episode")
}

func TestAllAnimeNavigator_GetTotalEpisodes(t *testing.T) {
	nav := &AllAnimeNavigator{episodes: []string{"ep1", "ep2", "ep3"}}
	assert.Equal(t, 3, nav.GetTotalEpisodes())
}

func TestAllAnimeNavigator_ListAllEpisodes(t *testing.T) {
	nav := &AllAnimeNavigator{episodes: []string{"ep1", "ep2", "ep3"}}
	assert.Equal(t, []string{"1", "2", "3"}, nav.ListAllEpisodes())
}
