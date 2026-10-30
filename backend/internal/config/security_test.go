package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRejectsUnsafeNetworkBind(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "allowed")
	if err := os.Mkdir(allowed, 0o700); err != nil { t.Fatal(err) }
	tests := []struct {
		name string
		cfg Config
		valid bool
	}{
		{"loopback read only", Config{Address: "127.0.0.1:8080"}, true},
		{"ipv6 loopback read only", Config{Address: "[::1]:8080"}, true},
		{"wildcard without opt in", Config{Address: ":8080"}, false},
		{"network without opt in", Config{Address: "0.0.0.0:8080"}, false},
		{"network opted in without TLS", Config{Address: "0.0.0.0:8080", AllowRemoteBind: true, OpsAuthToken: strings.Repeat("x", 32), OpsAllowedRoots: []string{allowed}}, false},
		{"network opted in without token", Config{Address: "0.0.0.0:8080", AllowRemoteBind: true, TLSCertFile: "cert.pem", TLSKeyFile: "key.pem", OpsAllowedRoots: []string{allowed}}, false},
		{"network opted in with security", Config{Address: "0.0.0.0:8080", AllowRemoteBind: true, TLSCertFile: "cert.pem", TLSKeyFile: "key.pem", OpsAuthToken: strings.Repeat("x", 32), OpsAllowedRoots: []string{allowed}}, true},
		{"weak token", Config{Address: "127.0.0.1:8080", OpsAuthToken: "1234"}, false},
		{"unsafe allowed root", Config{Address: "127.0.0.1:8080", OpsAllowedRoots: []string{string(filepath.Separator)}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err == nil) != tc.valid {
				t.Fatalf("Validate error = %v; expected valid = %t", err, tc.valid)
			}
		})
	}
}

func TestLoadSecurityConfiguration(t *testing.T) {
	t.Setenv("OPS_AUTH_TOKEN", strings.Repeat("y", 32))
	t.Setenv("OPS_ALLOWED_ROOTS", " /root/a, /root/b ")
	t.Setenv("ALLOW_REMOTE_BIND", "true")
	cfg := Load()
	if !cfg.AllowRemoteBind || len(cfg.OpsAllowedRoots) != 2 || cfg.OpsAuthToken != strings.Repeat("y", 32) {
		t.Fatalf("invalid security config: remote=%v roots=%d tokenLength=%d", cfg.AllowRemoteBind, len(cfg.OpsAllowedRoots), len(cfg.OpsAuthToken))
	}
}
