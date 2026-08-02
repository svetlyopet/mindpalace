package social

import (
	"net/url"
	"regexp"
	"strings"
)

// Platform identifies a supported social host.
type Platform string

const (
	PlatformX        Platform = "x"
	PlatformFacebook Platform = "facebook"
)

var (
	xStatusPath = regexp.MustCompile(`^/([^/]+)/status/(\d+)`)
	fbWatchPath = regexp.MustCompile(`^/watch/?$`)
)

// Match returns the platform and canonical URL when link is a supported public post URL.
func Match(link string) (Platform, string, bool) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", false
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	switch host {
	case "x.com", "twitter.com":
		return matchX(u)
	case "facebook.com", "m.facebook.com":
		return matchFacebook(u)
	case "fb.watch":
		return PlatformFacebook, canonicalFacebook(u), true
	default:
		return "", "", false
	}
}

func matchX(u *url.URL) (Platform, string, bool) {
	m := xStatusPath.FindStringSubmatch(u.Path)
	if len(m) != 3 {
		return "", "", false
	}
	canon := &url.URL{
		Scheme: "https",
		Host:   "twitter.com",
		Path:   "/" + m[1] + "/status/" + m[2],
	}
	return PlatformX, canon.String(), true
}

func matchFacebook(u *url.URL) (Platform, string, bool) {
	path := u.Path
	q := u.Query()

	switch {
	case strings.HasSuffix(path, "/posts") || strings.Contains(path, "/posts/"):
		return PlatformFacebook, canonicalFacebook(u), true
	case path == "/photo.php" && q.Get("fbid") != "":
		return PlatformFacebook, canonicalFacebook(u), true
	case path == "/story.php" && (q.Get("story_fbid") != "" || q.Get("id") != ""):
		return PlatformFacebook, canonicalFacebook(u), true
	case path == "/permalink.php" && q.Get("story_fbid") != "":
		return PlatformFacebook, canonicalFacebook(u), true
	case fbWatchPath.MatchString(path) && q.Get("v") != "":
		return PlatformFacebook, canonicalFacebook(u), true
	case strings.HasPrefix(path, "/reel/"):
		return PlatformFacebook, canonicalFacebook(u), true
	case strings.HasPrefix(path, "/videos/"):
		return PlatformFacebook, canonicalFacebook(u), true
	default:
		return "", "", false
	}
}

func canonicalFacebook(u *url.URL) string {
	canon := *u
	canon.Scheme = "https"
	canon.Host = "www.facebook.com"
	return canon.String()
}
