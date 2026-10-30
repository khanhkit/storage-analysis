package api

import (
	"time"

	"storageanalysis/internal/model"
)

type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ScanData struct {
	Node      model.Node `json:"node"`
	FromCache bool       `json:"fromCache"`
	ModTime   time.Time  `json:"modTime"`
	ScannedAt time.Time  `json:"scannedAt"`
	Warnings  []string   `json:"warnings,omitempty"`
}

type DriveData struct {
	Drives []string `json:"drives"`
}

type OpRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type DeleteRequest struct {
	Path string `json:"path"`
}

type OperationData struct {
	Mode        string `json:"mode"`
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	ExitCode    int    `json:"exitCode"`
	DurationMs  int64  `json:"durationMs"`
	Output      string `json:"output"`
}

type StatsData struct {
	CPUPercent        float64 `json:"cpuPercent"`
	ProcessCPUPercent float64 `json:"processCpuPercent"`
	Cores             int     `json:"cores"`
	Threads           int32   `json:"threads"`
	Goroutines        int     `json:"goroutines"`
	FilesScanned      int64   `json:"filesScanned"`
	DirsScanned       int64   `json:"dirsScanned"`
	Scanning          bool    `json:"scanning"`
	StartedAt         time.Time `json:"startedAt"`
	ElapsedMs         int64   `json:"elapsedMs"`
	DuplicateFilesScanned int64     `json:"duplicateFilesScanned"`
	DuplicateDirsScanned  int64     `json:"duplicateDirsScanned"`
	DuplicateHashed       int64     `json:"duplicateHashed"`
	DuplicateScanning     bool      `json:"duplicateScanning"`
	DuplicateStartedAt    time.Time `json:"duplicateStartedAt"`
	DuplicateElapsedMs    int64     `json:"duplicateElapsedMs"`
}

type DuplicateGroup struct {
	Hash      string   `json:"hash"`
	SizeBytes int64    `json:"sizeBytes"`
	Paths     []string `json:"paths"`
}

type DuplicateData struct {
	Groups         []DuplicateGroup `json:"groups"`
	FileCount      int              `json:"fileCount"`
	DuplicateFiles int              `json:"duplicateFiles"`
	ScannedAt      time.Time        `json:"scannedAt"`
	Warnings       []string         `json:"warnings,omitempty"`
}
