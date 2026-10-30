package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"storageanalysis/internal/config"
)

func TestOperationAuthorizationAndCSRF(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "allowed")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(allowed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(allowed, "delete.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(external, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 32)
	cfg := config.Config{
		OpsAuthToken:    token,
		OpsAllowedRoots: []string{allowed},
		AllowedOrigins:  []string{"http://localhost:5173"},
	}
	s := NewServer(cfg, nil, nil, nil)
	do := func(endpoint, body, auth, origin, fetchSite string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if fetchSite != "" {
			req.Header.Set("Sec-Fetch-Site", fetchSite)
		}
		res := httptest.NewRecorder()
		s.Router().ServeHTTP(res, req)
		return res.Code
	}
	for _, endpoint := range []string{"/api/ops/copy", "/api/ops/move", "/api/ops/delete"} {
		body := `{"source":"/tmp/a","destination":"/tmp/b"}`
		if endpoint == "/api/ops/delete" {
			body = `{"path":"/tmp/a"}`
		}
		if got := do(endpoint, body, "", "", ""); got != http.StatusUnauthorized {
			t.Fatalf("%s without auth: got %d", endpoint, got)
		}
		if got := do(endpoint, body, "invalid", "", ""); got != http.StatusUnauthorized {
			t.Fatalf("%s invalid auth: got %d", endpoint, got)
		}
		if got := do(endpoint, body, token, "https://evil.example", "cross-site"); got != http.StatusForbidden {
			t.Fatalf("%s cross-site: got %d", endpoint, got)
		}
	}
	if got := do("/api/ops/delete", `{"path":`+quoteJSON(external)+`}`, token, "", ""); got != http.StatusBadRequest {
		t.Fatalf("outside delete: got %d", got)
	}
	if got := do("/api/ops/delete", `{"path":`+quoteJSON(allowed)+`}`, token, "", ""); got != http.StatusBadRequest {
		t.Fatalf("allowed root delete: got %d", got)
	}
	if got := do("/api/ops/delete", `{"path":`+quoteJSON(target)+`}`, token, "http://localhost:5173", "same-site"); got != http.StatusOK {
		t.Fatalf("authorized delete: got %d", got)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("authorized delete did not remove target: %v", err)
	}
	if _, err := os.Stat(external); err != nil {
		t.Fatalf("outside file changed: %v", err)
	}
}

func quoteJSON(value string) string {
	return strconv.Quote(value)
}

func TestDisabledOperations(t *testing.T) {
	s := NewServer(config.Config{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/ops/delete", strings.NewReader(`{"path":"/x"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	s.Router().ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected disabled operations, got %d", res.Code)
	}
}

func TestRemoteReadRequiresToken(t *testing.T) {
	root := t.TempDir()
	s := NewServer(config.Config{
		AllowRemoteBind: true,
		OpsAllowedRoots: []string{root},
		OpsAuthToken:    strings.Repeat("x", 32),
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/drives", nil)
	res := httptest.NewRecorder()
	s.Router().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("remote read without auth: got %d", res.Code)
	}
}

func TestSameOriginHTTPSOperation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "safe.txt")
	if err := os.WriteFile(target, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("t", 32)
	s := NewServer(config.Config{
		OpsAuthToken:    token,
		OpsAllowedRoots: []string{root},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "https://storage.example/api/ops/delete", strings.NewReader(`{"path":`+quoteJSON(target)+`}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", "https://storage.example")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	res := httptest.NewRecorder()
	s.Router().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("same-origin HTTPS request rejected: %d %s", res.Code, res.Body.String())
	}
}