package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathsConfineOperations(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "allowed")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{allowed, outside} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(allowed, "item.txt")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := NewPaths([]string{allowed})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{allowed, outside, secret, filepath.Join(allowed, "..", "outside", "secret.txt"), ".", ""} {
		if err := p.Delete(path); err == nil {
			t.Errorf("deletion unexpectedly allowed: %q", path)
		}
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf("outside file changed: %v", err)
	}
	if _, err := p.CheckDestination(outside); err == nil {
		t.Fatal("outside destination accepted")
	}
	if _, err := p.CheckDestination(allowed); err == nil {
		t.Fatal("allowed root destination must not be directly mutable")
	}
	if _, _, err := p.CheckSource(target); err != nil {
		root := p.roots[0]
		relative, relErr := filepath.Rel(root, target)
		t.Fatalf("CheckSource(%q): %v (configured=%q, canonical=%q, relative=%q, relErr=%v)", target, err, allowed, root, relative, relErr)
	}
	if err := p.Delete(target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("file was not removed")
	}
}

func TestSymlinkEscapeBlocked(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "allowed")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(allowed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(allowed, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	p, err := NewPaths([]string{allowed})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Delete(link); err == nil {
		t.Fatal("symlink deletion must be rejected")
	}
	if _, err := p.CheckDestination(link); err == nil {
		t.Fatal("symlink destination must be rejected")
	}
}

func TestRootAndRelativePathsRejected(t *testing.T) {
	if _, err := NewPaths([]string{string(filepath.Separator)}); err == nil {
		t.Fatal("root allowlist accepted")
	}
	if _, err := NewPaths([]string{"relative"}); err == nil {
		t.Fatal("relative allowlist accepted")
	}
	if IsWithin("/safe", "/safe2/item") {
		t.Fatal("sibling path accepted")
	}
}
