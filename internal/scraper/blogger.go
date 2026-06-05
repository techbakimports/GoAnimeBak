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
	bloggerSIDRe      = regexp.MustCompile(`"FdrFJe":"([^"]+)"`)
	bloggerBuildRe    = regexp.MustCompile(`"cfb2h":"([^"]+)"`)
	bloggerATRe       = regexp.MustCompile(`"SNlM0e":"([^"]+)"`)
	bloggerVideoURLRe = regexp.MustCompile(`https?://[a-z0-9-]+\.googlevideo\.com/videoplayback[^"\\]+`)
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
func ResolveBloggerURLFull(bloggerURL string) (*BloggerResult, error) {
	tokenMatch := bloggerTokenRe.FindStringSubmatch(bloggerURL)
	if len(tokenMatch) < 2 {
		return nil, fmt.Errorf("could not extract token from Blogger URL: %s", bloggerURL)
	}
	token := tokenMatch[1]

	// Use a cookie jar to collect session cookies
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

	// Step 1: Load the Blogger page to extract session params
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

	// Try regex fallback first — sometimes the video URL is directly in the page
	if matches := bloggerVideoURLRe.FindString(pageText); matches != "" {
		decoded := decodeBloggerURL(matches)
		util.Debug("Blogger: found video URL directly in page", "url", decoded[:min(len(decoded), 80)])
		return &BloggerResult{
			VideoURL: decoded,
			Cookies:  collectCookies(jar, "https://www.blogger.com"),
		}, nil
	}

	// Step 2: Extract session params for batchexecute
	sidMatch := bloggerSIDRe.FindStringSubmatch(pageText)
	bhMatch := bloggerBuildRe.FindStringSubmatch(pageText)
	if len(sidMatch) < 2 || len(bhMatch) < 2 {
		return nil, errors.New("failed to extract Blogger session params (FdrFJe/cfb2h)")
	}
	sid := sidMatch[1]
	bh := bhMatch[1]
	at := ""
	if atMatch := bloggerATRe.FindStringSubmatch(pageText); len(atMatch) >= 2 {
		at = atMatch[1]
	}

	// Step 3: Call batchexecute to get the googlevideo URL
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
	batchReq.Header.Set("Referer", bloggerURL)
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

	// Collect all cookies from both blogger.com and googlevideo domains
	cookieStr := collectCookies(jar, "https://www.blogger.com")

	// Try regex on the raw body first
	if matches := bloggerVideoURLRe.FindString(bodyStr); matches != "" {
		decoded := decodeBloggerURL(matches)
		util.Debug("Blogger: resolved via batchexecute", "url", decoded[:min(len(decoded), 80)])
		return &BloggerResult{VideoURL: decoded, Cookies: cookieStr}, nil
	}

	// Try structured parsing
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
						return &BloggerResult{VideoURL: decoded, Cookies: cookieStr}, nil
					}
				}
			}
		}
	}

	return nil, errors.New("could not extract video URL from Blogger batchexecute response")
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

func collectCookies(jar *simpleCookieJar, baseURL string) string {
	var parts []string
	for _, cookies := range jar.cookies {
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	return strings.Join(parts, "; ")
}
