package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestClient points a client at a handler with an isolated session dir.
func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	home := t.TempDir()
	c := &Client{Base: srv.URL, Home: home, http: srv.Client()}
	return c
}

// vaultHandler is a minimal backend: unlock issues a cookie + csrf, then
// CSRF-protected writes are enforced.
func vaultHandler(t *testing.T, store map[string]string) http.HandlerFunc {
	t.Helper()
	const csrf = "csrf-token-xyz"
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status" && r.Method == "GET":
			_, err := r.Cookie("sid")
			json.NewEncoder(w).Encode(map[string]any{
				"initialized": true,
				"unlocked":    err == nil,
			})
		case r.URL.Path == "/api/unlock" && r.Method == "POST":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["password"] != "right" {
				w.WriteHeader(401)
				json.NewEncoder(w).Encode(map[string]any{"error": "bad password"})
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "sess-1", HttpOnly: true})
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "csrf": csrf})
		case r.URL.Path == "/api/entries" && r.Method == "POST":
			if r.Header.Get("X-CSRF-Token") != csrf {
				w.WriteHeader(403)
				json.NewEncoder(w).Encode(map[string]any{"error": "bad csrf"})
				return
			}
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			store[body["name"]] = body["value"]
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/entries" && r.Method == "GET":
			names := make([]string, 0, len(store))
			for k := range store {
				names = append(names, k)
			}
			json.NewEncoder(w).Encode(map[string]any{"entries": names})
		case strings.HasPrefix(r.URL.Path, "/api/entries/") && r.Method == "GET":
			name := strings.TrimPrefix(r.URL.Path, "/api/entries/")
			v, ok := store[name]
			if !ok {
				w.WriteHeader(404)
				json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"value": v})
		default:
			http.NotFound(w, r)
		}
	}
}

func TestUnlockPersistsSession(t *testing.T) {
	store := map[string]string{}
	c := newTestClient(t, vaultHandler(t, store))

	if err := c.Unlock("right"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	// cookie + csrf persisted
	if _, err := os.Stat(filepath.Join(c.Home, ".vault-cookie")); err != nil {
		t.Errorf("cookie file missing: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(c.Home, ".vault-csrf")); err != nil || string(b) != "csrf-token-xyz" {
		t.Errorf("csrf file = %q, err %v", b, err)
	}
}

func TestSetRequiresCSRF(t *testing.T) {
	store := map[string]string{}
	c := newTestClient(t, vaultHandler(t, store))
	if err := c.Unlock("right"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := c.Set("K", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if store["K"] != "v" {
		t.Fatalf("store[K] = %q, want v", store["K"])
	}
}

func TestSetWithoutSessionFails(t *testing.T) {
	store := map[string]string{}
	c := newTestClient(t, vaultHandler(t, store))
	if err := c.Set("K", "v"); err == nil {
		t.Fatal("Set without csrf should fail")
	}
}

func TestUnlockBadPassword(t *testing.T) {
	c := newTestClient(t, vaultHandler(t, map[string]string{}))
	err := c.Unlock("wrong")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad password") {
		t.Fatalf("error = %v, want bad password", err)
	}
}

func TestGetMissing(t *testing.T) {
	c := newTestClient(t, vaultHandler(t, map[string]string{}))
	if _, err := c.Get("NOPE"); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestMaskerRedactsValueAndFragments(t *testing.T) {
	m := NewMasker("supersecret")
	got := string(m.Mask([]byte("token=supersecret and frag=supers")))
	if strings.Contains(got, "supersecret") || strings.Contains(got, "supers") {
		t.Fatalf("secret leaked: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("no redaction marker in %q", got)
	}
}

func TestMaskerEmptyValueIsNoop(t *testing.T) {
	m := NewMasker("")
	in := []byte("plain text")
	if string(m.Mask(in)) != string(in) {
		t.Fatal("empty masker should not alter text")
	}
}

func TestValidateName(t *testing.T) {
	ok := []string{"A", "OPENAI_API_KEY", "a.b-c_d", "x123"}
	bad := []string{"", "with space", "semi;colon", "sl/ash", strings.Repeat("x", 129)}
	for _, s := range ok {
		if err := ValidateName(s); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateName(s); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", s)
		}
	}
}
