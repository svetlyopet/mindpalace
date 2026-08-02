package social

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

const (
	maxAssetBytes = 5 << 20
	maxAssetCount = 50
	assetsDirName = "assets"
)

// BuildOptions configures social entry construction.
type BuildOptions struct {
	Title string
	Type  vault.Type
}

// BuildEntry constructs a vault entry from a parsed social post.
// Image paths use assets/{hash}.ext placeholders rewritten after SaveMedia.
func BuildEntry(post *Post, opts BuildOptions) (*vault.Entry, string) {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = defaultTitle(post)
	}

	body := buildBody(post)
	plain := plainText(post)

	entryType := vault.TypeSocial
	if opts.Type != "" {
		entryType = opts.Type
	}

	e := &vault.Entry{
		Type:   entryType,
		Title:  title,
		Source: post.CanonicalURL,
		Body:   body,
		Extra: map[string]any{
			"platform": string(post.Platform),
		},
	}
	if authorExtra := post.Author.AuthorExtraMap(); authorExtra != nil {
		e.Extra["author"] = authorExtra
	}
	if post.Author.DisplayName != "" {
		e.Extra["author_name"] = post.Author.DisplayName
	}
	if post.Author.ProfileURL != "" {
		e.Extra["author_url"] = post.Author.ProfileURL
	}
	if post.PostID != "" {
		e.Extra["post_id"] = post.PostID
	}
	return e, plain
}

func defaultTitle(post *Post) string {
	snippet := post.Text
	snippet = strings.Join(strings.Fields(snippet), " ")
	if len(snippet) > 60 {
		snippet = snippet[:60] + "…"
	}
	if post.Author.DisplayName != "" && snippet != "" {
		return post.Author.DisplayName + ": " + snippet
	}
	if post.Author.DisplayName != "" {
		return post.Author.DisplayName
	}
	if snippet != "" {
		return snippet
	}
	return post.CanonicalURL
}

func buildBody(post *Post) string {
	var b strings.Builder
	b.WriteString("## Post\n\n")
	if post.Text != "" {
		b.WriteString(post.Text)
	} else {
		b.WriteString("_Post with media only._")
	}

	var mediaLines []string
	for _, img := range post.Images {
		rel := assetRelPath(img.URL, guessImageExt(img.URL))
		alt := img.Alt
		if alt == "" {
			alt = "image"
		}
		mediaLines = append(mediaLines, fmt.Sprintf("![%s](%s)", escapeMDAlt(alt), rel))
	}
	for _, vid := range post.Videos {
		if vid.PosterURL != "" {
			rel := assetRelPath(vid.PosterURL, guessImageExt(vid.PosterURL))
			mediaLines = append(mediaLines, fmt.Sprintf("[![%s](%s)](%s)", escapeMDAlt(vid.Label), rel, vid.LinkURL))
		} else if vid.LinkURL != "" {
			mediaLines = append(mediaLines, fmt.Sprintf("[%s](%s)", escapeMDAlt(vid.Label), vid.LinkURL))
		}
	}
	if len(mediaLines) > 0 {
		b.WriteString("\n\n## Media\n\n")
		b.WriteString(strings.Join(mediaLines, "\n\n"))
	}
	return appendPostThoughts(b.String(), post)
}

func appendPostThoughts(body string, post *Post) string {
	thoughts := strings.TrimSpace(post.Thoughts)
	if thoughts == "" {
		return body
	}
	section := "## Thoughts\n\n" + thoughts
	body = strings.TrimSpace(body)
	if body == "" {
		return section
	}
	return body + "\n\n" + section
}

func plainText(post *Post) string {
	var parts []string
	if post.Text != "" {
		parts = append(parts, post.Text)
	}
	for _, img := range post.Images {
		if img.Alt != "" {
			parts = append(parts, img.Alt)
		}
	}
	if thoughts := strings.TrimSpace(post.Thoughts); thoughts != "" {
		parts = append(parts, thoughts)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func escapeMDAlt(s string) string {
	return strings.ReplaceAll(s, "]", "\\]")
}

func assetRelPath(rawURL, ext string) string {
	return assetsDirName + "/" + assetFilename(rawURL, ext)
}

func assetFilename(rawURL, ext string) string {
	sum := sha256.Sum256([]byte(rawURL))
	name := hex.EncodeToString(sum[:8])
	ext = strings.ToLower(ext)
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}

func guessImageExt(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ".jpg"
	}
	ext := strings.ToLower(filepath.Ext(u.Path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return ext
	default:
		return ".jpg"
	}
}

// SaveMedia downloads remote images and video posters into entryDir/assets.
// It returns an updated body with /ui/entry/{entryID}/file/assets/... paths.
func SaveMedia(ctx context.Context, client *http.Client, entryID, entryDir string, post *Post) (string, []string) {
	if client == nil {
		client = http.DefaultClient
	}
	var warnings []string
	assetsDir := filepath.Join(entryDir, assetsDirName)
	if err := os.MkdirAll(assetsDir, fsperm.DirMode); err != nil {
		warnings = append(warnings, "social media: mkdir assets: "+err.Error())
		return buildBodyWithEntryPaths(entryID, post, map[string]string{}), warnings
	}

	downloaded := 0
	urlToRel := map[string]string{}

	download := func(rawURL string, ext string) string {
		if rawURL == "" {
			return ""
		}
		if rel, ok := urlToRel[rawURL]; ok {
			return rel
		}
		if downloaded >= maxAssetCount {
			warnings = append(warnings, "social media: asset limit reached")
			return assetRelPath(rawURL, ext)
		}
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme != "http" && u.Scheme != "https" {
			warnings = append(warnings, fmt.Sprintf("social media: invalid URL %q", rawURL))
			return assetRelPath(rawURL, ext)
		}
		data, gotExt, err := fetchImage(ctx, client, u)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("social media: fetch %s: %v", rawURL, err))
			return assetRelPath(rawURL, ext)
		}
		if gotExt != "" {
			ext = gotExt
		}
		name := assetFilename(rawURL, ext)
		dest := filepath.Join(assetsDir, name)
		if err := os.WriteFile(dest, data, fsperm.PrivateFileMode); err != nil {
			warnings = append(warnings, fmt.Sprintf("social media: write %s: %v", name, err))
			return assetRelPath(rawURL, ext)
		}
		downloaded++
		rel := assetsDirName + "/" + name
		urlToRel[rawURL] = rel
		return rel
	}

	for _, img := range post.Images {
		download(img.URL, guessImageExt(img.URL))
	}
	for _, vid := range post.Videos {
		if vid.PosterURL != "" {
			download(vid.PosterURL, guessImageExt(vid.PosterURL))
		}
	}

	return buildBodyWithEntryPaths(entryID, post, urlToRel), warnings
}

func buildBodyWithEntryPaths(entryID string, post *Post, urlToRel map[string]string) string {
	prefix := "/ui/entry/" + entryID + "/file/"
	resolve := func(rawURL, ext string) string {
		if rel, ok := urlToRel[rawURL]; ok {
			return prefix + rel
		}
		return prefix + assetRelPath(rawURL, ext)
	}

	var b strings.Builder
	b.WriteString("## Post\n\n")
	if post.Text != "" {
		b.WriteString(post.Text)
	} else {
		b.WriteString("_Post with media only._")
	}

	var mediaLines []string
	for _, img := range post.Images {
		path := resolve(img.URL, guessImageExt(img.URL))
		alt := img.Alt
		if alt == "" {
			alt = "image"
		}
		mediaLines = append(mediaLines, fmt.Sprintf("![%s](%s)", escapeMDAlt(alt), path))
	}
	for _, vid := range post.Videos {
		if vid.PosterURL != "" {
			path := resolve(vid.PosterURL, guessImageExt(vid.PosterURL))
			mediaLines = append(mediaLines, fmt.Sprintf("[![%s](%s)](%s)", escapeMDAlt(vid.Label), path, vid.LinkURL))
		} else if vid.LinkURL != "" {
			mediaLines = append(mediaLines, fmt.Sprintf("[%s](%s)", escapeMDAlt(vid.Label), vid.LinkURL))
		}
	}
	if len(mediaLines) > 0 {
		b.WriteString("\n\n## Media\n\n")
		b.WriteString(strings.Join(mediaLines, "\n\n"))
	}
	return appendPostThoughts(b.String(), post)
}

func fetchImage(ctx context.Context, client *http.Client, u *url.URL) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mindpalace/1.0 (+https://github.com/svetlyopet/mindpalace)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxAssetBytes {
		return nil, "", fmt.Errorf("asset exceeds %d bytes", maxAssetBytes)
	}
	ext := extFromContentType(ct)
	if ext == "" {
		ext = filepath.Ext(u.Path)
	}
	if !isImageExt(ext) {
		return nil, "", fmt.Errorf("not an image (%s)", ct)
	}
	return data, ext, nil
}

func extFromContentType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

func isImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// FetchOGImage tries to read og:image from a post page as a poster fallback.
func FetchOGImage(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mindpalace/1.0 (+https://github.com/svetlyopet/mindpalace)")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	const maxBody = 512 << 10
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", err
	}
	return parseOGImage(string(data)), nil
}

func parseOGImage(html string) string {
	const marker = `property="og:image"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		idx = strings.Index(html, `property='og:image'`)
	}
	if idx < 0 {
		return ""
	}
	chunk := html[idx:]
	contentIdx := strings.Index(chunk, "content=")
	if contentIdx < 0 {
		return ""
	}
	chunk = chunk[contentIdx+len("content="):]
	chunk = strings.TrimSpace(chunk)
	if len(chunk) == 0 {
		return ""
	}
	quote := chunk[0]
	if quote != '"' && quote != '\'' {
		return ""
	}
	chunk = chunk[1:]
	end := strings.IndexByte(chunk, quote)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(chunk[:end])
}

// FinalizeSocialEntry downloads post media and author avatar, updates entry body/extra, and writes entry.md.
func FinalizeSocialEntry(ctx context.Context, client *http.Client, e *vault.Entry, post *Post) []string {
	var warnings []string
	if e == nil || post == nil {
		return warnings
	}
	body, mediaWarns := SaveMedia(ctx, client, e.ID, e.Dir, post)
	warnings = append(warnings, mediaWarns...)
	e.Body = body
	if avatarPath, avatarWarns := SaveAuthorAvatar(ctx, client, e.Dir, post.Author.AvatarURL); avatarPath != "" {
		warnings = append(warnings, avatarWarns...)
		SetAuthorAvatar(e, avatarPath)
		post.Author.AvatarPath = avatarPath
	} else {
		warnings = append(warnings, avatarWarns...)
	}
	return warnings
}

func authorAssetFilename(rawURL, ext string) string {
	sum := sha256.Sum256([]byte("author:" + rawURL))
	name := "author-" + hex.EncodeToString(sum[:8])
	ext = strings.ToLower(ext)
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}

// SaveAuthorAvatar downloads the author profile image into entryDir/assets.
func SaveAuthorAvatar(ctx context.Context, client *http.Client, entryDir, avatarURL string) (string, []string) {
	var warnings []string
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return "", warnings
	}
	if client == nil {
		client = http.DefaultClient
	}
	assetsDir := filepath.Join(entryDir, assetsDirName)
	if err := os.MkdirAll(assetsDir, fsperm.DirMode); err != nil {
		warnings = append(warnings, "social author avatar: mkdir assets: "+err.Error())
		return "", warnings
	}
	u, err := url.Parse(avatarURL)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" {
		warnings = append(warnings, fmt.Sprintf("social author avatar: invalid URL %q", avatarURL))
		return "", warnings
	}
	data, gotExt, err := fetchImage(ctx, client, u)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("social author avatar: fetch %s: %v", avatarURL, err))
		return "", warnings
	}
	ext := guessImageExt(avatarURL)
	if gotExt != "" {
		ext = gotExt
	}
	name := authorAssetFilename(avatarURL, ext)
	dest := filepath.Join(assetsDir, name)
	if err := os.WriteFile(dest, data, fsperm.PrivateFileMode); err != nil {
		warnings = append(warnings, fmt.Sprintf("social author avatar: write %s: %v", name, err))
		return "", warnings
	}
	return assetsDirName + "/" + name, warnings
}

// EnrichAuthorAvatar resolves profile og:image into Author.AvatarURL.
func EnrichAuthorAvatar(ctx context.Context, client *http.Client, post *Post) []string {
	var warnings []string
	if post == nil || strings.TrimSpace(post.Author.ProfileURL) == "" || strings.TrimSpace(post.Author.AvatarURL) != "" {
		return warnings
	}
	avatar, err := FetchOGImage(ctx, client, post.Author.ProfileURL)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("social: author avatar resolve failed for %s: %v", post.Author.ProfileURL, err))
		return warnings
	}
	if avatar != "" {
		post.Author.AvatarURL = avatar
	}
	return warnings
}

// EnrichPhotoImages resolves X photo short links to direct pbs.twimg.com image URLs.
func EnrichPhotoImages(ctx context.Context, client *http.Client, post *Post) []string {
	var warnings []string
	seen := map[string]bool{}
	var enriched []MediaRef
	for _, img := range post.Images {
		raw := strings.TrimSpace(img.URL)
		if raw == "" {
			continue
		}
		if u, err := url.Parse(raw); err == nil && strings.EqualFold(u.Host, "pbs.twimg.com") {
			if !seen[raw] {
				seen[raw] = true
				enriched = append(enriched, img)
			}
			continue
		}
		resolved, err := FetchOGImage(ctx, client, raw)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("social: photo resolve failed for %s: %v", raw, err))
			if !seen[raw] {
				seen[raw] = true
				enriched = append(enriched, img)
			}
			continue
		}
		if resolved == "" {
			if !seen[raw] {
				seen[raw] = true
				enriched = append(enriched, img)
			}
			continue
		}
		if !seen[resolved] {
			seen[resolved] = true
			enriched = append(enriched, MediaRef{
				URL: resolved,
				Alt: img.Alt,
			})
		}
	}
	post.Images = enriched
	return warnings
}

// EnrichVideoPosters adds og:image posters for videos missing poster URLs.
func EnrichVideoPosters(ctx context.Context, client *http.Client, post *Post) []string {
	var warnings []string
	for i := range post.Videos {
		if post.Videos[i].PosterURL != "" {
			continue
		}
		link := post.Videos[i].LinkURL
		if link == "" {
			link = post.CanonicalURL
		}
		poster, err := FetchOGImage(ctx, client, link)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("social: poster fetch failed for %s: %v", link, err))
			continue
		}
		if poster != "" {
			post.Videos[i].PosterURL = poster
		}
	}
	return warnings
}
