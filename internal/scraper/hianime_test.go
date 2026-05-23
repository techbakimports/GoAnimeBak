package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── parseSearchResults ───────────────────────────────────────────────────────

func TestHiAnimeParseSearchResults_FlwItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="film_list-wrap">
				<div class="flw-item">
					<h3 class="film-name"><a href="/watch/naruto-shippuden-365">Naruto Shippuden</a></h3>
					<img class="film-poster-img" data-src="https://cdn.example.com/naruto.jpg">
				</div>
				<div class="flw-item">
					<h3 class="film-name"><a href="/watch/one-piece-100">One Piece</a></h3>
				</div>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	results, err := client.SearchAnime("naruto")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Naruto Shippuden", results[0].Name)
	assert.Equal(t, "One Piece", results[1].Name)
	assert.Contains(t, results[0].URL, "/watch/naruto-shippuden-365")
	assert.Equal(t, "https://cdn.example.com/naruto.jpg", results[0].ImageURL)
}

func TestHiAnimeParseSearchResults_FallbackAnchor(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>
			<div class="flw-item">
				<a href="/watch/attack-on-titan-99">Attack on Titan</a>
			</div>
		</body></html>`)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	results, err := client.SearchAnime("titan")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Attack on Titan", results[0].Name)
}

func TestHiAnimeParseSearchResults_Empty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><p>No results found.</p></body></html>`)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	results, err := client.SearchAnime("zzzznonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ─── parseEpisodeHTML ─────────────────────────────────────────────────────────

func TestHiAnimeParseEpisodeHTML(t *testing.T) {
	t.Parallel()

	ajaxHTML := `<div class="ss-list">
		<a data-id="101" data-number="1" title="Enter the Ninja World">Episode 1</a>
		<a data-id="102" data-number="2" title="Bonds">Episode 2</a>
		<a data-id="550" data-number="500" title="Final Battle">Episode 500</a>
	</div>`

	client := NewHiAnimeClient()
	episodes := client.parseEpisodeHTML(ajaxHTML)

	require.Len(t, episodes, 3)
	assert.Equal(t, 1, episodes[0].Num)
	assert.Equal(t, "101", episodes[0].URL) // data-id stored as URL
	assert.Equal(t, "Enter the Ninja World", episodes[0].Title.English)
	assert.Equal(t, 500, episodes[2].Num)
	assert.Equal(t, "550", episodes[2].URL)
}

func TestHiAnimeParseEpisodeHTML_MissingDataID(t *testing.T) {
	t.Parallel()

	// data-id missing → episode must be skipped
	ajaxHTML := `<div>
		<a data-number="1" title="Ep 1">Episode 1</a>
		<a data-id="200" data-number="2" title="Ep 2">Episode 2</a>
	</div>`

	client := NewHiAnimeClient()
	episodes := client.parseEpisodeHTML(ajaxHTML)

	require.Len(t, episodes, 1)
	assert.Equal(t, 2, episodes[0].Num)
}

func TestHiAnimeParseEpisodeHTML_DefaultTitle(t *testing.T) {
	t.Parallel()

	ajaxHTML := `<a data-id="99" data-number="7"></a>`

	client := NewHiAnimeClient()
	episodes := client.parseEpisodeHTML(ajaxHTML)

	require.Len(t, episodes, 1)
	assert.Equal(t, "Episode 7", episodes[0].Title.English)
}

// ─── extractAnimeID ───────────────────────────────────────────────────────────

func TestHiAnimeExtractAnimeID(t *testing.T) {
	t.Parallel()

	client := NewHiAnimeClient()

	tests := []struct {
		url  string
		want string
	}{
		{"https://hianimes.se/watch/naruto-shippuden-365", "365"},
		{"https://hianimes.se/watch/one-piece-100", "100"},
		{"https://hianimes.se/watch/attack-on-titan-99/", "99"},
		{"https://hianimes.se/watch/my-hero-academia-42?ep=1234", "42"},
		{"https://hianimes.se/watch/no-id-here", ""},
	}

	for _, tc := range tests {
		got := client.extractAnimeID(tc.url)
		assert.Equal(t, tc.want, got, "url: %s", tc.url)
	}
}

// ─── GetAnimeEpisodes (AJAX) ──────────────────────────────────────────────────

func TestHiAnimeGetAnimeEpisodes_AJAXResponse(t *testing.T) {
	t.Parallel()

	ajaxHTML := `<a data-id="1" data-number="1" title="Ep 1"></a>
				 <a data-id="2" data-number="2" title="Ep 2"></a>`

	ajaxPayload, _ := json.Marshal(map[string]any{
		"status": true,
		"html":   ajaxHTML,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(ajaxPayload)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	// URL must end with -<id> for extractAnimeID to parse correctly
	episodes, err := client.GetAnimeEpisodes(server.URL + "/watch/naruto-365")
	require.NoError(t, err)
	require.Len(t, episodes, 2)
	assert.Equal(t, 1, episodes[0].Num)
	assert.Equal(t, "1", episodes[0].URL)
}

func TestHiAnimeGetAnimeEpisodes_NoID(t *testing.T) {
	t.Parallel()

	client := NewHiAnimeClient()
	client.baseURL = "http://example.com"

	_, err := client.GetAnimeEpisodes("http://example.com/watch/no-trailing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not extract anime ID")
}

// ─── Stream extraction ────────────────────────────────────────────────────────

func TestHiAnimeFetchSources_DirectM3U8(t *testing.T) {
	t.Parallel()

	sourcesPayload, _ := json.Marshal(map[string]any{
		"status": true,
		"sources": []map[string]string{
			{"file": "https://cdn.example.com/hls/naruto.m3u8", "type": "hls"},
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sourcesPayload)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	streamURL, err := client.fetchSources("server-xyz")
	require.NoError(t, err)
	assert.Contains(t, streamURL, ".m3u8")
}

func TestHiAnimeFetchSources_EmbedFallback(t *testing.T) {
	t.Parallel()

	sourcesPayload, _ := json.Marshal(map[string]any{
		"status":  true,
		"link":    "https://megacloud.tv/embed/e-1/abc123",
		"sources": []any{},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sourcesPayload)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	streamURL, err := client.fetchSources("server-xyz")
	require.NoError(t, err)
	assert.Contains(t, streamURL, "megacloud.tv")
}

func TestHiAnimeFetchSources_NoSource(t *testing.T) {
	t.Parallel()

	sourcesPayload, _ := json.Marshal(map[string]any{
		"status":  true,
		"sources": []any{},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sourcesPayload)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL

	_, err := client.fetchSources("server-xyz")
	require.Error(t, err)
}

// ─── Domain fallback / challenge page ────────────────────────────────────────

func TestHiAnimeSearchChallengeBlocked(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head><title>Just a moment...</title></head><body><div id="cf-wrapper"></div></body></html>`)
	}))
	defer server.Close()

	client := NewHiAnimeClient()
	client.baseURL = server.URL
	client.maxRetries = 0
	client.retryDelay = 0

	_, err := client.SearchAnime("naruto")
	require.Error(t, err)
}

// ─── Live network (skipped with -short) ──────────────────────────────────────

func TestHiAnimeSearch_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live HiAnime network test in -short mode")
	}

	client := NewHiAnimeClient()
	results, err := client.SearchAnime("naruto")
	if err != nil {
		t.Skipf("HiAnime search failed (may be blocked/unavailable): %v", err)
	}
	if len(results) == 0 {
		t.Skip("HiAnime returned no results (site may be unavailable)")
	}

	t.Logf("Found %d results", len(results))
	for i, r := range results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %q — %s", i, r.Name, r.URL)
	}
}
