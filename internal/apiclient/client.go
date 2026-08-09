package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

const (
	defaultTimeout = 60 * time.Second
	probeTimeout   = 300 * time.Millisecond
)

// ProbeStatus is the result of probing a Mindpalace serve endpoint.
type ProbeStatus int

const (
	ProbeDown ProbeStatus = iota
	ProbeReady
	ProbeLocked
	ProbeOther
)

// Client talks to a running mp serve HTTP API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func New(addr, token string) *Client {
	base := strings.TrimRight(addr, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return &Client{
		BaseURL: base,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ProbeSession checks GET /api/session without requiring a Bearer token.
func ProbeSession(addr string) ProbeStatus {
	base := strings.TrimRight(addr, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/session", nil)
	if err != nil {
		return ProbeDown
	}
	client := &http.Client{Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeDown
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	switch resp.StatusCode {
	case http.StatusNoContent:
		return ProbeReady
	case http.StatusUnauthorized:
		if strings.TrimSpace(string(body)) == "locked" {
			return ProbeLocked
		}
		return ProbeOther
	default:
		return ProbeOther
	}
}

type CaptureReq struct {
	Kind     string    `json:"kind"`
	Text     string    `json:"text,omitempty"`
	URL      string    `json:"url,omitempty"`
	HTML     string    `json:"html,omitempty"`
	Title    string    `json:"title,omitempty"`
	Tags     *[]string `json:"tags,omitempty"`
	Type     string    `json:"type,omitempty"`
	Full     bool      `json:"full,omitempty"`
	Thoughts string    `json:"thoughts,omitempty"`
}

type CapturePreview struct {
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	SuggestedTags []string `json:"suggested_tags"`
}

func (c *Client) Session(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/api/session", nil, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return mapStatus(c.BaseURL, resp, nil)
}

func (c *Client) Unlock(ctx context.Context, password string) error {
	body, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/unlock", bytes.NewReader(body), false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return mapStatus(c.BaseURL, resp, nil)
}

func (c *Client) Lock(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodPost, "/api/lock", nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return mapStatus(c.BaseURL, resp, nil)
}

func (c *Client) ListEntries(ctx context.Context, q search.Query) ([]dto.SearchHit, error) {
	u, err := url.Parse(c.BaseURL + "/api/entries")
	if err != nil {
		return nil, err
	}
	vals := u.Query()
	if q.Text != "" {
		vals.Set("q", q.Text)
	}
	for _, tag := range q.Tags {
		vals.Add("tag", tag)
	}
	for _, typ := range q.Types {
		vals.Add("type", string(typ))
	}
	if !q.Since.IsZero() {
		// API ParseSince accepts YYYY-MM-DD (not RFC3339).
		vals.Set("since", q.Since.UTC().Format("2006-01-02"))
	}
	if q.Domain != "" {
		vals.Set("domain", q.Domain)
	}
	if q.Limit > 0 {
		vals.Set("limit", strconv.Itoa(q.Limit))
	}
	u.RawQuery = vals.Encode()

	resp, err := c.doURL(ctx, http.MethodGet, u.String(), nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var hits []dto.SearchHit
	if err := decodeJSON(c.BaseURL, resp, &hits); err != nil {
		return nil, err
	}
	return hits, nil
}

func (c *Client) GetEntry(ctx context.Context, id string) (dto.Entry, error) {
	var e dto.Entry
	resp, err := c.do(ctx, http.MethodGet, "/api/entries/"+url.PathEscape(id), nil, true)
	if err != nil {
		return e, err
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &e); err != nil {
		return e, err
	}
	return e, nil
}

func (c *Client) DeleteEntry(ctx context.Context, id string) (dto.DeleteResponse, error) {
	var out dto.DeleteResponse
	resp, err := c.do(ctx, http.MethodDelete, "/api/entries/"+url.PathEscape(id), nil, true)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) UpdateTags(ctx context.Context, id string, add, remove []string) (dto.Entry, error) {
	var e dto.Entry
	body, err := json.Marshal(map[string][]string{"add": add, "remove": remove})
	if err != nil {
		return e, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/entries/"+url.PathEscape(id)+"/tags", bytes.NewReader(body), true)
	if err != nil {
		return e, err
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &e); err != nil {
		return e, err
	}
	return e, nil
}

func (c *Client) Tags(ctx context.Context) ([]dto.TagCount, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/tags", nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tags []dto.TagCount
	if err := decodeJSON(c.BaseURL, resp, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (c *Client) Capture(ctx context.Context, req CaptureReq) (dto.CaptureResponse, error) {
	var out dto.CaptureResponse
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/capture", bytes.NewReader(body), true)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) CapturePreview(ctx context.Context, req CaptureReq) (CapturePreview, error) {
	var out CapturePreview
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/capture/preview", bytes.NewReader(body), true)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) CaptureUpload(ctx context.Context, filename string, data []byte, title string, tags []string, thoughts string) (dto.CaptureResponse, error) {
	var out dto.CaptureResponse
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return out, err
	}
	if _, err := part.Write(data); err != nil {
		return out, err
	}
	if title != "" {
		_ = w.WriteField("title", title)
	}
	if thoughts != "" {
		_ = w.WriteField("thoughts", thoughts)
	}
	if tags != nil {
		raw, err := json.Marshal(tags)
		if err != nil {
			return out, err
		}
		_ = w.WriteField("tags", string(raw))
	}
	if err := w.Close(); err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/capture/upload", &buf)
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return out, fmt.Errorf("mindpalace server at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	if err := decodeJSON(c.BaseURL, resp, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, auth bool) (*http.Response, error) {
	return c.doURL(ctx, method, c.BaseURL+path, body, auth)
}

func (c *Client) doURL(ctx context.Context, method, rawURL string, body io.Reader, auth bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("mindpalace server at %s: %w", c.BaseURL, err)
	}
	return resp, nil
}

func decodeJSON(base string, resp *http.Response, dest any) error {
	if err := mapStatus(base, resp, dest); err != nil {
		return err
	}
	if dest == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("mindpalace server at %s: decode response: %w", base, err)
	}
	return nil
}

func mapStatus(base string, resp *http.Response, dest any) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		if dest == nil || resp.StatusCode == http.StatusNoContent {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
		return nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return vault.ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		msg := strings.TrimSpace(string(b))
		switch msg {
		case "locked":
			return vault.ErrLocked
		case "wrong password":
			return vault.ErrWrongPassword
		case "unauthorized":
			return fmt.Errorf("mindpalace server at %s: unauthorized", base)
		default:
			if msg == "" {
				msg = resp.Status
			}
			return fmt.Errorf("mindpalace server at %s: %s", base, msg)
		}
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("mindpalace server at %s: %s", base, msg)
	}
}

// ErrServeRunning is returned when a CLI operation cannot run while mp serve holds the vault.
var ErrServeRunning = errors.New("mp serve is running")

func RefuseReindex() error {
	return fmt.Errorf("%w; stop the server first (the UI refreshes the index automatically)", ErrServeRunning)
}

func RefuseEncryptionChange() error {
	return fmt.Errorf("%w; cannot change vault encryption while the server is running; stop the server first", ErrServeRunning)
}
