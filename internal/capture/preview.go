package capture

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

// Preview holds extracted metadata for tag prompting without persisting an entry.
type Preview struct {
	Title         string
	Type          vault.Type
	SuggestedTags []string
}

func (c *Capturer) PreviewNote(ctx context.Context, body string, opts Options) (*Preview, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("note body is empty")
	}
	title := opts.Title
	if title == "" {
		title = FirstLineTitle(body)
	}
	entryType := vault.TypeNote
	if opts.Type != "" {
		entryType = opts.Type
	}
	suggested, _ := c.suggestForIndex(ctx, title, body)
	return &Preview{
		Title:         title,
		Type:          entryType,
		SuggestedTags: suggested,
	}, nil
}

func (c *Capturer) PreviewURL(ctx context.Context, link string, opts Options) (*Preview, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mindpalace/1.0 (+https://github.com/svetlyopet/mindpalace)")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch url: HTTP %d", resp.StatusCode)
	}
	const maxBody = 10 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("page exceeds size limit (%d MB)", maxBody>>20)
	}
	return c.PreviewHTML(ctx, link, body, opts)
}

func (c *Capturer) PreviewHTML(ctx context.Context, link string, rawHTML []byte, opts Options) (*Preview, error) {
	pageURL, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	e, plain, err := previewEntryFromHTML(link, pageURL, rawHTML, opts)
	if err != nil {
		return nil, err
	}
	suggested, _ := c.suggestForIndex(ctx, e.Title, plain)
	return &Preview{
		Title:         e.Title,
		Type:          e.Type,
		SuggestedTags: suggested,
	}, nil
}

func (c *Capturer) SuggestTags(ctx context.Context, title, indexText string) ([]string, error) {
	suggested, _ := c.suggestForIndex(ctx, title, indexText)
	return suggested, nil
}

func (c *Capturer) suggestForIndex(ctx context.Context, title, indexText string) ([]string, []string) {
	if !c.cfg.AutoTag || c.tagger == nil {
		return nil, nil
	}
	excerpt := indexText
	if len(excerpt) > 2000 {
		excerpt = excerpt[:2000]
	}
	suggested, err := c.tagger.SuggestTags(ctx, title, excerpt)
	if err != nil {
		return nil, []string{"auto-tag: " + err.Error()}
	}
	return suggested, nil
}

// FirstLineTitle suggests a title from the first non-empty line of note body text.
func FirstLineTitle(body string) string {
	line, _, _ := strings.Cut(body, "\n")
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimSpace(line)
	if line == "" {
		return "Untitled note"
	}
	if len(line) > 80 {
		return line[:80]
	}
	return line
}
