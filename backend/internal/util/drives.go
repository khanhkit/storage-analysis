package util

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func ListDrives() ([]string, error) {
	if runtime.GOOS != "windows" {
		return []string{"/"}, nil
	}

	output, err := exec.Command("wmic", "logicaldisk", "get", "name").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	drives := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.EqualFold(trimmed, "name") {
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			drives = append(drives, trimmed+"\\")
		}
	}

	if len(drives) == 0 {
		return nil, errors.New("no drives found")
	}

	return drives, nil
}
