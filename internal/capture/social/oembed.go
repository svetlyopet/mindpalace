package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	xOEmbedBase        = "https://publish.twitter.com/oembed"
	fbGraphOEmbedPost  = "https://graph.facebook.com/v26.0/oembed_post"
	fbGraphOEmbedVideo = "https://graph.facebook.com/v26.0/oembed_video"
	maxOEmbedBody      = 1 << 20
)

type oEmbedResponse struct {
	HTML       string `json:"html"`
	AuthorName string `json:"author_name"`
	AuthorURL  string `json:"author_url"`
	Provider   string `json:"provider_name"`
}

// Fetch retrieves and parses a public post via the platform oEmbed API.
func Fetch(ctx context.Context, client *http.Client, platform Platform, canonicalURL string) (*Post, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint, err := oEmbedEndpoint(platform, canonicalURL)
	if err != nil {
		return nil, err
	}
	raw, err := fetchOEmbedJSON(ctx, client, endpoint)
	if err != nil {
		return nil, err
	}
	var resp oEmbedResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("social oembed decode: %w", err)
	}
	if strings.TrimSpace(resp.HTML) == "" {
		return nil, fmt.Errorf("social oembed: empty html")
	}
	parsed, err := ParseOEmbedHTML(resp.HTML)
	if err != nil {
		return nil, err
	}
	post := &Post{
		Platform:     platform,
		CanonicalURL: canonicalURL,
		Author:       BuildAuthorFromOEmbed(platform, resp.AuthorName, resp.AuthorURL, resp.HTML),
		PostID:       extractPostID(platform, canonicalURL),
		Text:         parsed.Text,
		Images:       parsed.Images,
		Videos:       parsed.Videos,
	}
	if post.Text == "" && len(post.Images) == 0 && len(post.Videos) == 0 {
		return nil, fmt.Errorf("social oembed: no post content extracted")
	}
	return post, nil
}

func oEmbedEndpoint(platform Platform, canonicalURL string) (string, error) {
	switch platform {
	case PlatformX:
		return xOEmbedBase + "?omit_script=true&url=" + url.QueryEscape(canonicalURL), nil
	case PlatformFacebook:
		base := fbGraphOEmbedPost
		if isFacebookVideoURL(canonicalURL) {
			base = fbGraphOEmbedVideo
		}
		return base + "?omitscript=true&url=" + url.QueryEscape(canonicalURL), nil
	default:
		return "", fmt.Errorf("unsupported platform %q", platform)
	}
}

func isFacebookVideoURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	path := u.Path
	q := u.Query()
	if strings.EqualFold(u.Host, "fb.watch") {
		return true
	}
	if fbWatchPath.MatchString(path) && q.Get("v") != "" {
		return true
	}
	if strings.HasPrefix(path, "/reel/") || strings.HasPrefix(path, "/videos/") {
		return true
	}
	return false
}

func fetchOEmbedJSON(ctx context.Context, client *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mindpalace/1.0 (+https://github.com/svetlyopet/mindpalace)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social oembed fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("social oembed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOEmbedBody+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxOEmbedBody {
		return nil, fmt.Errorf("social oembed: response too large")
	}
	return data, nil
}

func extractPostID(platform Platform, canonicalURL string) string {
	u, err := url.Parse(canonicalURL)
	if err != nil {
		return ""
	}
	switch platform {
	case PlatformX:
		m := xStatusPath.FindStringSubmatch(u.Path)
		if len(m) == 3 {
			return m[2]
		}
	case PlatformFacebook:
		q := u.Query()
		if v := q.Get("v"); v != "" {
			return v
		}
		if v := q.Get("fbid"); v != "" {
			return v
		}
		if v := q.Get("story_fbid"); v != "" {
			return v
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}
	return ""
}
