package capture

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/svetlyopet/mindpalace/internal/capture/social"
	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func (c *Capturer) URL(ctx context.Context, link string, opts Options) (*Result, error) {
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
	return c.ingestHTML(ctx, link, req.URL, body, opts)
}

// URLFromHTML captures a page from supplied HTML (extension / server-side fetch skip).
func (c *Capturer) URLFromHTML(ctx context.Context, link string, html []byte, opts Options) (*Result, error) {
	pageURL, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	return c.ingestHTML(ctx, link, pageURL, html, opts)
}

func entryFromHTML(link string, pageURL *url.URL, rawHTML []byte, title string, opts Options) (*vault.Entry, string, error) {
	article, err := readability.FromReader(strings.NewReader(string(rawHTML)), pageURL)
	if err != nil {
		return nil, "", fmt.Errorf("readability: %w", err)
	}
	var htmlContent strings.Builder
	if err := article.RenderHTML(&htmlContent); err != nil {
		return nil, "", fmt.Errorf("readability html: %w", err)
	}
	md, err := htmltomd.ConvertString(htmlContent.String())
	if err != nil {
		md = ""
	}
	var textContent strings.Builder
	if err := article.RenderText(&textContent); err != nil {
		return nil, "", fmt.Errorf("readability text: %w", err)
	}
	plain := strings.TrimSpace(textContent.String())
	if plain == "" {
		plain = strings.TrimSpace(md)
	}
	if md == "" {
		md = plain
	}

	entryType := vault.TypeArticle
	if opts.Type != "" {
		entryType = opts.Type
	} else if socialHost(pageURL.Hostname()) {
		entryType = vault.TypeSocial
	}

	excerpt := "## Captured excerpt\n\n" + strings.TrimSpace(md)
	body := AppendThoughts(excerpt, opts.Thoughts)
	plain = MergeIndexText(plain, opts.Thoughts)
	e := &vault.Entry{
		Type:   entryType,
		Title:  title,
		Source: link,
		Body:   body,
	}
	return e, plain, nil
}

func (c *Capturer) ingestHTML(ctx context.Context, link string, pageURL *url.URL, rawHTML []byte, opts Options) (*Result, error) {
	e, plain, post, warnings, err := c.buildEntryFromHTML(ctx, link, pageURL, rawHTML, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.applyTags(ctx, e, plain, opts)
	if err != nil {
		return nil, err
	}
	res.Warnings = append(res.Warnings, warnings...)
	if err := c.vault.Create(res.Entry); err != nil {
		return nil, err
	}
	if post != nil {
		res.Warnings = append(res.Warnings, social.FinalizeSocialEntry(ctx, c.client, res.Entry, post)...)
		if err := vault.WriteEntry(res.Entry.Dir, res.Entry, c.vault.Cipher()); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(filepath.Join(res.Entry.Dir, "extracted.txt"), []byte(plain), fsperm.PrivateFileMode); err != nil {
		return nil, err
	}
	if opts.FullHTML || c.cfg.FullHTML {
		res.Warnings = append(res.Warnings, bundlePage(ctx, c.client, pageURL, rawHTML, res.Entry.ID, res.Entry.Dir)...)
	}
	return res, nil
}
