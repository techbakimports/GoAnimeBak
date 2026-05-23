package scraper

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── parseSearchResults ───────────────────────────────────────────────────────

func TestAniNekoParseSearchResults_WatchLinks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="anime-card">
				<a href="/watch/naruto-shippuuden">
					<img src="https://cdn.example.com/naruto.jpg">
					Naruto Shippuuden
				</a>
			</div>
			<div class="anime-card">
				<a href="/watch/one-piece">One Piece</a>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "One Piece", results[1].Name)
	assert.Contains(t, results[0].URL, "/watch/naruto-shippuuden")
	assert.Equal(t, "https://cdn.example.com/naruto.jpg", results[0].ImageURL)
}

func TestAniNekoParseSearchResults_SkipsEpisodeLinks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/watch/naruto">Naruto</a>
			<a href="/watch/naruto/ep-1">Episode 1</a>
			<a href="/watch/naruto/ep-2">Episode 2</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 1, "episode links must be excluded")
	assert.Contains(t, results[0].URL, "/watch/naruto")
}

func TestAniNekoParseSearchResults_Deduplication(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/watch/naruto">Naruto</a>
			<a href="/watch/naruto">Naruto again</a>
			<a href="/watch/one-piece">One Piece</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	assert.Len(t, results, 2, "duplicate URLs must be deduplicated")
}

func TestAniNekoParseSearchResults_TitleFromHeading(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="item">
				<a href="/watch/bleach">
					<img src="cover.jpg">
				</a>
				<h3>Bleach</h3>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("bleach")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Bleach", results[0].Name)
}

func TestAniNekoParseSearchResults_Empty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><p>No results found.</p></body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("zzzznonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ─── parseEpisodeList ─────────────────────────────────────────────────────────

func TestAniNekoParseEpisodeList(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/watch/naruto/ep-1">Episode 1</a>
			<a href="/watch/naruto/ep-2">Episode 2</a>
			<a href="/watch/naruto/ep-500">Episode 500</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	episodes, err := client.GetAnimeEpisodes(server.URL + "/watch/naruto")
	require.NoError(t, err)
	require.Len(t, episodes, 3)

	// Must be sorted ascending
	assert.Equal(t, 1, episodes[0].Num)
	assert.Equal(t, 2, episodes[1].Num)
	assert.Equal(t, 500, episodes[2].Num)
	assert.Contains(t, episodes[0].URL, "/ep-1")
}

func TestAniNekoParseEpisodeList_Deduplication(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/watch/naruto/ep-1">Ep 1</a>
			<a href="/watch/naruto/ep-1">Ep 1 again</a>
			<a href="/watch/naruto/ep-2">Ep 2</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	episodes, err := client.GetAnimeEpisodes(server.URL + "/watch/naruto")
	require.NoError(t, err)
	assert.Len(t, episodes, 2, "duplicate episode numbers must be deduplicated")
}

func TestAniNekoParseEpisodeList_DefaultTitle(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/watch/naruto/ep-7"></a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	episodes, err := client.GetAnimeEpisodes(server.URL + "/watch/naruto")
	require.NoError(t, err)
	require.Len(t, episodes, 1)
	assert.Equal(t, "Episode 7", episodes[0].Title.English)
}

// ─── GetEpisodeStreamURL ──────────────────────────────────────────────────────

func TestAniNekoStreamURL_Iframe(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<iframe src="https://embed.example.com/player?id=abc123"></iframe>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.NoError(t, err)
	assert.Equal(t, "https://embed.example.com/player?id=abc123", streamURL)
}

func TestAniNekoStreamURL_VideoElement(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<video>
				<source src="https://cdn.example.com/stream/naruto-ep1.mp4">
			</video>
		</body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.NoError(t, err)
	assert.Contains(t, streamURL, "naruto-ep1")
}

func TestAniNekoStreamURL_M3U8InSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><script>
			var config = {"sources": [{"file": "https://cdn.example.com/hls/naruto.m3u8"}]};
		</script></body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.NoError(t, err)
	assert.Contains(t, streamURL, ".m3u8")
}

func TestAniNekoStreamURL_EmbedPattern(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><script>
			var src = "https://player.example.com/embed/abc123";
		</script></body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.NoError(t, err)
	assert.Contains(t, streamURL, "embed")
}

func TestAniNekoStreamURL_NoStream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><p>Loading...</p></body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	_, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.Error(t, err)
}

func TestAniNekoStreamURL_ChallengePageBlocked(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head><title>Just a moment...</title></head><body><div id="cf-wrapper"></div></body></html>`)
	}))
	defer server.Close()

	client := NewAniNekoClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	_, err := client.GetEpisodeStreamURL(server.URL + "/watch/naruto/ep-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceUnavailable))
}

// ─── Live network (skipped with -short) ──────────────────────────────────────

func TestAniNekoSearch_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live AniNeko network test in -short mode")
	}

	client := NewAniNekoClient()
	results, err := client.SearchAnime("naruto")
	if err != nil {
		t.Skipf("AniNeko search failed (may be blocked/unavailable): %v", err)
	}
	if len(results) == 0 {
		t.Skip("AniNeko returned no results (site may be unavailable or using CSR)")
	}

	t.Logf("Found %d results", len(results))
	for i, r := range results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %q — %s", i, r.Name, r.URL)
	}
}
