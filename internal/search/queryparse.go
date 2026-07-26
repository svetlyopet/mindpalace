package search

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

// QueryFromURL parses list/search parameters from an HTTP URL query.
// If tag is set (non-empty), it replaces any tag query list with a single tag filter.
func QueryFromURL(u url.URL, now time.Time) (Query, error) {
	q := Query{Limit: 20}
	vals := u.Query()
	if v := vals.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return q, err
		}
		q.Limit = n
	}
	q.Text = vals.Get("q")
	if tag := vals.Get("tag"); tag != "" {
		q.Tags = []string{tag}
	} else {
		for _, tag := range vals["tag"] {
			if tag != "" {
				q.Tags = append(q.Tags, tag)
			}
		}
	}
	q.Domain = vals.Get("domain")
	for _, t := range vals["type"] {
		if t == "" {
			continue
		}
		typ := vault.Type(t)
		if !typ.Valid() {
			return q, errors.New("invalid type")
		}
		q.Types = append(q.Types, typ)
	}
	if since := vals.Get("since"); since != "" {
		tm, err := ParseSince(since, now)
		if err != nil {
			return q, err
		}
		q.Since = tm
	}
	return q, nil
}

// QueryFromListParams parses CLI list/search flags into a Query.
func QueryFromListParams(text string, tags, types []string, since, domain string, limit int, now time.Time) (Query, error) {
	q := Query{
		Text:   text,
		Tags:   append([]string(nil), tags...),
		Domain: domain,
		Limit:  limit,
	}
	for _, t := range types {
		typ := vault.Type(t)
		if !typ.Valid() {
			return q, errors.New("invalid type " + t)
		}
		q.Types = append(q.Types, typ)
	}
	if since != "" {
		tm, err := ParseSince(since, now)
		if err != nil {
			return q, err
		}
		q.Since = tm
	}
	return q, nil
}
