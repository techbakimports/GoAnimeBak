package playback

import (
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestHandleUserNavigation(t *testing.T) {
	episodes := []models.Episode{
		{URL: "https://example.com/ep1", Number: "1"},
		{URL: "https://example.com/ep2", Number: "2"},
		{URL: "https://example.com/ep3", Number: "3"},
	}

	t.Run("previous", func(t *testing.T) {
		url, numStr, num := handleUserNavigation("p", episodes, 2, 3)
		assert.Equal(t, "https://example.com/ep1", url)
		assert.Equal(t, "1", numStr)
		assert.Equal(t, 1, num)
	})

	t.Run("next", func(t *testing.T) {
		url, numStr, num := handleUserNavigation("n", episodes, 2, 3)
		assert.Equal(t, "https://example.com/ep3", url)
		assert.Equal(t, "3", numStr)
		assert.Equal(t, 3, num)
	})

	t.Run("previous clamped to first episode", func(t *testing.T) {
		url, numStr, num := handleUserNavigation("p", episodes, 1, 3)
		assert.Equal(t, "https://example.com/ep1", url)
		assert.Equal(t, "1", numStr)
		assert.Equal(t, 1, num)
	})

	t.Run("next clamped to last episode", func(t *testing.T) {
		url, numStr, num := handleUserNavigation("n", episodes, 3, 3)
		assert.Equal(t, "https://example.com/ep3", url)
		assert.Equal(t, "3", numStr)
		assert.Equal(t, 3, num)
	})

	t.Run("fuzzy selection with empty list returns error and keeps currentNum", func(t *testing.T) {
		url, numStr, num := handleUserNavigation("e", []models.Episode{}, 2, 3)
		assert.Empty(t, url)
		assert.Empty(t, numStr)
		assert.Equal(t, 2, num)
	})
}

func TestHandleUserNavigationEnhanced_NonAllAnimeSourceDelegates(t *testing.T) {
	anime := &models.Anime{Source: "AnimeFire", URL: "https://animefire.io/anime/some-long-slug-1234567890"}
	episodes := []models.Episode{
		{URL: "https://example.com/ep1", Number: "1"},
		{URL: "https://example.com/ep2", Number: "2"},
	}

	url, numStr, num := handleUserNavigationEnhanced("n", episodes, 1, 2, anime)

	assert.Equal(t, "https://example.com/ep2", url)
	assert.Equal(t, "2", numStr)
	assert.Equal(t, 2, num)
}

func TestHandleAllAnimeNavigation_CurrentEpisodeNotFoundFallsBack(t *testing.T) {
	anime := &models.Anime{Source: "AllAnime", URL: "abc123XYZ"}
	episodes := []models.Episode{
		{URL: "https://example.com/ep1", Number: "1", Num: 1},
		{URL: "https://example.com/ep2", Number: "2", Num: 2},
	}

	// currentNum 99 matches no episode's Num field, so handleAllAnimeNavigation
	// must fall back to handleUserNavigation without making any network calls.
	url, numStr, num := handleAllAnimeNavigation("n", episodes, 99, 2, anime)

	assert.Equal(t, "https://example.com/ep2", url)
	assert.Equal(t, "2", numStr)
	assert.Equal(t, 2, num)
}
