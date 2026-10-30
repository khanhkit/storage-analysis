package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths restricts destructive operations to explicitly configured directories.
// os.Root is used for deletion so path traversal and symlink changes cannot
// redirect recursive deletion outside the configured root.
type Paths struct {
	roots []string
}

func NewPaths(allowed []string) (*Paths, error) {
	p := &Paths{}
	for _, raw := range allowed {
		if !filepath.IsAbs(raw) {
			return nil, fmt.Errorf("OPS_ALLOWED_ROOTS entry must be absolute: %q", raw)
		}
		clean := filepath.Clean(raw)
		if isFilesystemRoot(clean) {
			return nil, fmt.Errorf("filesystem root cannot be an allowed operation directory: %q", raw)
		}
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed root %q: %w", raw, err)
		}
		if isFilesystemRoot(resolved) {
			return nil, fmt.Errorf("allowed root resolves to a filesystem root: %q", raw)
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("allowed root must be an existing directory: %q", raw)
		}
		// Windows may canonicalize RUNNER~1 into runneradmin in EvalSymlinks.
		// Preserve the configured spelling so incoming Win32 paths still match;
		// the resolved spelling is an equivalent alias of the same directory.
		p.roots = append(p.roots, clean)
		if resolved != clean {
			p.roots = append(p.roots, resolved)
		}
	}
	return p, nil
}

func (p *Paths) Enabled() bool { return p != nil && len(p.roots) > 0 }

// CheckSource ensures a requested existing source is confined to an allowed root.
func (p *Paths) CheckSource(path string) (string, bool, error) {
	root, rel, abs, err := p.open(path)
	if err != nil {
		return "", false, err
	}
	defer root.Close()
	info, err := root.Stat(rel)
	if err != nil {
		return "", false, err
	}
	return abs, info.IsDir(), nil
}

// CheckDestination requires an existing destination directory inside an allowed root.
func (p *Paths) CheckDestination(path string) (string, error) {
	root, rel, abs, err := p.open(path)
	if err != nil {
		return "", err
	}
	defer root.Close()
	info, err := root.Stat(rel)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("destination must be an existing directory")
	}
	return abs, nil
}

// Delete performs the removal relative to an open, confined os.Root.
func (p *Paths) Delete(path string) error {
	root, rel, _, err := p.open(path)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Lstat(rel); err != nil {
		return err
	}
	return root.RemoveAll(rel)
}

func (p *Paths) open(path string) (*os.Root, string, string, error) {
	if !p.Enabled() {
		return nil, "", "", errors.New("file operations are disabled")
	}
	if path == "" || strings.TrimSpace(path) != path || !filepath.IsAbs(path) {
		return nil, "", "", errors.New("an absolute, non-empty path is required")
	}
	// Reject Windows device namespaces and alternate data streams via the
	// root-aware filesystem APIs below. Device paths are not regular paths.
	if strings.HasPrefix(path, `\\.\`) || strings.HasPrefix(path, `\\?\`) {
		return nil, "", "", errors.New("device paths are not allowed")
	}
	abs := filepath.Clean(path)
	if isFilesystemRoot(abs) {
		return nil, "", "", errors.New("filesystem root is forbidden")
	}

	for _, allowed := range p.roots {
		rel, err := filepath.Rel(allowed, abs)
		if err != nil || rel == "." || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			continue
		}
		root, err := os.OpenRoot(allowed)
		if err != nil {
			return nil, "", "", err
		}
		// Refuse symlinks/junctions anywhere in the requested path.
		// This also disallows currently-existing reparse-point redirects used
		// by Robocopy, whose external process cannot hold os.Root handles.
		parts := strings.Split(rel, string(filepath.Separator))
		for i := range parts {
			component := filepath.Join(parts[:i+1]...)
			info, err := root.Lstat(component)
			if err != nil {
				root.Close()
				return nil, "", "", err
			}
			if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 {
				root.Close()
				return nil, "", "", errors.New("symlinks and irregular paths are not allowed")
			}
		}
		return root, rel, abs, nil
	}
	return nil, "", "", errors.New("path is outside OPS_ALLOWED_ROOTS or names a protected root")
}

func isFilesystemRoot(path string) bool {
	volume := filepath.VolumeName(path)
	return path == string(filepath.Separator) || path == volume ||
		path == volume+string(filepath.Separator)
}

// IsWithin reports whether child is the same as, or below, parent.
func IsWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
