// Package scraper provides web scraping for HiAnime (hianimes.se).
// Uses the aniwatch.to-compatible AJAX API for search, episodes, and streams.
package scraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/util"
)

const (
	HiAnimeBase      = "https://hianimes.se"
	hiAnimeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

var (
	// hiAnimeIDRe extracts the numeric anime ID appended by aniwatch-style sites.
	// HiAnime always appends the DB ID as the last hyphen-separated token, e.g.
	// "/watch/naruto-shippuden-365" → "365". Titles never end with a plain number
	// in the aniwatch codebase — slugs like "one-piece-1000" would have id "1000"
	// which is correct (it IS the anime's DB id, not the episode count).
	hiAnimeIDRe   = regexp.MustCompile(`-(\d+)$`)
	hiAnimeM3U8Re = regexp.MustCompile(`(https?://[^\s"'<>]+\.m3u8[^\s"'<>]*)`)
)

// hiAnimeAJAXResp is the standard JSON envelope returned by HiAnime AJAX endpoints.
type hiAnimeAJAXResp struct {
	Status bool   `json:"status"`
	HTML   string `json:"html"`
}

// hiAnimeSourcesResp contains streaming source data from the sources endpoint.
type hiAnimeSourcesResp struct {
	Status  bool   `json:"status"`
	Link    string `json:"link"`
	Sources []struct {
		File string `json:"file"`
		Type string `json:"type"`
	} `json:"sources"`
}

// HiAnimeClient handles interactions with hianimes.se (aniwatch.to-style API).
type HiAnimeClient struct {
	client     *http.Client
	baseURL    string
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewHiAnimeClient creates a new HiAnime client.
func NewHiAnimeClient() *HiAnimeClient {
	return &HiAnimeClient{
		client:     util.NewFastClient(),
		baseURL:    HiAnimeBase,
		userAgent:  hiAnimeUserAgent,
		maxRetries: 2,
		retryDelay: 300 * time.Millisecond,
	}
}

func (c *HiAnimeClient) decorateRequest(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", c.baseURL+"/")
}

func (c *HiAnimeClient) decorateAJAX(req *http.Request) {
	c.decorateRequest(req)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func (c *HiAnimeClient) shouldRetry(attempt int) bool { return attempt < c.maxRetries }
func (c *HiAnimeClient) sleep()                        { time.Sleep(c.retryDelay) }

// SearchAnime searches hianimes.se via the /search page.
// The results page uses .flw-item cards, same layout as aniwatch.to and 9anime.
func (c *HiAnimeClient) SearchAnime(query string) ([]*models.Anime, error) {
	searchURL := fmt.Sprintf("%s/search?keyword=%s", c.baseURL, url.QueryEscape(strings.TrimSpace(query)))
	util.Debug("HiAnime search", "url", searchURL)

	attempts := c.maxRetries + 1
	var lastErr error

	for attempt := range attempts {
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.decorateRequest(req)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, fmt.Errorf("request: %w", err)
		}

		if err := checkHTTPStatus(resp, "HiAnime search"); err != nil {
			_ = resp.Body.Close()
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("parse HTML: %w", err)
		}

		if err := checkChallengeDocument(doc, "HiAnime search"); err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		return c.parseSearchResults(doc), nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("hianime: search failed")
}

// parseSearchResults extracts .flw-item cards from the search results page.
func (c *HiAnimeClient) parseSearchResults(doc *goquery.Document) []*models.Anime {
	var animes []*models.Anime

	doc.Find(".flw-item, .film_list-wrap .item").Each(func(_ int, item *goquery.Selection) {
		linkTag := item.Find("h3.film-name a, .film-name a").First()
		if linkTag.Length() == 0 {
			linkTag = item.Find("a[href*='/watch/']").First()
		}
		if linkTag.Length() == 0 {
			return
		}

		title := strings.TrimSpace(linkTag.Text())
		href, _ := linkTag.Attr("href")
		if title == "" || href == "" {
			return
		}

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = c.baseURL + href
		}

		var imgURL string
		if img := item.Find("img.film-poster-img, img[data-src]").First(); img.Length() > 0 {
			imgURL, _ = img.Attr("data-src")
			if imgURL == "" {
				imgURL, _ = img.Attr("src")
			}
		}

		animes = append(animes, &models.Anime{
			Name:      title,
			URL:       fullURL,
			ImageURL:  imgURL,
			Source:    "HiAnime",
			MediaType: models.MediaTypeAnime,
		})
	})

	return animes
}

// GetAnimeEpisodes fetches the episode list via the HiAnime AJAX API.
// animeURL must be the full watch URL (e.g. https://hianimes.se/watch/naruto-365).
func (c *HiAnimeClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	util.Debug("HiAnime episodes", "url", animeURL)

	animeID := c.extractAnimeID(animeURL)
	if animeID == "" {
		return nil, fmt.Errorf("hianime: could not extract anime ID from %q", animeURL)
	}

	ajaxURL := fmt.Sprintf("%s/ajax/v2/episode/list/%s", c.baseURL, animeID)
	attempts := c.maxRetries + 1
	var lastErr error

	for attempt := range attempts {
		req, err := http.NewRequest("GET", ajaxURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.decorateAJAX(req)
		req.Header.Set("Referer", animeURL)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, fmt.Errorf("request: %w", err)
		}

		if err := checkHTTPStatus(resp, "HiAnime episodes"); err != nil {
			_ = resp.Body.Close()
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		var ajaxResp hiAnimeAJAXResp
		if err := json.Unmarshal(body, &ajaxResp); err != nil {
			lastErr = fmt.Errorf("parse JSON: %w", err)
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, lastErr
		}

		if !ajaxResp.Status || ajaxResp.HTML == "" {
			return nil, errors.New("hianime: episode list API returned empty response")
		}

		episodes := c.parseEpisodeHTML(ajaxResp.HTML)
		sort.Slice(episodes, func(i, j int) bool { return episodes[i].Num < episodes[j].Num })
		return episodes, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("hianime: failed to get episodes")
}

// parseEpisodeHTML parses the HTML snippet returned by the episode list AJAX endpoint.
// Each episode link has a data-id (used for the stream API) and data-number.
func (c *HiAnimeClient) parseEpisodeHTML(html string) []models.Episode {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var episodes []models.Episode
	doc.Find("a[data-id][data-number]").Each(func(_ int, s *goquery.Selection) {
		dataID, _ := s.Attr("data-id")
		dataNum, _ := s.Attr("data-number")
		title, _ := s.Attr("title")

		num, err := strconv.Atoi(dataNum)
		if err != nil || dataID == "" {
			return
		}
		if title == "" {
			title = fmt.Sprintf("Episode %d", num)
		}

		episodes = append(episodes, models.Episode{
			Number: fmt.Sprintf("Episode %d", num),
			Num:    num,
			URL:    dataID, // data-id is passed to stream API — same pattern as NineAnime
			Title:  models.TitleDetails{English: title},
		})
	})

	return episodes
}

// GetEpisodeStreamURL fetches the stream URL using the HiAnime AJAX chain.
// episodeID is the data-id value from the episode list (see parseEpisodeHTML).
// The chain is: servers list → pick "sub" server → sources endpoint.
// Sources may return a direct M3U8 (preferred) or an embed URL (e.g. megacloud.tv).
// Megacloud embed URLs require additional decryption and will not play directly in mpv;
// in that case the caller receives the embed URL and playback may fail.
func (c *HiAnimeClient) GetEpisodeStreamURL(episodeID string) (string, error) {
	util.Debug("HiAnime stream", "episodeID", episodeID)

	serverID, err := c.pickSubServer(episodeID)
	if err != nil {
		return "", fmt.Errorf("hianime: get servers: %w", err)
	}

	return c.fetchSources(serverID)
}

func (c *HiAnimeClient) pickSubServer(episodeID string) (string, error) {
	serversURL := fmt.Sprintf("%s/ajax/v2/episode/servers?episodeId=%s", c.baseURL, episodeID)

	req, err := http.NewRequest("GET", serversURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.decorateAJAX(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "HiAnime servers"); err != nil {
		return "", err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	var ajaxResp hiAnimeAJAXResp
	if err := json.Unmarshal(body, &ajaxResp); err != nil {
		return "", fmt.Errorf("parse JSON: %w", err)
	}
	if !ajaxResp.Status || ajaxResp.HTML == "" {
		return "", errors.New("hianime: no servers available")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(ajaxResp.HTML))
	if err != nil {
		return "", fmt.Errorf("parse servers HTML: %w", err)
	}

	var serverID string
	// Prefer "sub" type; fall back to first available
	doc.Find("li[data-id]").Each(func(_ int, s *goquery.Selection) {
		if serverID != "" {
			return
		}
		dataType, _ := s.Attr("data-type")
		id, _ := s.Attr("data-id")
		if dataType == "sub" && id != "" {
			serverID = id
		}
	})
	if serverID == "" {
		serverID, _ = doc.Find("li[data-id]").First().Attr("data-id")
	}
	if serverID == "" {
		return "", errors.New("hianime: no server found in response")
	}
	return serverID, nil
}

func (c *HiAnimeClient) fetchSources(serverID string) (string, error) {
	sourcesURL := fmt.Sprintf("%s/ajax/v2/episode/sources?id=%s", c.baseURL, serverID)

	req, err := http.NewRequest("GET", sourcesURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.decorateAJAX(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "HiAnime sources"); err != nil {
		return "", err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	var sourcesResp hiAnimeSourcesResp
	if err := json.Unmarshal(body, &sourcesResp); err != nil {
		return "", fmt.Errorf("parse JSON: %w", err)
	}

	// Prefer direct M3U8 sources
	for _, src := range sourcesResp.Sources {
		if src.File != "" && strings.Contains(src.File, ".m3u8") {
			return validateStreamURL(src.File, "HiAnime")
		}
	}

	// Embed link (e.g. megacloud.tv) as fallback
	if sourcesResp.Link != "" {
		return validateStreamURL(sourcesResp.Link, "HiAnime")
	}

	// Last resort: scan raw body for M3U8
	if m := hiAnimeM3U8Re.FindString(string(body)); m != "" {
		return validateStreamURL(m, "HiAnime")
	}

	return "", errors.New("hianime: no stream source found")
}

// extractAnimeID extracts the numeric anime ID from a watch URL.
// HiAnime (aniwatch.to-style) always appends the DB id as the last token:
//   https://hianimes.se/watch/naruto-shippuden-365  →  "365"
//   https://hianimes.se/watch/one-piece-100         →  "100"
func (c *HiAnimeClient) extractAnimeID(watchURL string) string {
	path := strings.TrimSuffix(watchURL, "/")
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if m := hiAnimeIDRe.FindStringSubmatch(path); len(m) >= 2 {
		return m[1]
	}
	return ""
}
