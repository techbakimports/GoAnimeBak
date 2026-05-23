// Package scraper provides web scraping for GogoAnime sites.
// Tries domains in cascade: gogoanime.by → gogoanime.or.at
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

const gogoAnimeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// GogoAnimeDomains is the ordered fallback list tried on every request.
var GogoAnimeDomains = []string{
	"https://gogoanime.by",
	"https://gogoanime.or.at",
}

var (
	gogoEpisodeNumRe = regexp.MustCompile(`-episode-(\d+)`)
	gogoM3U8Re       = regexp.MustCompile(`(https?://[^\s"'<>]+\.m3u8[^\s"'<>]*)`)
	gogoEmbedURLRe   = regexp.MustCompile(`(?:src|file)\s*[=:]\s*["'](https?://[^"']+(?:playtaku|streaming\.php|embed)[^"']*)["']`)
)

// GogoAnimeClient scrapes GogoAnime-based sites across multiple fallback domains.
type GogoAnimeClient struct {
	client     *http.Client
	domains    []string
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewGogoAnimeClient creates a new GogoAnime client.
func NewGogoAnimeClient() *GogoAnimeClient {
	return &GogoAnimeClient{
		client:     util.NewFastClient(),
		domains:    GogoAnimeDomains,
		userAgent:  gogoAnimeUserAgent,
		maxRetries: 2,
		retryDelay: 300 * time.Millisecond,
	}
}

func (c *GogoAnimeClient) decorateRequest(req *http.Request, base string) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if base != "" {
		req.Header.Set("Referer", base+"/")
	}
}

func (c *GogoAnimeClient) shouldRetry(attempt int) bool { return attempt < c.maxRetries }
func (c *GogoAnimeClient) sleep()                        { time.Sleep(c.retryDelay) }

// SearchAnime searches across all configured GogoAnime domains.
func (c *GogoAnimeClient) SearchAnime(query string) ([]*models.Anime, error) {
	query = strings.TrimSpace(query)
	util.Debug("GogoAnime search", "query", query)

	var lastErr error
	for _, domain := range c.domains {
		results, err := c.searchOnDomain(domain, query)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		util.Debug("GogoAnime domain unavailable", "domain", domain, "error", err)
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("gogoanime: all domains failed: %w", lastErr)
	}
	return nil, nil
}

func (c *GogoAnimeClient) searchOnDomain(domain, query string) ([]*models.Anime, error) {
	searchURL := fmt.Sprintf("%s/?s=%s", domain, url.QueryEscape(query))
	attempts := c.maxRetries + 1

	for attempt := range attempts {
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.decorateRequest(req, domain)

		resp, err := c.client.Do(req)
		if err != nil {
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, fmt.Errorf("request: %w", err)
		}

		if err := checkHTTPStatus(resp, "GogoAnime search"); err != nil {
			_ = resp.Body.Close()
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

		if err := checkChallengeDocument(doc, "GogoAnime search"); err != nil {
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		return c.parseSearchResults(doc, domain), nil
	}
	return nil, errors.New("gogoanime: max retries exceeded")
}

// gogoSearchSelectors is tried in order — most specific first, broad scan last.
// GogoAnime.by (WordPress) and GogoAnime.or.at use different themes, so we try
// multiple targeted selectors before falling back to the broad link scan.
var gogoSearchSelectors = []string{
	".entry-title a[href]",
	".post-title a[href]",
	"article h2 a[href]",
	"article h3 a[href]",
	"article h4 a[href]",
	".item h4 a[href]",
	".item h3 a[href]",
	".film-name a[href]",
}

// parseSearchResults extracts anime cards from the search results page.
// GogoAnime sites link to /series/ (gogoanime.by) or /anime/ (gogoanime.or.at) paths.
// Tries targeted WordPress heading selectors before falling back to a broad link scan.
func (c *GogoAnimeClient) parseSearchResults(doc *goquery.Document, domain string) []*models.Anime {
	var animes []*models.Anime
	seen := make(map[string]bool)

	addResult := func(href, title string, s *goquery.Selection) {
		if (!strings.Contains(href, "/series/") && !strings.Contains(href, "/anime/")) ||
			strings.Contains(href, "-episode-") {
			return
		}
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = domain + href
		}
		if seen[fullURL] || title == "" {
			return
		}
		seen[fullURL] = true

		var imgURL string
		if img := s.Closest("article, .item, .post").Find("img").First(); img.Length() > 0 {
			imgURL, _ = img.Attr("src")
			if imgURL == "" || strings.HasPrefix(imgURL, "data:") {
				imgURL, _ = img.Attr("data-src")
			}
		}

		animes = append(animes, &models.Anime{
			Name:      title,
			URL:       fullURL,
			ImageURL:  imgURL,
			Source:    "GogoAnime",
			MediaType: models.MediaTypeAnime,
		})
	}

	// Pass 1: targeted heading/title selectors (more reliable title text)
	for _, sel := range gogoSearchSelectors {
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			title := strings.TrimSpace(s.Text())
			if title == "" {
				title, _ = s.Attr("title")
			}
			addResult(href, title, s)
		})
		if len(animes) > 0 {
			return animes
		}
	}

	// Pass 2: broad scan — catches themes that wrap title+image in a single <a>
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		title := strings.TrimSpace(s.Text())
		if title == "" {
			title, _ = s.Attr("title")
		}
		addResult(href, title, s)
	})

	return animes
}

// GetAnimeEpisodes fetches the episode list from a GogoAnime series page.
func (c *GogoAnimeClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	util.Debug("GogoAnime episodes", "url", animeURL)
	domain := c.domainOf(animeURL)
	attempts := c.maxRetries + 1

	var lastErr error
	for attempt := range attempts {
		req, err := http.NewRequest("GET", animeURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.decorateRequest(req, domain)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, fmt.Errorf("request: %w", err)
		}

		if err := checkHTTPStatus(resp, "GogoAnime episodes"); err != nil {
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

		if err := checkChallengeDocument(doc, "GogoAnime episodes"); err != nil {
			lastErr = err
			if c.shouldRetry(attempt) {
				c.sleep()
				continue
			}
			return nil, err
		}

		eps := c.parseEpisodeLinks(doc, domain)
		sort.Slice(eps, func(i, j int) bool { return eps[i].Num < eps[j].Num })
		return eps, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("gogoanime: failed to get episodes")
}

func (c *GogoAnimeClient) parseEpisodeLinks(doc *goquery.Document, domain string) []models.Episode {
	var episodes []models.Episode
	seen := make(map[int]bool)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		matches := gogoEpisodeNumRe.FindStringSubmatch(href)
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
			fullURL = domain + href
		}

		episodes = append(episodes, models.Episode{
			Number: fmt.Sprintf("Episode %d", num),
			Num:    num,
			URL:    fullURL,
		})
	})

	return episodes
}

// GetEpisodeStreamURL extracts the streaming URL from a GogoAnime episode page.
// GogoAnime loads its video player via JavaScript, so this is best-effort:
//   - If the page renders an <iframe> server-side, the iframe src is returned directly.
//   - If an M3U8 URL is inlined in the page source, it is returned.
//   - If a known embed URL pattern (playtaku, streaming.php) is found, it is returned.
// When all strategies fail, an error is returned and playback will not be possible
// for that episode without a JavaScript-capable renderer.
func (c *GogoAnimeClient) GetEpisodeStreamURL(episodeURL string) (string, error) {
	util.Debug("GogoAnime stream", "url", episodeURL)
	domain := c.domainOf(episodeURL)

	req, err := http.NewRequest("GET", episodeURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.decorateRequest(req, domain)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "GogoAnime stream"); err != nil {
		return "", err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	html := string(body)

	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); err == nil {
		if err := checkChallengeDocument(doc, "GogoAnime stream"); err != nil {
			return "", err
		}

		// Strategy 1: iframe embed
		if src, exists := doc.Find("iframe[src]").First().Attr("src"); exists && src != "" {
			util.Debug("GogoAnime: iframe found", "src", src)
			return validateStreamURL(src, "GogoAnime")
		}
		// Strategy 2: video element
		if src, exists := doc.Find("video source[src]").Attr("src"); exists && src != "" {
			return validateStreamURL(src, "GogoAnime")
		}
	}

	// Strategy 3: M3U8 directly in page source
	if m := gogoM3U8Re.FindString(html); m != "" {
		util.Debug("GogoAnime: M3U8 in source", "url", m)
		return validateStreamURL(m, "GogoAnime")
	}

	// Strategy 4: known embed URL patterns
	if m := gogoEmbedURLRe.FindStringSubmatch(html); len(m) >= 2 {
		util.Debug("GogoAnime: embed URL found", "url", m[1])
		return validateStreamURL(m[1], "GogoAnime")
	}

	return "", fmt.Errorf("gogoanime: could not extract stream URL from %s", episodeURL)
}

// domainOf returns the base domain for a GogoAnime URL.
func (c *GogoAnimeClient) domainOf(rawURL string) string {
	for _, d := range c.domains {
		if strings.HasPrefix(rawURL, d) {
			return d
		}
	}
	if i := strings.Index(rawURL, "://"); i >= 0 {
		rest := rawURL[i+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			return rawURL[:i+3+slash]
		}
	}
	return c.domains[0]
}
