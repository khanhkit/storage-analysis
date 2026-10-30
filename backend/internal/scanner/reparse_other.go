//go:build !windows

package scanner

import "os"

func isReparsePoint(info os.FileInfo) bool {
	return false
}
