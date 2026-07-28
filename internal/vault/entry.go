package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"gopkg.in/yaml.v3"
)

func ReadEntry(dir string, c *Cipher) (*Entry, error) {
	path := filepath.Join(dir, "entry.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseEntryMarkdown(data, dir, c)
}

func parseEntryMarkdown(data []byte, dir string, c *Cipher) (*Entry, error) {
	var raw map[string]any
	body, err := frontmatter.Parse(bytes.NewReader(data), &raw)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter in %s: %w", dir, err)
	}
	e := &Entry{
		Extra: make(map[string]any),
		Dir:   dir,
		Body:  string(body),
	}
	var legacyEnc bool
	var legacyBodyEnc string
	for k, v := range raw {
		switch strings.ToLower(k) {
		case "id":
			e.ID = asString(v)
		case "title":
			e.Title = asString(v)
		case "created":
			e.Created = parseTime(v)
		case "type":
			e.Type = Type(asString(v))
		case "source":
			e.Source = asString(v)
		case "tags":
			e.Tags = asStringSlice(v)
		case "mp_enc":
			legacyEnc = v == true || asString(v) == "true"
		case "mp_body_enc":
			legacyBodyEnc = asString(v)
		default:
			e.Extra[k] = v
		}
	}
	if e.ID == "" {
		return nil, fmt.Errorf("entry %s: missing id in frontmatter", dir)
	}
	if e.Title == "" {
		return nil, fmt.Errorf("entry %s: missing title in frontmatter", dir)
	}
	if e.Created.IsZero() {
		return nil, fmt.Errorf("entry %s: missing created in frontmatter", dir)
	}
	if e.Type == "" || !e.Type.Valid() {
		return nil, fmt.Errorf("entry %s: invalid or missing type", dir)
	}
	if legacyEnc || legacyBodyEnc != "" {
		if legacyBodyEnc == "" {
			return nil, fmt.Errorf("entry %s: missing mp_body_enc", dir)
		}
		if c == nil {
			return nil, ErrLocked
		}
		blob, err := DecodeBlob(legacyBodyEnc)
		if err != nil {
			return nil, err
		}
		plain, err := c.Decrypt(blob)
		if err != nil {
			return nil, err
		}
		e.Body = string(plain)
		return e, nil
	}
	bodyBytes := []byte(e.Body)
	if isEncryptedBlob(bodyBytes) {
		if c == nil {
			return nil, ErrLocked
		}
		plain, err := c.Decrypt(bodyBytes)
		if err != nil {
			return nil, err
		}
		e.Body = string(plain)
	}
	return e, nil
}

func isEncryptedBlob(b []byte) bool {
	return len(b) >= len(encMagic) && string(b[:len(encMagic)]) == encMagic
}

func WriteEntry(dir string, e *Entry, c *Cipher) error {
	if e.Extra == nil {
		e.Extra = make(map[string]any)
	}
	merged := make(map[string]any, len(e.Extra)+8)
	for k, v := range e.Extra {
		if strings.EqualFold(k, "mp_enc") || strings.EqualFold(k, "mp_body_enc") {
			continue
		}
		merged[k] = v
	}
	delete(merged, "mp_enc")
	delete(merged, "mp_body_enc")
	merged["id"] = e.ID
	merged["title"] = e.Title
	merged["created"] = e.Created.Format(time.RFC3339)
	merged["type"] = string(e.Type)
	if e.Source != "" {
		merged["source"] = e.Source
	} else {
		delete(merged, "source")
	}
	if len(e.Tags) > 0 {
		merged["tags"] = e.Tags
	} else {
		delete(merged, "tags")
	}

	var bodyBytes []byte
	if c != nil {
		enc, err := c.Encrypt([]byte(e.Body))
		if err != nil {
			return err
		}
		bodyBytes = enc
	} else {
		bodyBytes = []byte(e.Body)
	}

	var fm bytes.Buffer
	enc := yaml.NewEncoder(&fm)
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(fm.Bytes())
	out.WriteString("---\n")
	if len(bodyBytes) > 0 {
		out.Write(bodyBytes)
		if c == nil && !bytes.HasSuffix(bodyBytes, []byte("\n")) {
			out.WriteByte('\n')
		}
	}
	path := filepath.Join(dir, "entry.md")
	return os.WriteFile(path, out.Bytes(), fsperm.PrivateFileMode)
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(v)
	}
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			s := strings.TrimSpace(asString(x))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func parseTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if tm, err := time.Parse(time.RFC3339, t); err == nil {
			return tm
		}
		if tm, err := time.Parse("2006-01-02T15:04:05Z07:00", t); err == nil {
			return tm
		}
	}
	return time.Time{}
}
