package capture

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

const maxUploadSize = 10 << 20

func (c *Capturer) Upload(ctx context.Context, filename string, data []byte, opts Options) (*Result, error) {
	if len(data) > maxUploadSize {
		return nil, fmt.Errorf("file exceeds size limit (%d MB)", maxUploadSize>>20)
	}
	path, cleanup, err := writeUploadTemp(filename, data)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return c.File(ctx, path, opts)
}

func (c *Capturer) PreviewUpload(ctx context.Context, filename string, data []byte, opts Options) (*Preview, error) {
	if len(data) > maxUploadSize {
		return nil, fmt.Errorf("file exceeds size limit (%d MB)", maxUploadSize>>20)
	}
	base := safeUploadBasename(filename)
	title := opts.Title
	if title == "" {
		title = base
	}
	ext := strings.ToLower(filepath.Ext(base))
	var entryType vault.Type
	var indexText string

	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		entryType = vault.TypeScreenshot
		if opts.Type != "" {
			entryType = opts.Type
		}
		path, cleanup, err := writeUploadTemp(filename, data)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		if c.cfg.OCR != "off" {
			txt, _, warn := runOCR(path)
			if warn == "" {
				indexText = txt
			}
		}
	default:
		entryType = vault.TypeSnippet
		if opts.Type != "" {
			entryType = opts.Type
		}
		if isLikelyText(data) {
			indexText = string(data)
		}
	}

	suggested, _ := c.suggestForIndex(ctx, title, indexText)
	return &Preview{
		Title:         title,
		Type:          entryType,
		SuggestedTags: suggested,
	}, nil
}

func writeUploadTemp(filename string, data []byte) (path string, cleanup func(), err error) {
	base := safeUploadBasename(filename)
	ext := filepath.Ext(base)
	f, err := os.CreateTemp("", "mp-upload-*"+ext)
	if err != nil {
		return "", nil, err
	}
	path = f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func safeUploadBasename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		return "upload"
	}
	return name
}

func isLikelyText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	for _, b := range check {
		if b == 0 {
			return false
		}
	}
	return true
}
