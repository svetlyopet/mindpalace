package capture

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/capture/social"
	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// PreviewSocial returns metadata for a public X or Facebook post URL (oEmbed-only).
func (c *Capturer) PreviewSocial(ctx context.Context, link string, opts Options) (*Preview, error) {
	res, err := c.buildSocialResult(ctx, link, opts, true)
	if err != nil {
		return nil, err
	}
	title := res.entry.Title
	if t := strings.TrimSpace(opts.Title); t != "" {
		title = t
	}
	suggested, _ := c.suggestForIndex(ctx, title, res.plain)
	return &Preview{
		Title:         title,
		Type:          vault.TypeSocial,
		SuggestedTags: suggested,
	}, nil
}

// Social captures a public X or Facebook post via oEmbed (strict: no Readability fallback).
func (c *Capturer) Social(ctx context.Context, link string, opts Options) (*Result, error) {
	res, err := c.buildSocialResult(ctx, link, opts, true)
	if err != nil {
		return nil, err
	}
	return c.persistSocialResult(ctx, res, opts)
}

func (c *Capturer) buildSocialResult(ctx context.Context, link string, opts Options, strict bool) (*socialBuildResult, error) {
	if !c.cfg.SocialOEmbed {
		if strict {
			return nil, fmt.Errorf("social oEmbed capture is disabled in config")
		}
		return nil, nil
	}
	plat, canon, ok := social.Match(link)
	if !ok {
		if strict {
			return nil, fmt.Errorf("not a supported social post URL")
		}
		return nil, nil
	}
	post, err := social.Fetch(ctx, c.client, plat, canon)
	if err != nil {
		if strict {
			return nil, fmt.Errorf("social oembed: %w", err)
		}
		return &socialBuildResult{warnings: []string{"social: oEmbed failed, used Readability fallback: " + err.Error()}}, nil
	}
	warnings := social.EnrichVideoPosters(ctx, c.client, post)
	warnings = append(warnings, social.EnrichPhotoImages(ctx, c.client, post)...)
	warnings = append(warnings, social.EnrichAuthorAvatar(ctx, c.client, post)...)
	post.Thoughts = strings.TrimSpace(opts.Thoughts)
	buildOpts := social.BuildOptions{Type: vault.TypeSocial}
	if title := strings.TrimSpace(opts.Title); title != "" {
		buildOpts.Title = title
	}
	e, plain := social.BuildEntry(post, buildOpts)
	return &socialBuildResult{entry: e, plain: plain, post: post, warnings: warnings}, nil
}

func (c *Capturer) persistSocialResult(ctx context.Context, socialRes *socialBuildResult, opts Options) (*Result, error) {
	res, err := c.applyTags(ctx, socialRes.entry, socialRes.plain, opts)
	if err != nil {
		return nil, err
	}
	res.Warnings = append(res.Warnings, socialRes.warnings...)
	if err := c.vault.Create(res.Entry); err != nil {
		return nil, err
	}
	if socialRes.post != nil {
		res.Warnings = append(res.Warnings, social.FinalizeSocialEntry(ctx, c.client, res.Entry, socialRes.post)...)
		if err := vault.WriteEntry(res.Entry.Dir, res.Entry, c.vault.Cipher()); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(filepath.Join(res.Entry.Dir, "extracted.txt"), []byte(socialRes.plain), fsperm.PrivateFileMode); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Capturer) trySocialEntry(ctx context.Context, link string, opts Options) (*socialBuildResult, bool) {
	res, err := c.buildSocialResult(ctx, link, opts, false)
	if err != nil || res == nil {
		return nil, false
	}
	if res.entry != nil {
		return res, true
	}
	if len(res.warnings) > 0 {
		return res, false
	}
	return nil, false
}
