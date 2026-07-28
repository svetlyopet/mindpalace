package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookie = "mp_session"

func (s *Server) sessionKey() []byte {
	sum := sha256.Sum256([]byte("mp-session:" + s.token + ":" + s.vaultFP))
	return sum[:]
}

func (s *Server) signSession(exp int64) string {
	m := hmac.New(sha256.New, s.sessionKey())
	_, _ = m.Write([]byte(strconv.FormatInt(exp, 10)))
	_, _ = m.Write([]byte(":"))
	_, _ = m.Write([]byte(s.vaultFP))
	sig := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	return strconv.FormatInt(exp, 10) + "." + sig
}

func (s *Server) verifySession(value string) bool {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return value == s.signSession(exp)
}

func sessionCookieSecure(r *http.Request) bool {
	return r != nil && r.TLS != nil
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Add(7 * 24 * time.Hour).Unix()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.signSession(exp),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   sessionCookieSecure(r),
		MaxAge:   7 * 24 * 3600,
	})
}

func (s *Server) checkSessionCookie(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return s.verifySession(c.Value)
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   sessionCookieSecure(r),
	})
}

func (s *Server) authorizeAPI(r *http.Request) bool {
	if s.lib.Vault.Encrypted() && !s.lib.Vault.Unlocked() {
		return false
	}
	if s.checkBearer(r) || s.checkSessionCookie(r) {
		return true
	}
	return false
}

func (s *Server) ensureBrowserSession(w http.ResponseWriter, r *http.Request) {
	if r != nil && s.checkSessionCookie(r) {
		return
	}
	if s.lib.Vault.Encrypted() && !s.lib.Vault.Unlocked() {
		return
	}
	s.setSessionCookie(w, r)
}

func (s *Server) requireUnlockedUI(w http.ResponseWriter, r *http.Request) bool {
	if !s.lib.Vault.Encrypted() || s.lib.Vault.Unlocked() {
		s.ensureBrowserSession(w, r)
		return true
	}
	if r.URL.Path == "/unlock" {
		return true
	}
	http.Redirect(w, r, "/unlock", http.StatusFound)
	return false
}
