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

func TestGogoAnimeParseSearchResults_WordPressArticle(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<article class="post">
				<h2 class="entry-title"><a href="/series/naruto-shippuuden/">Naruto Shippuuden</a></h2>
				<img src="https://example.com/naruto.jpg">
			</article>
			<article class="post">
				<h2 class="entry-title"><a href="/series/one-piece/">One Piece</a></h2>
			</article>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Naruto Shippuuden", results[0].Name)
	assert.Equal(t, "One Piece", results[1].Name)
	assert.Contains(t, results[0].URL, "/series/naruto-shippuuden/")
	assert.Equal(t, "https://example.com/naruto.jpg", results[0].ImageURL)
}

func TestGogoAnimeParseSearchResults_ItemH4(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="item">
				<h4><a href="/anime/attack-on-titan/">Attack on Titan</a></h4>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("titan")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Attack on Titan", results[0].Name)
}

func TestGogoAnimeParseSearchResults_SkipsEpisodeLinks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<article>
				<h2><a href="/series/naruto/">Naruto</a></h2>
			</article>
			<article>
				<h2><a href="/naruto-episode-1-english-subbed/">Naruto Episode 1</a></h2>
			</article>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 1, "episode links must be excluded from search results")
	assert.Equal(t, "Naruto", results[0].Name)
}

func TestGogoAnimeParseSearchResults_Empty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><p>No results found.</p></body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("zzzznonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ─── parseEpisodeLinks ────────────────────────────────────────────────────────

func TestGogoAnimeParseEpisodeLinks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/naruto-shippuuden-episode-1-english-subbed/">Episode 1</a>
			<a href="/naruto-shippuuden-episode-2-english-subbed/">Episode 2</a>
			<a href="/naruto-shippuuden-episode-500-english-subbed/">Episode 500</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	episodes, err := client.GetAnimeEpisodes(server.URL + "/series/naruto-shippuuden/")
	require.NoError(t, err)
	require.Len(t, episodes, 3)

	// Must be sorted ascending
	assert.Equal(t, 1, episodes[0].Num)
	assert.Equal(t, 2, episodes[1].Num)
	assert.Equal(t, 500, episodes[2].Num)
	assert.Contains(t, episodes[0].URL, "-episode-1-")
}

func TestGogoAnimeParseEpisodeLinks_Deduplication(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Same episode linked twice (header + body)
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/naruto-episode-1-english-subbed/">Ep 1</a>
			<a href="/naruto-episode-1-english-subbed/">Ep 1 again</a>
			<a href="/naruto-episode-2-english-subbed/">Ep 2</a>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	episodes, err := client.GetAnimeEpisodes(server.URL + "/series/naruto/")
	require.NoError(t, err)
	assert.Len(t, episodes, 2, "duplicate episode numbers must be deduplicated")
}

// ─── GetEpisodeStreamURL ──────────────────────────────────────────────────────

func TestGogoAnimeStreamURL_Iframe(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="myvideo">
				<iframe src="https://playtaku.net/streaming.php?id=abc123"></iframe>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/naruto-episode-1-english-subbed/")
	require.NoError(t, err)
	assert.Equal(t, "https://playtaku.net/streaming.php?id=abc123", streamURL)
}

func TestGogoAnimeStreamURL_M3U8InSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><script>
			var sources = [{"file":"https://cdn.example.com/hls/naruto-ep1.m3u8","label":"720p"}];
		</script></body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	streamURL, err := client.GetEpisodeStreamURL(server.URL + "/naruto-episode-1-english-subbed/")
	require.NoError(t, err)
	assert.Contains(t, streamURL, ".m3u8")
}

func TestGogoAnimeStreamURL_NoStream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><p>Episode loading...</p></body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	_, err := client.GetEpisodeStreamURL(server.URL + "/naruto-episode-1-english-subbed/")
	require.Error(t, err)
}

func TestGogoAnimeStreamURL_ChallengePageBlocked(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head><title>Just a moment...</title></head><body><div id="cf-wrapper"></div></body></html>`)
	}))
	defer server.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{server.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	_, err := client.GetEpisodeStreamURL(server.URL + "/naruto-episode-1-english-subbed/")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceUnavailable))
}

// ─── Domain fallback ──────────────────────────────────────────────────────────

func TestGogoAnimeDomainFallback(t *testing.T) {
	t.Parallel()

	// First server always returns 500; second returns a valid result
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<article><h2><a href="/series/naruto/">Naruto</a></h2></article>
		</body></html>`)
	}))
	defer good.Close()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()

	client := NewGogoAnimeClient()
	client.domains = []string{bad.URL, good.URL}
	client.maxRetries = 0
	client.retryDelay = 0

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 1, "should fall back to second domain when first is down")
}

// ─── Live network (skipped with -short) ──────────────────────────────────────

func TestGogoAnimeSearch_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live GogoAnime network test in -short mode")
	}

	client := NewGogoAnimeClient()
	results, err := client.SearchAnime("naruto")
	if err != nil {
		t.Skipf("GogoAnime search failed (may be blocked/unavailable): %v", err)
	}
	if len(results) == 0 {
		t.Skip("GogoAnime returned no results (site may be unavailable)")
	}

	t.Logf("Found %d results", len(results))
	for i, r := range results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %q — %s", i, r.Name, r.URL)
	}
}
