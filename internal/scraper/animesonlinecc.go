// Package scraper provides web scraping functionality for animesonlinecc.to
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
	animesOnlineCCBase = "https://animesonlinecc.to"
)

// animesOnlineCCEpNumRe extracts episode number from slugs like "naruto-classico-episodio-42"
var animesOnlineCCEpNumRe = regexp.MustCompile(`-episodio-(\d+)/?$`)

// AnimesOnlineCCClient handles interactions with animesonlinecc.to.
// Architecture: WordPress site, server-rendered HTML, no Cloudflare.
// Videos are embedded as Blogger iframes (same as Goyabu), resolved
// via ResolveBloggerURLFull to googlevideo CDN URLs.
type AnimesOnlineCCClient struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

// NewAnimesOnlineCCClient creates a new client for animesonlinecc.to
func NewAnimesOnlineCCClient() *AnimesOnlineCCClient {
	return &AnimesOnlineCCClient{
		client:    util.NewFastClient(),
		baseURL:   animesOnlineCCBase,
		userAgent: UserAgent,
	}
}

func (c *AnimesOnlineCCClient) decorateRequest(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8")
	req.Header.Set("Referer", c.baseURL+"/")
}

// SearchAnime searches by following /pesquisar/{slug} which WordPress redirects
// to the best-matching anime page. Returns the matched anime or an error.
func (c *AnimesOnlineCCClient) SearchAnime(query string) ([]*models.Anime, error) {
	query = strings.TrimSpace(query)
	// Convert spaces to hyphens for the URL slug
	slug := strings.ToLower(strings.ReplaceAll(query, " ", "-"))
	searchURL := fmt.Sprintf("%s/pesquisar/%s", c.baseURL, url.PathEscape(slug))
	util.Debug("AnimesOnlineCC search", "url", searchURL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.decorateRequest(req)

	// Use a client that follows redirects and records final URL
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "AnimesOnlineCC search"); err != nil {
		return nil, err
	}

	// After redirect, the final URL should be an /anime/ page
	finalURL := resp.Request.URL.String()
	if !strings.Contains(finalURL, "/anime/") {
		return nil, errors.New("animesonlinecc: search did not match any anime")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	if err := checkChallengeDocument(doc, "AnimesOnlineCC search"); err != nil {
		return nil, err
	}

	title := c.extractPageTitle(doc, query)
	imgURL := c.extractThumbnail(doc)

	return []*models.Anime{{
		Name:     title,
		URL:      finalURL,
		ImageURL: imgURL,
	}}, nil
}

// GetAnimeEpisodes returns all episodes listed on the anime page.
// Episodes are found as /episodio/ links; numbers are parsed from the slug.
func (c *AnimesOnlineCCClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	util.Debug("AnimesOnlineCC episodes", "url", animeURL)

	req, err := http.NewRequest("GET", animeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.decorateRequest(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("episodes request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "AnimesOnlineCC episodes"); err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	if err := checkChallengeDocument(doc, "AnimesOnlineCC episodes"); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var episodes []models.Episode

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.Contains(href, "/episodio/") {
			return
		}
		// Normalise to absolute URL
		if !strings.HasPrefix(href, "http") {
			href = c.baseURL + href
		}
		if seen[href] {
			return
		}
		seen[href] = true

		num := 0
		if m := animesOnlineCCEpNumRe.FindStringSubmatch(href); len(m) >= 2 {
			if parsed, err := strconv.Atoi(m[1]); err == nil {
				num = parsed
			}
		}

		episodes = append(episodes, models.Episode{
			Number: fmt.Sprintf("Episódio %d", num),
			Num:    num,
			URL:    href,
		})
	})

	if len(episodes) == 0 {
		return nil, errors.New("animesonlinecc: no episodes found")
	}

	// Sort ascending by episode number
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Num < episodes[j].Num
	})

	// Re-assign sequential numbers for episodes where num==0 (couldn't parse)
	for i := range episodes {
		if episodes[i].Num == 0 {
			episodes[i].Num = i + 1
			episodes[i].Number = fmt.Sprintf("Episódio %d", i+1)
		}
	}

	return episodes, nil
}

// GetStreamURL fetches the episode page and returns the Blogger embed URL.
// The caller (GoyabuAdapter pattern) or pkg/goanime/client.go safety net
// resolves it to a direct googlevideo CDN URL via ResolveBloggerURLFull.
func (c *AnimesOnlineCCClient) GetStreamURL(episodeURL string) (string, error) {
	util.Debug("AnimesOnlineCC stream URL", "url", episodeURL)

	req, err := http.NewRequest("GET", episodeURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	c.decorateRequest(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("episode request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkHTTPStatus(resp, "AnimesOnlineCC episode page"); err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	if err := checkChallengeDocument(doc, "AnimesOnlineCC episode page"); err != nil {
		return "", err
	}

	// Primary: Blogger iframe (same pattern as Goyabu)
	if src, exists := doc.Find(`iframe[src*="blogger.com"], iframe[src*="blogspot.com"]`).Attr("src"); exists && src != "" {
		util.Debug("AnimesOnlineCC: Blogger iframe found", "src", src[:min(len(src), 80)])
		return src, nil
	}

	// Fallback: any iframe with a src
	if src, exists := doc.Find("iframe[src]").First().Attr("src"); exists && src != "" {
		util.Debug("AnimesOnlineCC: generic iframe fallback", "src", src[:min(len(src), 80)])
		return src, nil
	}

	return "", errors.New("animesonlinecc: no video embed found in episode page")
}

// extractPageTitle gets the anime title from the page.
func (c *AnimesOnlineCCClient) extractPageTitle(doc *goquery.Document, fallback string) string {
	// Try the post/entry title heading
	for _, sel := range []string{"h1.entry-title", "h1.post-title", "h1"} {
		if t := strings.TrimSpace(doc.Find(sel).First().Text()); t != "" {
			return t
		}
	}
	// Fall back to <title> tag, stripping the " – Site Name" suffix
	if t := doc.Find("title").First().Text(); t != "" {
		if idx := strings.Index(t, " – "); idx > 0 {
			return strings.TrimSpace(t[:idx])
		}
		if idx := strings.Index(t, " | "); idx > 0 {
			return strings.TrimSpace(t[:idx])
		}
		if idx := strings.Index(t, " - "); idx > 0 {
			return strings.TrimSpace(t[:idx])
		}
		return strings.TrimSpace(t)
	}
	return fallback
}

// extractThumbnail finds a suitable thumbnail URL from the page.
func (c *AnimesOnlineCCClient) extractThumbnail(doc *goquery.Document) string {
	// Prefer Open Graph image
	if content, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && content != "" {
		return content
	}
	// First content image (skip logos/icons)
	var found string
	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		if found != "" {
			return
		}
		src, _ := s.Attr("src")
		lc := strings.ToLower(src)
		if strings.Contains(lc, "logo") || strings.Contains(lc, "icon") || strings.Contains(lc, "avatar") {
			return
		}
		if strings.HasPrefix(src, "http") {
			found = src
		}
	})
	return found
}
