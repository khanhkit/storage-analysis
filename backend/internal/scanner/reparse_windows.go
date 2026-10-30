//go:build windows

package scanner

import (
	"os"
	"syscall"
)

func isReparsePoint(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil {
		return false
	}
	return data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
