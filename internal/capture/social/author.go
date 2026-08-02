package social

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

var xHandleFromHTML = regexp.MustCompile(`\(@([A-Za-z0-9_]+)\)`)

// Author holds social post author metadata.
type Author struct {
	DisplayName string
	Handle      string // X only
	ProfileURL  string
	AvatarURL   string // remote URL before download
	AvatarPath  string // relative assets/ path after download
}

// BuildAuthorFromOEmbed constructs author metadata from oEmbed fields.
func BuildAuthorFromOEmbed(platform Platform, authorName, authorURL, html string) Author {
	a := Author{
		DisplayName: strings.TrimSpace(authorName),
		ProfileURL:  strings.TrimSpace(authorURL),
	}
	if platform == PlatformX {
		a.Handle = extractXHandle(authorURL, html)
	}
	return a
}

func extractXHandle(profileURL, html string) string {
	if h := handleFromProfileURL(profileURL); h != "" {
		return h
	}
	return handleFromOEmbedHTML(html)
}

func handleFromProfileURL(profileURL string) string {
	u, err := url.Parse(strings.TrimSpace(profileURL))
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	if host != "x.com" && host != "twitter.com" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 1 || parts[0] == "" {
		return ""
	}
	segment := parts[0]
	if segment == "status" || segment == "i" || segment == "intent" {
		return ""
	}
	for _, r := range segment {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return ""
		}
	}
	return segment
}

func handleFromOEmbedHTML(html string) string {
	m := xHandleFromHTML.FindStringSubmatch(html)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// AuthorExtraMap returns the nested author object for entry frontmatter.
func (a Author) AuthorExtraMap() map[string]any {
	if a.DisplayName == "" && a.Handle == "" && a.ProfileURL == "" && a.AvatarPath == "" {
		return nil
	}
	out := map[string]any{}
	if a.DisplayName != "" {
		out["display_name"] = a.DisplayName
	}
	if a.Handle != "" {
		out["handle"] = a.Handle
	}
	if a.ProfileURL != "" {
		out["profile_url"] = a.ProfileURL
	}
	if a.AvatarPath != "" {
		out["avatar"] = a.AvatarPath
	}
	return out
}

// SetAuthorAvatar updates the avatar path in entry Extra author map.
func SetAuthorAvatar(e *vault.Entry, avatarRelPath string) {
	if e == nil || e.Extra == nil || avatarRelPath == "" {
		return
	}
	raw, ok := e.Extra["author"]
	if !ok {
		return
	}
	author, ok := raw.(map[string]any)
	if !ok {
		return
	}
	author["avatar"] = avatarRelPath
}
