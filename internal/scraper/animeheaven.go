package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/util"
)

const AnimeHeavenBase = "https://animeheaven.me"

var (
	ahSearchResultRe = regexp.MustCompile(`<a href='(anime\.php\?[a-z0-9]+)'><img class='coverimg'[^>]+alt='([^']+)'`)
	ahGateaRe        = regexp.MustCompile(`onclick='gatea\("([a-f0-9]{32})"\)'`)
	ahEpNumRe        = regexp.MustCompile(`<div class='watch2 bc[^']*'>(\d+)</div>`)
	ahStreamRe       = regexp.MustCompile(`<source src='(https://[a-z0-9\-]+\.animeheaven\.me/video\.mp4\?[^'"]+)'`)
)

type AnimeHeavenClient struct {
	client *http.Client
}

func NewAnimeHeavenClient() *AnimeHeavenClient {
	return &AnimeHeavenClient{client: util.NewFastClient()}
}

func (c *AnimeHeavenClient) SearchAnime(query string) ([]*models.Anime, error) {
	searchURL := AnimeHeavenBase + "/search.php?s=" + url.QueryEscape(query)
	body, err := c.fetchHTML(searchURL, AnimeHeavenBase+"/")
	if err != nil {
		return nil, fmt.Errorf("animeheaven search: %w", err)
	}

	matches := ahSearchResultRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	animes := make([]*models.Anime, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		animeURL := AnimeHeavenBase + "/" + m[1]
		if seen[animeURL] {
			continue
		}
		seen[animeURL] = true
		animes = append(animes, &models.Anime{
			Name: m[2],
			URL:  animeURL,
		})
	}
	return animes, nil
}

func (c *AnimeHeavenClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	body, err := c.fetchHTML(animeURL, AnimeHeavenBase+"/")
	if err != nil {
		return nil, fmt.Errorf("animeheaven episodes: %w", err)
	}

	hashes := ahGateaRe.FindAllStringSubmatch(body, -1)
	numbers := ahEpNumRe.FindAllStringSubmatch(body, -1)

	episodes := make([]models.Episode, 0, len(hashes))
	for i, h := range hashes {
		num := ""
		numInt := 0
		if i < len(numbers) {
			num = numbers[i][1]
			numInt, _ = strconv.Atoi(num)
		}
		if num == "" {
			numInt = i + 1
			num = strconv.Itoa(numInt)
		}
		episodes = append(episodes, models.Episode{
			Number: num,
			Num:    numInt,
			URL:    AnimeHeavenBase + "/gate.php#" + h[1],
		})
	}
	return episodes, nil
}

// GetStreamURL resolves the MP4 URL for an episode.
// episodeURL must be "https://animeheaven.me/gate.php#<md5hash>".
func (c *AnimeHeavenClient) GetStreamURL(episodeURL string) (string, error) {
	hash := extractAnimeHeavenHash(episodeURL)
	if hash == "" {
		return "", fmt.Errorf("animeheaven: no episode hash in URL %q", episodeURL)
	}

	req, err := http.NewRequest("GET", AnimeHeavenBase+"/gate.php", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", AnimeHeavenBase+"/")
	req.Header.Set("Cookie", "key="+hash)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("animeheaven gate.php: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", fmt.Errorf("animeheaven gate.php read: %w", err)
	}
	body := string(bodyBytes)

	m := ahStreamRe.FindStringSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("animeheaven: no video URL found for hash %s", hash)
	}

	// Use the first primary CDN source; strip trailing error parameters
	videoURL := m[1]
	if idx := strings.Index(videoURL, "&error"); idx != -1 {
		videoURL = videoURL[:idx]
	}

	util.Debugf("AnimeHeaven stream: %s", videoURL)
	return videoURL, nil
}

func extractAnimeHeavenHash(episodeURL string) string {
	idx := strings.LastIndex(episodeURL, "#")
	if idx == -1 {
		return ""
	}
	hash := episodeURL[idx+1:]
	if len(hash) == 32 {
		return hash
	}
	return ""
}

func (c *AnimeHeavenClient) fetchHTML(rawURL, referer string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", referer)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
