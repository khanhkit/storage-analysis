//go:build !windows

package duplicates

import "os"

func isReparsePoint(info os.FileInfo) bool {
	return false
}
