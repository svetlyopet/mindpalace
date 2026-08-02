package server

import (
	"fmt"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

type socialAuthorView struct {
	DisplayName string
	Handle      string
	ProfileURL  string
	AvatarURL   string
	Platform    string
}

func socialAuthorFromEntry(e *vault.Entry) *socialAuthorView {
	if e == nil || e.Type != vault.TypeSocial {
		return nil
	}
	platform := extraString(e.Extra, "platform")
	display := extraString(e.Extra, "author_name")
	profile := extraString(e.Extra, "author_url")
	handle := ""
	avatar := ""
	if raw, ok := e.Extra["author"].(map[string]any); ok {
		if v := extraString(raw, "display_name"); v != "" {
			display = v
		}
		if v := extraString(raw, "profile_url"); v != "" {
			profile = v
		}
		handle = extraString(raw, "handle")
		avatar = extraString(raw, "avatar")
	}
	if display == "" && profile == "" {
		return nil
	}
	view := &socialAuthorView{
		DisplayName: display,
		Handle:      handle,
		ProfileURL:  profile,
		Platform:    platform,
	}
	if avatar != "" {
		view.AvatarURL = "/ui/entry/" + e.ID + "/file/" + strings.TrimPrefix(avatar, "/")
	}
	return view
}

func extraString(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	v, ok := extra[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprint(s))
	}
}
