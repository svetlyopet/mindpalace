package capture

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

type Tagger interface {
	SuggestTags(ctx context.Context, title, content string) ([]string, error)
}

// TagPrompter collects tags from the user; suggested tags are hints only.
type TagPrompter interface {
	PromptTags(ctx context.Context, title string, suggested []string) ([]string, error)
}

type Options struct {
	Title        string
	Tags         []string
	TagsExplicit bool
	Prompter     TagPrompter
	Type         vault.Type
	FullHTML     bool
}

type Result struct {
	Entry         *vault.Entry
	SuggestedTags []string
	OCRUsed       bool
	ExtractedText string
	Warnings      []string
}

type Capturer struct {
	vault  *vault.Vault
	tagger Tagger
	cfg    config.CaptureConfig
	client *http.Client
}

func New(v *vault.Vault, tagger Tagger, cfg config.CaptureConfig) *Capturer {
	return &Capturer{
		vault:  v,
		tagger: tagger,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Capturer) Note(ctx context.Context, body string, opts Options) (*Result, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("note body is empty")
	}
	if err := validateTitle(opts.Title); err != nil {
		return nil, err
	}
	e := &vault.Entry{
		Type:  vault.TypeNote,
		Body:  body,
		Title: opts.Title,
	}
	if opts.Type != "" {
		e.Type = opts.Type
	}
	res, err := c.applyTags(ctx, e, body, opts)
	if err != nil {
		return nil, err
	}
	if err := c.vault.Create(res.Entry); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Capturer) File(ctx context.Context, path string, opts Options) (*Result, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if err := validateTitle(opts.Title); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var res *Result
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		res, err = c.captureImage(ctx, path, opts)
	default:
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		body := string(data)
		e := &vault.Entry{
			Type:  vault.TypeSnippet,
			Title: opts.Title,
			Body:  body,
		}
		if opts.Type != "" {
			e.Type = opts.Type
		}
		res, err = c.applyTags(ctx, e, body, opts)
	}
	if err != nil {
		return nil, err
	}
	if err := c.vault.Create(res.Entry); err != nil {
		return nil, err
	}
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
		dest := filepath.Join(res.Entry.Dir, "screenshot"+ext)
		if ext == ".jpeg" {
			dest = filepath.Join(res.Entry.Dir, "screenshot.jpg")
		}
		if err := copyFile(path, dest); err != nil {
			return nil, err
		}
		if res.ExtractedText != "" {
			_ = os.WriteFile(filepath.Join(res.Entry.Dir, "extracted.txt"), []byte(res.ExtractedText), 0o644)
		}
	}
	return res, nil
}

func (c *Capturer) applyTags(ctx context.Context, e *vault.Entry, indexText string, opts Options) (*Result, error) {
	res := &Result{Entry: e, Warnings: []string{}}
	if opts.TagsExplicit {
		e.Tags = normalizeTags(opts.Tags)
		return res, nil
	}
	suggested, warns := c.suggestForIndex(ctx, e.Title, indexText)
	res.Warnings = append(res.Warnings, warns...)
	res.SuggestedTags = suggested
	if opts.Prompter != nil && c.cfg.AutoTag && c.tagger != nil {
		chosen, err := opts.Prompter.PromptTags(ctx, e.Title, suggested)
		if err != nil {
			return nil, err
		}
		e.Tags = normalizeTags(chosen)
		return res, nil
	}
	e.Tags = normalizeTags(opts.Tags)
	return res, nil
}

func (c *Capturer) captureImage(ctx context.Context, path string, opts Options) (*Result, error) {
	e := &vault.Entry{
		Type:  vault.TypeScreenshot,
		Title: opts.Title,
	}
	if opts.Type != "" {
		e.Type = opts.Type
	}
	var plain string
	ocrUsed := false
	warnings := []string{}
	if c.cfg.OCR != "off" {
		txt, used, warn := runOCR(path)
		if warn != "" {
			warnings = append(warnings, warn)
		} else {
			plain = txt
			ocrUsed = used
		}
	}
	res, err := c.applyTags(ctx, e, plain, opts)
	if err != nil {
		return nil, err
	}
	res.OCRUsed = ocrUsed
	res.Warnings = append(res.Warnings, warnings...)
	res.ExtractedText = plain
	return res, nil
}

func runOCR(imgPath string) (text string, used bool, warning string) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", false, "tesseract not found, screenshot not OCR'd"
	}
	out, err := exec.Command("tesseract", imgPath, "stdout").Output()
	if err != nil {
		return "", false, "tesseract failed: " + err.Error()
	}
	return strings.TrimSpace(string(out)), true, ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func socialHost(host string) bool {
	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")
	switch host {
	case "x.com", "twitter.com", "reddit.com", "bsky.app", "mastodon.social":
		return true
	}
	return strings.Contains(host, "mastodon.")
}

// ParseTitleEditorText returns the first non-empty line that does not start with #.
func ParseTitleEditorText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

// ParseTagEditorText parses tag editor buffer text: skips blank lines and lines starting with #.
func ParseTagEditorText(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return normalizeTags(lines)
}

func normalizeTags(in []string) []string {
	var out []string
	for _, t := range in {
		if t = normalizeTag(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func normalizeTag(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	s = re.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	return s
}

// KeywordTagger suggests tags via stopword-filtered term frequency.
type KeywordTagger struct{}

func (KeywordTagger) SuggestTags(_ context.Context, title, content string) ([]string, error) {
	text := title + " " + content
	freq := map[string]int{}
	for _, w := range tokenize(text) {
		if stopwords[w] {
			continue
		}
		if len(w) < 3 {
			continue
		}
		freq[w]++
	}
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	var tags []string
	for i := 0; i < len(pairs) && len(tags) < 5; i++ {
		tags = append(tags, pairs[i].k)
	}
	return tags, nil
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var words []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			words = append(words, b.String())
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "with": true, "this": true,
	"from": true, "are": true, "was": true, "were": true, "have": true, "has": true,
	"you": true, "your": true, "about": true, "into": true, "will": true, "can": true,
	"not": true, "but": true, "all": true, "one": true, "our": true, "out": true,
}
