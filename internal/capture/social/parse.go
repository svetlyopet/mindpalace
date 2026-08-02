package social

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Post holds extracted social post content before persistence.
type Post struct {
	Platform     Platform
	CanonicalURL string
	Author       Author
	PostID       string
	Text         string
	Images       []MediaRef
	Videos       []VideoRef
	Thoughts     string // user-provided commentary
}

// MediaRef is a remote image to download into assets/.
type MediaRef struct {
	URL string
	Alt string
}

// VideoRef is a video link with an optional poster image URL.
type VideoRef struct {
	LinkURL   string
	PosterURL string
	Label     string
}

type parsedOEmbed struct {
	Text   string
	Images []MediaRef
	Videos []VideoRef
}

// ParseOEmbedHTML extracts post text and media references from oEmbed blockquote HTML.
func ParseOEmbedHTML(rawHTML string) (*parsedOEmbed, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("social parse html: %w", err)
	}
	out := &parsedOEmbed{}
	walkOEmbed(doc, out)
	out.Text = strings.TrimSpace(out.Text)
	return out, nil
}

func walkOEmbed(n *html.Node, out *parsedOEmbed) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p":
			text := textContent(n)
			text = strings.TrimSpace(text)
			if text != "" {
				if out.Text != "" {
					out.Text += "\n\n"
				}
				out.Text += text
			}
		case "img":
			src := attrValue(n, "src")
			if src != "" && !strings.HasPrefix(src, "data:") {
				out.Images = append(out.Images, MediaRef{
					URL: src,
					Alt: attrValue(n, "alt"),
				})
			}
		case "a":
			href := attrValue(n, "href")
			label := strings.TrimSpace(textContent(n))
			switch {
			case isPhotoLink(href, label):
				out.Images = append(out.Images, MediaRef{
					URL: href,
					Alt: label,
				})
			case isVideoLink(href):
				if label == "" {
					label = "Video"
				}
				out.Videos = append(out.Videos, VideoRef{
					LinkURL: href,
					Label:   label,
				})
			}
		case "video":
			poster := attrValue(n, "poster")
			src := attrValue(n, "src")
			link := src
			if link == "" {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "source" {
						if s := attrValue(c, "src"); s != "" {
							link = s
							break
						}
					}
				}
			}
			if link != "" || poster != "" {
				out.Videos = append(out.Videos, VideoRef{
					LinkURL:   link,
					PosterURL: poster,
					Label:     "Video",
				})
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkOEmbed(c, out)
	}
}

func isPhotoLink(href, label string) bool {
	href = strings.TrimSpace(href)
	if href != "" {
		u, err := url.Parse(href)
		if err == nil && strings.EqualFold(u.Host, "pic.twitter.com") {
			return true
		}
	}
	label = strings.ToLower(strings.TrimSpace(label))
	return strings.HasPrefix(label, "pic.twitter.com/")
}

func isVideoLink(href string) bool {
	href = strings.TrimSpace(href)
	if href == "" {
		return false
	}
	u, err := url.Parse(href)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if host == "video.twimg.com" {
		return true
	}
	if strings.Contains(host, "facebook.com") && (strings.Contains(u.Path, "/videos/") || strings.Contains(u.Path, "/watch")) {
		return true
	}
	if host == "fb.watch" {
		return true
	}
	return false
}

func textContent(n *html.Node) string {
	var buf bytes.Buffer
	collectText(n, &buf)
	return buf.String()
}

func collectText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style") {
			continue
		}
		collectText(c, buf)
	}
}

func attrValue(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
