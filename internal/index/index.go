package index

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"github.com/svetlyopet/mindpalace/internal/fsutil"
	"github.com/svetlyopet/mindpalace/internal/vault"

	"github.com/blevesearch/bleve/v2/search/query"
)

const mappingVersion = "2"

type EntryMeta struct {
	ID      string      `json:"id"`
	Title   string      `json:"title"`
	Created time.Time   `json:"created"`
	Type    vault.Type  `json:"type"`
	Source  string      `json:"source,omitempty"`
	Tags    []string    `json:"tags,omitempty"`
	Dir     string      `json:"dir"`
	MTime   time.Time   `json:"mtime"`
}

type Index struct {
	vaultRoot string
	indexPath string
	metaPath  string
	bleve     bleve.Index
	meta      map[string]EntryMeta
}

type RebuildStats struct {
	Indexed int
	Removed int
	Took    time.Duration
}

func Open(vaultRoot string) (*Index, error) {
	vaultRoot = filepath.Clean(vaultRoot)
	ix := &Index{
		vaultRoot: vaultRoot,
		indexPath: vault.IndexDir(vaultRoot),
		metaPath:  vault.MetaPath(vaultRoot),
		meta:      make(map[string]EntryMeta),
	}

	if err := os.MkdirAll(filepath.Dir(ix.indexPath), fsperm.DirMode); err != nil {
		return nil, err
	}
	if err := ix.loadMeta(); err != nil {
		return nil, err
	}

	mappingOK, _ := ix.checkMappingVersion()
	idx, err := bleve.Open(ix.indexPath)
	if err != nil {
		idx, err = bleve.New(ix.indexPath, ix.buildMapping())
		if err != nil {
			return nil, fmt.Errorf("create bleve index: %w", err)
		}
		_ = ix.writeMappingVersion()
	} else if !mappingOK {
		_ = idx.Close()
		_ = os.RemoveAll(ix.indexPath)
		idx, err = bleve.New(ix.indexPath, ix.buildMapping())
		if err != nil {
			return nil, err
		}
		_ = ix.writeMappingVersion()
	}
	ix.bleve = idx
	if err := ix.reconcileMetaWithBleve(); err != nil {
		return nil, err
	}
	return ix, nil
}

func (ix *Index) reconcileMetaWithBleve() error {
	count, err := ix.bleve.DocCount()
	if err != nil {
		return err
	}
	if count == 0 && len(ix.meta) > 0 {
		ix.meta = make(map[string]EntryMeta)
		return ix.saveMeta()
	}
	return nil
}

func (ix *Index) Close() error {
	if ix.bleve != nil {
		return ix.bleve.Close()
	}
	return nil
}

func (ix *Index) Bleve() bleve.Index {
	return ix.bleve
}

func (ix *Index) All() []EntryMeta {
	out := make([]EntryMeta, 0, len(ix.meta))
	for _, m := range ix.meta {
		out = append(out, m)
	}
	return out
}

func (ix *Index) Get(id string) (EntryMeta, bool) {
	m, ok := ix.meta[id]
	return m, ok
}

func (ix *Index) Put(e *vault.Entry) error {
	if e == nil || e.Dir == "" {
		return fmt.Errorf("entry dir required")
	}
	st, err := os.Stat(filepath.Join(e.Dir, "entry.md"))
	if err != nil {
		return err
	}
	extracted, _ := os.ReadFile(filepath.Join(e.Dir, "extracted.txt"))
	meta := EntryMeta{
		ID:      e.ID,
		Title:   e.Title,
		Created: e.Created,
		Type:    e.Type,
		Source:  e.Source,
		Tags:    append([]string(nil), e.Tags...),
		Dir:     e.Dir,
		MTime:   st.ModTime(),
	}
	ix.meta[e.ID] = meta
	if err := ix.saveMeta(); err != nil {
		return err
	}
	doc := map[string]any{
		"id":        e.ID,
		"title":     e.Title,
		"body":      e.Body,
		"extracted": string(extracted),
		"tags":      append([]string(nil), e.Tags...),
		"type":      string(e.Type),
		"created":   e.Created,
		"domain":    domainFromSource(e.Source),
	}
	return ix.bleve.Index(e.ID, doc)
}

func (ix *Index) Delete(id string) error {
	delete(ix.meta, id)
	if err := ix.saveMeta(); err != nil {
		return err
	}
	return ix.bleve.Delete(id)
}

func (ix *Index) Rebuild(ctx context.Context, v *vault.Vault) (RebuildStats, error) {
	start := time.Now()
	stats := RebuildStats{}
	seen := make(map[string]struct{})
	err := v.Walk(func(e *vault.Entry) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := ix.Put(e); err != nil {
			return err
		}
		seen[e.ID] = struct{}{}
		stats.Indexed++
		return nil
	})
	if err != nil {
		return stats, err
	}
	for id := range ix.meta {
		if _, ok := seen[id]; !ok {
			if err := ix.Delete(id); err != nil {
				return stats, err
			}
			stats.Removed++
		}
	}
	stats.Took = time.Since(start)
	return stats, nil
}

func (ix *Index) Refresh(ctx context.Context, v *vault.Vault) error {
	if err := ix.reconcileMetaWithBleve(); err != nil {
		return err
	}
	onDisk := make(map[string]struct{})
	err := v.WalkDirs(func(dir string, mtime time.Time) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		e, err := vault.ReadEntry(dir, v.Cipher())
		if err != nil {
			return nil
		}
		onDisk[e.ID] = struct{}{}
		cached, ok := ix.meta[e.ID]
		if !ok || mtime.After(cached.MTime) {
			if err := ix.Put(e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for id := range ix.meta {
		if _, ok := onDisk[id]; !ok {
			if err := ix.Delete(id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ix *Index) buildMapping() mapping.IndexMapping {
	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = en.AnalyzerName
	title := bleve.NewTextFieldMapping()
	title.Analyzer = en.AnalyzerName
	body := bleve.NewTextFieldMapping()
	body.Analyzer = en.AnalyzerName
	tags := bleve.NewTextFieldMapping()
	tags.Analyzer = keyword.Name
	typeF := bleve.NewKeywordFieldMapping()
	domain := bleve.NewKeywordFieldMapping()
	created := bleve.NewDateTimeFieldMapping()

	doc := bleve.NewDocumentMapping()
	doc.AddFieldMappingsAt("title", title)
	doc.AddFieldMappingsAt("body", body)
	doc.AddFieldMappingsAt("extracted", body)
	doc.AddFieldMappingsAt("tags", tags)
	doc.AddFieldMappingsAt("type", typeF)
	doc.AddFieldMappingsAt("domain", domain)
	doc.AddFieldMappingsAt("created", created)
	im.AddDocumentMapping("_default", doc)
	return im
}

func (ix *Index) loadMeta() error {
	data, err := os.ReadFile(ix.metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var list []EntryMeta
	if err := json.Unmarshal(data, &list); err != nil {
		// corrupt cache — rebuild empty; Refresh/Rebuild will fix
		ix.meta = make(map[string]EntryMeta)
		return nil
	}
	for _, m := range list {
		ix.meta[m.ID] = m
	}
	return nil
}

func (ix *Index) saveMeta() error {
	list := make([]EntryMeta, 0, len(ix.meta))
	for _, m := range ix.meta {
		list = append(list, m)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(ix.metaPath)
	tmp, err := os.CreateTemp(dir, "meta-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = fsutil.CloseFile(tmp)
		fsutil.RemoveBestEffort(tmpPath)
		return err
	}
	if err := fsutil.CloseFile(tmp); err != nil {
		fsutil.RemoveBestEffort(tmpPath)
		return err
	}
	return os.Rename(tmpPath, ix.metaPath)
}

func (ix *Index) mappingVersionPath() string {
	return filepath.Join(filepath.Dir(ix.indexPath), "mapping_version")
}

func (ix *Index) checkMappingVersion() (bool, error) {
	data, err := os.ReadFile(ix.mappingVersionPath())
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) == mappingVersion, nil
}

func (ix *Index) writeMappingVersion() error {
	return os.WriteFile(ix.mappingVersionPath(), []byte(mappingVersion), fsperm.PrivateFileMode)
}

func domainFromSource(source string) string {
	if source == "" {
		return ""
	}
	if i := strings.Index(source, "://"); i >= 0 {
		source = source[i+3:]
	}
	if i := strings.IndexAny(source, "/?#"); i >= 0 {
		source = source[:i]
	}
	return strings.ToLower(strings.TrimPrefix(source, "www."))
}

// MatchAll builds a conjunction query for Bleve filters.
func MatchAll(qs ...query.Query) query.Query {
	if len(qs) == 0 {
		return bleve.NewMatchAllQuery()
	}
	if len(qs) == 1 {
		return qs[0]
	}
	cq := bleve.NewConjunctionQuery(qs...)
	return cq
}
