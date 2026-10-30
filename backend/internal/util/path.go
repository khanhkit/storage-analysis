package util

import (
	"errors"
	"path/filepath"
	"strings"
)

func NormalizePath(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("path is required")
	}

	abs, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func IsRootPath(input string) bool {
	cleaned := filepath.Clean(input)
	volume := filepath.VolumeName(cleaned)
	if volume == "" {
		return cleaned == string(filepath.Separator)
	}

	remainder := strings.TrimPrefix(cleaned, volume)
	return remainder == "" || remainder == string(filepath.Separator)
}
