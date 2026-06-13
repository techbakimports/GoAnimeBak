package scraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/alvarorichard/Goanime/internal/util"
)

// Blogger video resolution — extracts the actual googlevideo CDN URL from a
// Blogger video embed page (e.g. https://www.blogger.com/video.g?token=...).

var (
	bloggerTokenRe    = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)
	bloggerVideoURLRe = regexp.MustCompile(`https?://[a-z0-9-]+\.googlevideo\.com/videoplayback[^"\\]+`)

	// WIZ session ID: known field name first, then heuristic (large negative integer)
	bloggerSIDPatterns = []*regexp.Regexp{
		regexp.MustCompile(`"FdrFJe"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"[A-Za-z0-9_]{4,8}"\s*:\s*"(-\d{15,20})"`),
	}
	// WIZ build label: known field name first, then heuristic (boq_ prefix)
	bloggerBuildPatterns = []*regexp.Regexp{
		regexp.MustCompile(`"cfb2h"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"[A-Za-z0-9_]{4,8}"\s*:\s*"(boq_[A-Za-z0-9_./-]+)"`),
	}
	bloggerATRe = regexp.MustCompile(`"SNlM0e"\s*:\s*"([^"]+)"`)

	// HTML fallbacks
	bloggerVideoSrcRe  = regexp.MustCompile(`(?i)<(?:video|source)[^>]+src=["']([^"']+)["']`)
	bloggerOGVideoRe   = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:video["'][^>]+content=["']([^"']+)["']`)
	bloggerOGVideoRe2  = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:video["']`)
	bloggerPlayURLRe   = regexp.MustCompile(`"(?:play_url|videoUrl|contentUrl|iurl|url)"\s*:\s*"(https?://[^"]*(?:googlevideo|video)[^"]+)"`)
)

// BloggerResult holds the resolved video URL and any cookies/headers
// needed to access it (googlevideo.com requires session cookies).
type BloggerResult struct {
	VideoURL string
	Cookies  string // Cookie header value for the video request
}

// ResolveBloggerURL extracts the direct googlevideo CDN URL from a Blogger
// video embed page. Simple wrapper that returns just the URL.
func ResolveBloggerURL(bloggerURL string) (string, error) {
	result, err := ResolveBloggerURLFull(bloggerURL)
	if err != nil {
		return "", err
	}
	return result.VideoURL, nil
}

// ResolveBloggerURLFull resolves a Blogger embed URL and returns both the
// video URL and cookies needed to access googlevideo.com.
// Tries multiple strategies in order: direct page scan → HTML tags → batchexecute.
func ResolveBloggerURLFull(bloggerURL string) (*BloggerResult, error) {
	tokenMatch := bloggerTokenRe.FindStringSubmatch(bloggerURL)
	if len(tokenMatch) < 2 {
		return nil, fmt.Errorf("could not extract token from Blogger URL: %s", bloggerURL)
	}
	token := tokenMatch[1]

	jar, _ := newSimpleCookieJar()
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", bloggerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/125.0.0.0 Mobile Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to load Blogger page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	pageBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read Blogger page: %w", err)
	}
	pageText := string(pageBody)
	cookieStr := collectCookies(jar, "https://www.blogger.com")

	// Strategy 1: direct googlevideo URL in page
	if u := bloggerVideoURLRe.FindString(pageText); u != "" {
		decoded := decodeBloggerURL(u)
		util.Debug("Blogger: found video URL directly in page", "url", decoded[:min(len(decoded), 80)])
		return &BloggerResult{VideoURL: decoded, Cookies: cookieStr}, nil
	}

	// Strategy 2: HTML <video>/<source> tags
	if m := bloggerVideoSrcRe.FindStringSubmatch(pageText); len(m) >= 2 {
		u := decodeBloggerURL(m[1])
		if strings.HasPrefix(u, "http") {
			util.Debug("Blogger: found video URL in HTML tag", "url", u[:min(len(u), 80)])
			return &BloggerResult{VideoURL: u, Cookies: cookieStr}, nil
		}
	}

	// Strategy 3: og:video meta tag
	for _, re := range []*regexp.Regexp{bloggerOGVideoRe, bloggerOGVideoRe2} {
		if m := re.FindStringSubmatch(pageText); len(m) >= 2 {
			u := decodeBloggerURL(m[1])
			if strings.HasPrefix(u, "http") {
				util.Debug("Blogger: found video URL in og:video", "url", u[:min(len(u), 80)])
				return &BloggerResult{VideoURL: u, Cookies: cookieStr}, nil
			}
		}
	}

	// Strategy 4: JSON patterns in page (play_url, videoUrl, etc.)
	if m := bloggerPlayURLRe.FindStringSubmatch(pageText); len(m) >= 2 {
		u := decodeBloggerURL(m[1])
		util.Debug("Blogger: found video URL in JSON pattern", "url", u[:min(len(u), 80)])
		return &BloggerResult{VideoURL: u, Cookies: cookieStr}, nil
	}

	// Strategy 5: batchexecute (requires WIZ session params)
	sid := extractFirst(pageText, bloggerSIDPatterns)
	bh := extractFirst(pageText, bloggerBuildPatterns)
	if sid == "" || bh == "" {
		util.Debug("Blogger: WIZ session params not found, trying video-play endpoint", "sid_found", sid != "", "bh_found", bh != "")
	} else {
		at := ""
		if m := bloggerATRe.FindStringSubmatch(pageText); len(m) >= 2 {
			at = m[1]
		}

		if result, err := bloggerBatchExecute(client, token, sid, bh, at, bloggerURL); err == nil {
			result.Cookies = cookieStr
			return result, nil
		} else {
			util.Debug("Blogger: batchexecute failed, trying video-play fallback", "error", err)
		}
	}

	// Strategy 6: video-play.mp4 redirect endpoint (last resort)
	if result, err := bloggerVideoPlayEndpoint(client, token, cookieStr); err == nil {
		return result, nil
	}

	return nil, errors.New("failed to resolve Blogger video: all strategies exhausted")
}

// extractFirst tries each pattern in order and returns the first match.
func extractFirst(text string, patterns []*regexp.Regexp) string {
	for _, re := range patterns {
		if m := re.FindStringSubmatch(text); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}

// bloggerBatchExecute performs the batchexecute API call to get the video URL.
func bloggerBatchExecute(client *http.Client, token, sid, bh, at, referer string) (*BloggerResult, error) {
	inner, err := json.Marshal([]any{token, "", 0})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inner data: %w", err)
	}
	freq, err := json.Marshal([][]any{{[]any{"WcwnYd", string(inner), nil, "generic"}}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal freq data: %w", err)
	}
	postData := "f.req=" + url.QueryEscape(string(freq))
	if at != "" {
		postData += "&at=" + url.QueryEscape(at)
	}

	batchURL := fmt.Sprintf(
		"https://www.blogger.com/_/BloggerVideoPlayerUi/data/batchexecute?rpcids=WcwnYd&source-path=%%2Fvideo.g&f.sid=%s&bl=%s&hl=en-US&_reqid=100001&rt=c",
		url.QueryEscape(sid), url.QueryEscape(bh),
	)

	batchReq, err := http.NewRequest("POST", batchURL, strings.NewReader(postData))
	if err != nil {
		return nil, fmt.Errorf("failed to create batchexecute request: %w", err)
	}
	batchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	batchReq.Header.Set("X-Same-Domain", "1")
	batchReq.Header.Set("Origin", "https://www.blogger.com")
	batchReq.Header.Set("Referer", referer)
	batchReq.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/125.0.0.0 Mobile Safari/537.36")

	batchResp, err := client.Do(batchReq)
	if err != nil {
		return nil, fmt.Errorf("batchexecute request failed: %w", err)
	}
	defer func() { _ = batchResp.Body.Close() }()

	batchBody, err := io.ReadAll(io.LimitReader(batchResp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read batchexecute response: %w", err)
	}
	if batchResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batchexecute returned status %d", batchResp.StatusCode)
	}

	bodyStr := string(batchBody)

	// Quick regex scan first
	if u := bloggerVideoURLRe.FindString(bodyStr); u != "" {
		decoded := decodeBloggerURL(u)
		util.Debug("Blogger: resolved via batchexecute (regex)", "url", decoded[:min(len(decoded), 80)])
		return &BloggerResult{VideoURL: decoded}, nil
	}

	// Structured JSON parse of batchexecute response
	for _, line := range strings.Split(bodyStr, "\n") {
		if !strings.Contains(line, "wrb.fr") {
			continue
		}
		var outer []any
		if err := json.Unmarshal([]byte(line), &outer); err != nil {
			continue
		}
		for _, entry := range outer {
			arr, ok := entry.([]any)
			if !ok || len(arr) < 3 {
				continue
			}
			if fmt.Sprint(arr[0]) != "wrb.fr" || fmt.Sprint(arr[1]) != "WcwnYd" {
				continue
			}
			var data []any
			if err := json.Unmarshal(fmt.Append(nil, arr[2]), &data); err != nil {
				continue
			}
			for _, item := range data {
				itemArr, ok := item.([]any)
				if !ok || len(itemArr) == 0 {
					continue
				}
				for _, stream := range itemArr {
					streamArr, ok := stream.([]any)
					if !ok || len(streamArr) < 2 {
						continue
					}
					if urlStr, ok := streamArr[0].(string); ok && strings.Contains(urlStr, "googlevideo.com") {
						decoded := decodeBloggerURL(urlStr)
						return &BloggerResult{VideoURL: decoded}, nil
					}
				}
			}
		}
	}

	return nil, errors.New("batchexecute returned no video URL")
}

// bloggerVideoPlayEndpoint tries the Blogger video-play redirect endpoint.
// Blogger sometimes serves a direct redirect to the video file from this URL.
func bloggerVideoPlayEndpoint(client *http.Client, token, cookies string) (*BloggerResult, error) {
	playURL := "https://www.blogger.com/video-play.mp4?contentId=" + token

	req, err := http.NewRequest("GET", playURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/125.0.0.0 Mobile Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "googlevideo.com") || strings.Contains(finalURL, ".mp4") {
		util.Debug("Blogger: resolved via video-play endpoint", "url", finalURL[:min(len(finalURL), 80)])
		return &BloggerResult{VideoURL: finalURL, Cookies: cookies}, nil
	}

	// Also scan the response body for video URLs
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if u := bloggerVideoURLRe.FindString(string(body)); u != "" {
		decoded := decodeBloggerURL(u)
		return &BloggerResult{VideoURL: decoded, Cookies: cookies}, nil
	}

	return nil, errors.New("video-play endpoint did not return a video URL")
}

func decodeBloggerURL(raw string) string {
	s := strings.ReplaceAll(raw, "\\u003d", "=")
	s = strings.ReplaceAll(s, "\\u0026", "&")
	return s
}

// Simple cookie jar that just stores cookies per URL
type simpleCookieJar struct {
	cookies map[string][]*http.Cookie
}

func newSimpleCookieJar() (*simpleCookieJar, error) {
	return &simpleCookieJar{cookies: make(map[string][]*http.Cookie)}, nil
}

func (j *simpleCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	key := u.Scheme + "://" + u.Host
	j.cookies[key] = append(j.cookies[key], cookies...)
}

func (j *simpleCookieJar) Cookies(u *url.URL) []*http.Cookie {
	key := u.Scheme + "://" + u.Host
	return j.cookies[key]
}

func collectCookies(jar *simpleCookieJar, _ string) string {
	var parts []string
	for _, cookies := range jar.cookies {
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	return strings.Join(parts, "; ")
}
