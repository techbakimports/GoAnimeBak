// Package scraper provides web scraping for AniNeko (anineko.to).
package scraper

import (
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
	AniNekoBase      = "https://anineko.to"
	aniNekoUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

var (
	aniNekoEpNumRe = regexp.MustCompile(`/ep-(\d+)(?:[/?#]|$)`)
	aniNekoM3U8Re  = regexp.MustCompile(`(https?://[^\s"'<>]+\.m3u8[^\s"'<>]*)`)
	aniNekoSrcRe   = regexp.MustCompile(`(?:src|file|url)\s*[=:]\s*["'](https?://[^"']+)["']`)
)

// AniNekoClient handles scraping of anineko.to.
type AniNekoClient struct {
	client     *http.Client
	baseURL    string
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewAniNekoClient creates a new AniNeko client.
func NewAniNekoClient() *AniNekoClient {
	return &AniNekoClient{
		client:     util.NewFastClient(),
		baseURL:    AniNekoBase,
		userAgent:  aniNekoUserAgent,
		maxRetries: 2,
		retryDelay: 300 * time.Millisecond,
	}
}

func (c *AniNekoClient) decorateRequest(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", c.baseURL+"/")
}

func (c *AniNekoClient) shouldRetry(attempt int) bool { return attempt < c.maxRetries }
func (c *AniNekoClient) sleep()                        { time.Sleep(c.retryDelay) }

// SearchAnime searches anineko.to via the /browse?q= endpoint.
// Note: if AniNeko migrates to client-side rendering (CSR), the HTML response
// will be a shell with no content and results will always be empty. In that case
// the scraper will return an empty slice without error, and a headless renderer
// would be required to proceed.
func (c *AniNekoClient) SearchAnime(query string) ([]*models.Anime, error) {
	searchURL := fmt.Sprintf("%s/browse?q=%s", c.baseURL, url.QueryEscape(strings.TrimSpace(query)))
	util.Debug("AniNeko search", "url", searchURL)

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

		if err := checkHTTPStatus(resp, "AniNeko search"); err != nil {
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

		if err := checkChallengeDocument(doc, "AniNeko search"); err != nil {
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
	return nil, errors.New("anineko: search failed")
}

func (c *AniNekoClient) parseSearchResults(doc *goquery.Document) []*models.Anime {
	var animes []*models.Anime
	seen := make(map[string]bool)

	// AniNeko uses /watch/{slug} for anime pages and /watch/{slug}/ep-N for episodes
	doc.Find("a[href*='/watch/']").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		// Skip episode links
		if strings.Contains(href, "/ep-") {
			return
		}

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = c.baseURL + href
		}
		if seen[fullURL] {
			return
		}
		seen[fullURL] = true

		title := strings.TrimSpace(s.Text())
		if title == "" {
			if h := s.Closest(".anime-card, .item, article").Find("h2, h3, h4").First(); h.Length() > 0 {
				title = strings.TrimSpace(h.Text())
			}
		}
		if title == "" {
			return
		}

		var imgURL string
		if img := s.Find("img").First(); img.Length() > 0 {
			imgURL, _ = img.Attr("src")
			if imgURL == "" {
				imgURL, _ = img.Attr("data-src")
			}
		}

		animes = append(animes, &models.Anime{
			Name:      title,
			URL:       fullURL,
			ImageURL:  imgURL,
			Source:    "AniNeko",
			MediaType: models.MediaTypeAnime,
		})
	})

	return animes
}

// GetAnimeEpisodes fetches the episode list from an anime's /watch/{slug} page.
func (c *AniNekoClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	util.Debug("AniNeko episodes", "url", animeURL)
	attempts := c.maxRetries + 1
	var lastErr error

	for attempt := range attempts {
		req, err := http.NewRequest("GET", animeURL, nil)
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

		if err := checkHTTPStatus(resp, "AniNeko episodes"); err != nil {
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

		if err := checkChallengeDocument(doc, "AniNeko episodes"); err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		eps := c.parseEpisodeList(doc)
		sort.Slice(eps, func(i, j int) bool { return eps[i].Num < eps[j].Num })
		return eps, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("anineko: failed to get episodes")
}

func (c *AniNekoClient) parseEpisodeList(doc *goquery.Document) []models.Episode {
	var episodes []models.Episode
	seen := make(map[int]bool)

	// AniNeko episode links: /watch/{slug}/ep-{N}
	doc.Find("a[href*='/ep-']").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		matches := aniNekoEpNumRe.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil || num <= 0 || seen[num] {
			return
		}
		seen[num] = true

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = c.baseURL + href
		}

		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = fmt.Sprintf("Episode %d", num)
		}

		episodes = append(episodes, models.Episode{
			Number: fmt.Sprintf("Episode %d", num),
			Num:    num,
			URL:    fullURL,
			Title:  models.TitleDetails{English: title},
		})
	})

	return episodes
}

// GetEpisodeStreamURL attempts to extract the stream URL from an episode page.
// AniNeko loads its video player via JavaScript. This scraper parses static HTML only,
// so stream extraction is best-effort:
//   - If the site uses server-side rendering, an <iframe> or <video> src may be present.
//   - If AniNeko uses client-side rendering (React/Next.js), the player HTML will be
//     absent from the static response and all strategies will fail gracefully.
// In the CSR case, an error is returned and no stream URL is available without a
// JavaScript-capable renderer.
func (c *AniNekoClient) GetEpisodeStreamURL(episodeURL string) (string, error) {
	util.Debug("AniNeko stream", "url", episodeURL)

	req, err := http.NewRequest("GET", episodeURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.decorateRequest(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "AniNeko stream"); err != nil {
		return "", err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	html := string(body)

	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); err == nil {
		if err := checkChallengeDocument(doc, "AniNeko stream"); err != nil {
			return "", err
		}

		// Strategy 1: iframe src
		if src, exists := doc.Find("iframe[src]").First().Attr("src"); exists && src != "" {
			util.Debug("AniNeko: iframe found", "src", src)
			return validateStreamURL(src, "AniNeko")
		}

		// Strategy 2: video element
		if src, exists := doc.Find("video source[src]").Attr("src"); exists && src != "" {
			return validateStreamURL(src, "AniNeko")
		}
	}

	// Strategy 3: M3U8 directly in page source
	if m := aniNekoM3U8Re.FindString(html); m != "" {
		util.Debug("AniNeko: M3U8 in source", "url", m)
		return validateStreamURL(m, "AniNeko")
	}

	// Strategy 4: any embed/player URL in script attributes
	for _, m := range aniNekoSrcRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 2 {
			u := m[1]
			lower := strings.ToLower(u)
			if strings.Contains(lower, "embed") || strings.Contains(lower, "player") ||
				strings.Contains(lower, "stream") || strings.Contains(lower, ".m3u8") {
				util.Debug("AniNeko: embed URL found", "url", u)
				return validateStreamURL(u, "AniNeko")
			}
		}
	}

	return "", fmt.Errorf("anineko: could not extract stream URL from %s", episodeURL)
}
