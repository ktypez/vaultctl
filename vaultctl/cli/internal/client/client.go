// Package client speaks to the vaultctl backend over HTTP: cookies, CSRF,
// and the REST surface. Session files live in a session dir (default: $HOME)
// and are Netscape-cookie compatible with the legacy Python CLI.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultURL = "http://127.0.0.1:8787"
	// HomeEnv overrides the session dir (cookie/csrf/password files).
	HomeEnv = "VAULTCTL_HOME"
)

// ErrNoPassword is returned when no master password source is available.
var ErrNoPassword = errors.New(
	"no password: set VAULT_PASSWORD or create ~/.vault-password (mode 0600)")

// Client holds a configured vault HTTP client.
type Client struct {
	Base string
	Home string
	http *http.Client
}

// Status mirrors GET /api/status.
type Status struct {
	Initialized bool
	Unlocked    bool
}

// AuditRecord is one line of data/audit.jsonl (values are never recorded).
type AuditRecord struct {
	TS       string `json:"ts"`
	Action   string `json:"action"`
	Name     string `json:"name"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	Remote   string `json:"remote"`
}

// APIError carries an HTTP status and the backend's error message.
type APIError struct {
	Code    int
	Message string
}

func (e APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s (HTTP %d)", e.Message, e.Code)
	}
	return fmt.Sprintf("HTTP %d", e.Code)
}

// New creates a client using environment overrides.
func New() *Client {
	base := os.Getenv("VAULT_URL")
	if base == "" {
		base = defaultURL
	}
	home := os.Getenv(HomeEnv)
	if home == "" {
		home = os.Getenv("HOME")
	}
	// Direct transport: bypass system http_proxy for local loopback connections.
	tr := &http.Transport{
		Proxy: nil,
	}
	return &Client{
		Base: strings.TrimSuffix(base, "/"),
		Home: home,
		http: &http.Client{Timeout: 10 * time.Second, Transport: tr},
	}
}

func (c *Client) cookieFile() string { return filepath.Join(c.Home, ".vault-cookie") }
func (c *Client) csrfFile() string   { return filepath.Join(c.Home, ".vault-csrf") }

// PasswordFile is where the master password may be cached (mode 0600).
func (c *Client) PasswordFile() string { return filepath.Join(c.Home, ".vault-password") }

// URL returns the backend base URL.
func (c *Client) URL() string { return c.Base }

// Password resolves the master password: $VAULT_PASSWORD, then the cached file.
func (c *Client) Password() (string, error) {
	if p := os.Getenv("VAULT_PASSWORD"); p != "" {
		return p, nil
	}
	b, err := os.ReadFile(c.PasswordFile())
	if err != nil {
		return "", ErrNoPassword
	}
	if p := strings.TrimSpace(string(b)); p != "" {
		return p, nil
	}
	return "", ErrNoPassword
}

// loadCookie reads the persisted session cookie (name, value).
func (c *Client) loadCookie() (string, string) {
	b, err := os.ReadFile(c.cookieFile())
	if err != nil {
		return "", ""
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		// Netscape: domain \t flag \t path \t secure \t expiry \t name \t value
		if f := strings.Split(ln, "\t"); len(f) >= 7 {
			return f[5], f[6]
		}
		// Bare value fallback.
		if !strings.Contains(ln, "\t") && !strings.Contains(ln, " ") {
			return "sid", ln
		}
	}
	return "", ""
}

func (c *Client) saveCookie(name, value string) error {
	domain := urlHost(c.Base)
	line := fmt.Sprintf("# Netscape HTTP Cookie File\n%s\tFALSE\t/\tFALSE\t0\t%s\t%s\n",
		domain, name, value)
	if err := os.WriteFile(c.cookieFile(), []byte(line), 0o600); err != nil {
		return err
	}
	return os.Chmod(c.cookieFile(), 0o600)
}

func urlHost(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" {
		return "127.0.0.1"
	}
	return u.Hostname()
}

func (c *Client) saveCSRF(token string) error {
	if err := os.WriteFile(c.csrfFile(), []byte(token), 0o600); err != nil {
		return err
	}
	return os.Chmod(c.csrfFile(), 0o600)
}

func (c *Client) readCSRF() string {
	b, err := os.ReadFile(c.csrfFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ClearSession removes cached cookie + CSRF files.
func (c *Client) ClearSession() {
	os.Remove(c.cookieFile())
	os.Remove(c.csrfFile())
}

// call performs a request and returns status code, decoded JSON, and error.
// Set-Cookie headers are persisted automatically.
func (c *Client) call(method, path string, body any) (int, map[string]any, error) {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if name, value := c.loadCookie(); value != "" {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if csrf := c.readCSRF(); csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot reach vault at %s: %w", c.Base, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	for _, ck := range resp.Cookies() {
		if ck.Value != "" {
			_ = c.saveCookie(ck.Name, ck.Value)
		}
	}

	var m map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			m = nil
		}
	}
	if resp.StatusCode >= 400 {
		msg := errMsg(m, "")
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return resp.StatusCode, m, APIError{Code: resp.StatusCode, Message: msg}
	}
	return resp.StatusCode, m, nil
}

func errMsg(m map[string]any, fallback string) string {
	if m == nil {
		return fallback
	}
	for _, k := range []string{"error", "message", "detail"} {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

func boolField(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	switch v := m[k].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case float64:
		return v != 0
	default:
		return false
	}
}

// Status queries the backend.
func (c *Client) Status() (Status, error) {
	_, m, err := c.call("GET", "/api/status", nil)
	if err != nil {
		return Status{}, err
	}
	return Status{
		Initialized: boolField(m, "initialized"),
		Unlocked:    boolField(m, "unlocked"),
	}, nil
}

// Unlock authenticates, persisting the session cookie and CSRF token.
func (c *Client) Unlock(pw string) error {
	_, m, err := c.call("POST", "/api/unlock", map[string]string{"password": pw})
	if err != nil {
		return err
	}
	if !boolField(m, "ok") {
		return APIError{Code: 401, Message: errMsg(m, "unlock failed")}
	}
	csrf := strField(m, "csrf")
	if csrf == "" {
		return APIError{Code: 500, Message: "unlock response missing csrf token"}
	}
	return c.saveCSRF(csrf)
}

func strField(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	s, _ := m[k].(string)
	return s
}

// Lock invalidates the server session and removes local state.
func (c *Client) Lock() error {
	_, _, err := c.call("POST", "/api/lock", nil)
	// Local wipe is correct even when the server session already expired.
	c.ClearSession()
	return err
}

// IsLocked reports whether the vault needs unlocking.
func (c *Client) IsLocked() bool {
	s, err := c.Status()
	return err != nil || !s.Unlocked
}

// EnsureUnlocked unlocks with the cached password when needed.
func (c *Client) EnsureUnlocked() error {
	if !c.IsLocked() {
		return nil
	}
	p, err := c.Password()
	if err != nil {
		return err
	}
	return c.Unlock(p)
}

// List returns entry names (backend sorts them).
func (c *Client) List() ([]string, error) {
	_, m, err := c.call("GET", "/api/entries", nil)
	if err != nil {
		return nil, err
	}
	raw, _ := m["entries"].([]any)
	names := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			names = append(names, s)
		}
	}
	return names, nil
}

// Has reports whether an entry exists.
func (c *Client) Has(name string) bool {
	names, err := c.List()
	if err != nil {
		return false
	}
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// Get retrieves a secret value (caller must treat it as confidential).
func (c *Client) Get(name string) (string, error) {
	_, m, err := c.call("GET", "/api/entries/"+url.PathEscape(name), nil)
	if err != nil {
		return "", err
	}
	v, ok := m["value"].(string)
	if !ok {
		return "", APIError{Code: 404, Message: errMsg(m, "not found")}
	}
	return v, nil
}

// Set creates or updates an entry.
func (c *Client) Set(name, value string) error {
	_, _, err := c.call("POST", "/api/entries", map[string]string{"name": name, "value": value})
	return err
}

// Delete removes an entry.
func (c *Client) Delete(name string) error {
	_, _, err := c.call("DELETE", "/api/entries/"+url.PathEscape(name), nil)
	return err
}

// ChangePassword rotates the master password and caches the new one.
func (c *Client) ChangePassword(current, next string) error {
	_, _, err := c.call("POST", "/api/change-password", map[string]string{
		"current": current,
		"new":     next,
		"confirm": next,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.PasswordFile(), []byte(next+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chmod(c.PasswordFile(), 0o600)
}

// Audit returns the last n audit records.
func (c *Client) Audit(n int) ([]AuditRecord, error) {
	_, m, err := c.call("GET", fmt.Sprintf("/api/audit?n=%d", n), nil)
	if err != nil {
		return nil, err
	}
	raw, _ := m["entries"].([]any)
	recs := make([]AuditRecord, 0, len(raw))
	for _, v := range raw {
		b, _ := json.Marshal(v)
		var r AuditRecord
		if json.Unmarshal(b, &r) == nil {
			recs = append(recs, r)
		}
	}
	return recs, nil
}

// Masker redacts a secret (and any 6+ char fragment) from text.
type Masker struct {
	value string
}

// NewMasker builds a masker for a known secret value.
func NewMasker(value string) *Masker { return &Masker{value: value} }

// Mask redacts every occurrence of the secret and its 6-char fragments.
func (m *Masker) Mask(b []byte) []byte {
	if m == nil || m.value == "" {
		return b
	}
	out := strings.ReplaceAll(string(b), m.value, "[REDACTED]")
	if len(m.value) >= 6 {
		for k := 0; k+5 < len(m.value); k++ {
			out = strings.ReplaceAll(out, m.value[k:k+6], "[REDACTED]")
		}
	}
	return []byte(out)
}

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)

// ValidName reports whether s is a safe entry name.
func ValidName(s string) bool {
	return len(s) >= 1 && len(s) <= 128 && nameRe.MatchString(s)
}

// ValidateName returns a user-friendly error for bad input, or nil.
func ValidateName(s string) error {
	if !ValidName(s) {
		return errors.New("invalid name: use letters, numbers, . - _ (1-128 chars)")
	}
	return nil
}
