package capture

import (
	"context"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/svetlyopet/mindpalace/internal/capture/social"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

type socialBuildResult struct {
	entry    *vault.Entry
	plain    string
	post     *social.Post
	warnings []string
}

func (c *Capturer) buildEntryFromHTML(ctx context.Context, link string, pageURL *url.URL, rawHTML []byte, opts Options) (*vault.Entry, string, *social.Post, []string, error) {
	title := strings.TrimSpace(opts.Title)
	if err := validateTitle(title); err != nil {
		return nil, "", nil, nil, err
	}
	socialRes, socialOK := c.trySocialEntry(ctx, link, opts)
	if socialOK {
		return socialRes.entry, socialRes.plain, socialRes.post, socialRes.warnings, nil
	}
	var warnings []string
	if socialRes != nil {
		warnings = socialRes.warnings
	}
	e, plain, err := entryFromHTML(link, pageURL, rawHTML, title, opts)
	return e, plain, nil, warnings, err
}

func (c *Capturer) previewEntryFromHTML(ctx context.Context, link string, pageURL *url.URL, rawHTML []byte, opts Options) (*vault.Entry, string, []string, error) {
	socialRes, socialOK := c.trySocialEntry(ctx, link, opts)
	if socialOK {
		if title := strings.TrimSpace(opts.Title); title != "" {
			socialRes.entry.Title = title
		}
		return socialRes.entry, socialRes.plain, socialRes.warnings, nil
	}
	var warnings []string
	if socialRes != nil {
		warnings = socialRes.warnings
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		article, err := readability.FromReader(strings.NewReader(string(rawHTML)), pageURL)
		if err != nil {
			return nil, "", warnings, err
		}
		title = strings.TrimSpace(article.Title())
		if title == "" {
			title = link
		}
	}
	e, plain, err := entryFromHTML(link, pageURL, rawHTML, title, opts)
	return e, plain, warnings, err
}
