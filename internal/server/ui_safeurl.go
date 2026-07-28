package server

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/search"
)

func safeHTTPURL(raw string) (href string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.String(), true
	default:
		return "", false
	}
}

func navLinkClass(activeNav, link string) string {
	if activeNav == link {
		return "nav-link is-active"
	}
	return "nav-link"
}

func tabBtnClass(active bool) string {
	if active {
		return "tab-btn is-active"
	}
	return "tab-btn"
}

func sourceLinkHTML(href, label string) template.HTML {
	if href == "" || label == "" {
		return ""
	}
	// nosemgrep: go.lang.security.audit.net.formatted-template-string.formatted-template-string -- href/label validated http(s) and html-escaped
	s := fmt.Sprintf(
		`<a class="source-link" href="%s" target="_blank" rel="noopener">%s</a>`,
		html.EscapeString(href),
		html.EscapeString(label),
	)
	return template.HTML(s)
}

func buildTypeFilterOptions(types []string, selected string) []typeFilterOption {
	out := make([]typeFilterOption, 0, len(types))
	for _, t := range types {
		out = append(out, typeFilterOption{
			Value:    t,
			Label:    t,
			Selected: t == selected,
		})
	}
	return out
}

func entryRowsFromResults(results []search.Result, selectedID string) []entryListRow {
	out := make([]entryListRow, len(results))
	for i, r := range results {
		cls := "entry-row"
		if r.Meta.ID == selectedID {
			cls = "entry-row is-selected"
		}
		out[i] = entryListRow{
			Meta:      r.Meta,
			Score:     r.Score,
			Fragments: r.Fragments,
			RowClass:  cls,
		}
	}
	return out
}
