package library

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

var ErrForbiddenPath = errors.New("forbidden path")

// Library orchestrates vault, index, search, and capture workflows.
type Library struct {
	Vault    *vault.Vault
	Index    *index.Index
	Searcher *search.Searcher
	Capturer *capture.Capturer
}

func New(v *vault.Vault, ix *index.Index, sr *search.Searcher, c *capture.Capturer) *Library {
	return &Library{
		Vault:    v,
		Index:    ix,
		Searcher: sr,
		Capturer: c,
	}
}

func (l *Library) GetEntry(id string) (*vault.Entry, error) {
	return l.Vault.Get(id)
}

func (l *Library) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	return l.Searcher.Search(ctx, q)
}

func (l *Library) ListTags() []search.TagCount {
	return l.Searcher.Tags()
}

func (l *Library) IndexEntry(e *vault.Entry) error {
	if l.Index == nil || e == nil {
		return nil
	}
	return l.Index.Put(e)
}

type DeleteResult struct {
	ID      string
	Title   string
	Reindex index.RebuildStats
}

func (l *Library) DeleteEntry(ctx context.Context, id string) (DeleteResult, error) {
	var out DeleteResult
	e, err := l.Vault.Get(id)
	if err != nil {
		return out, err
	}
	out.ID = e.ID
	out.Title = e.Title
	if err := l.Vault.Delete(e.ID); err != nil {
		return out, err
	}
	stats, err := l.Index.Rebuild(ctx, l.Vault)
	if err != nil {
		return out, err
	}
	out.Reindex = stats
	return out, nil
}

func (l *Library) Reindex(ctx context.Context) (index.RebuildStats, error) {
	return l.Index.Rebuild(ctx, l.Vault)
}

func (l *Library) ReindexEntryAfterEdit(_ context.Context, id string) error {
	e, err := l.Vault.Get(id)
	if err != nil {
		return err
	}
	return l.IndexEntry(e)
}

func (l *Library) UpdateTags(_ context.Context, id string, add, remove []string) (*vault.Entry, error) {
	e, err := l.Vault.Get(id)
	if err != nil {
		return nil, err
	}
	e.Tags = MergeTags(e.Tags, add, remove)
	if err := l.Vault.Update(e); err != nil {
		return nil, err
	}
	if err := l.IndexEntry(e); err != nil {
		return nil, err
	}
	return e, nil
}

// MergeTags applies add/remove to existing tags and returns a sorted slice.
func MergeTags(existing, add, remove []string) []string {
	set := map[string]struct{}{}
	for _, t := range existing {
		set[t] = struct{}{}
	}
	for _, t := range remove {
		delete(set, t)
	}
	for _, t := range add {
		set[t] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func (l *Library) CommitCapture(_ context.Context, res *capture.Result) error {
	if res == nil || res.Entry == nil {
		return nil
	}
	return l.IndexEntry(res.Entry)
}

type EntryFile struct {
	Rel  string
	Data []byte
}

func (l *Library) ReadEntryFile(id, rel string) (EntryFile, error) {
	var out EntryFile
	e, err := l.Vault.Get(id)
	if err != nil {
		return out, err
	}
	if rel == "" {
		return out, vault.ErrNotFound
	}
	clean, abs, err := safeEntryPath(e.Dir, rel)
	if err != nil {
		return out, err
	}
	data, err := vault.ReadFileBytes(abs, l.Vault.Cipher())
	if err != nil {
		return out, err
	}
	out.Rel = clean
	out.Data = data
	return out, nil
}

func safeEntryPath(entryDir, rel string) (clean, abs string, err error) {
	clean = filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", ErrForbiddenPath
	}
	abs = filepath.Join(entryDir, clean)
	relCheck, err := filepath.Rel(entryDir, abs)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		return "", "", ErrForbiddenPath
	}
	return clean, abs, nil
}

// CaptureOptionsFromFields builds capture.Options shared by CLI and HTTP adapters.
func CaptureOptionsFromFields(title string, tags []string, tagsExplicit bool, typ vault.Type, fullHTML bool, thoughts string) capture.Options {
	opts := capture.Options{
		Title:        title,
		Tags:         tags,
		TagsExplicit: tagsExplicit,
		FullHTML:     fullHTML,
		Thoughts:     strings.TrimSpace(thoughts),
	}
	if typ != "" {
		opts.Type = typ
	}
	return opts
}
