//go:build windows

package scanner

import (
	"os"
	"syscall"
	"testing"
	"time"
)

type fakeFileInfo struct {
	name        string
	mode        os.FileMode
	modTime     time.Time
	attributes  uint32
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any {
	return &syscall.Win32FileAttributeData{FileAttributes: f.attributes}
}

func TestIsReparsePoint(t *testing.T) {
	info := fakeFileInfo{attributes: syscall.FILE_ATTRIBUTE_REPARSE_POINT}
	if !isReparsePoint(info) {
		t.Fatalf("expected reparse point to be detected")
	}
}

func TestIsReparsePointFalse(t *testing.T) {
	info := fakeFileInfo{attributes: 0}
	if isReparsePoint(info) {
		t.Fatalf("expected non-reparse point to be false")
	}
}
