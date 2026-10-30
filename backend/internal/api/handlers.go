package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"storageanalysis/internal/config"
	"storageanalysis/internal/duplicates"
	"storageanalysis/internal/ops"
	"storageanalysis/internal/scanner"
	"storageanalysis/internal/security"
	"storageanalysis/internal/system"
	"storageanalysis/internal/util"
)

type Server struct {
	cfg        config.Config
	paths      *security.Paths
	pathErr    error
	scanner    *scanner.Scanner
	dupScanner *duplicates.Scanner
	ops        *ops.Robocopy
}

func NewServer(cfg config.Config, scanner *scanner.Scanner, dupScanner *duplicates.Scanner, opsClient *ops.Robocopy) *Server {
	paths, pathErr := security.NewPaths(cfg.OpsAllowedRoots)
	return &Server{
		cfg:        cfg,
		paths:      paths,
		pathErr:    pathErr,
		scanner:    scanner,
		dupScanner: dupScanner,
		ops:        opsClient,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/drives", s.handleDrives)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/children", s.handleChildren)
	mux.HandleFunc("/api/duplicates", s.handleDuplicates)
	mux.HandleFunc("/api/ops/copy", s.handleCopy)
	mux.HandleFunc("/api/ops/move", s.handleMove)
	mux.HandleFunc("/api/ops/delete", s.handleDelete)
	mux.HandleFunc("/", s.handleStatic)

	return s.withSecurity(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"status": "ok"},
	})
}

func (s *Server) handleDrives(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	drives, err := util.ListDrives()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[DriveData]{
		Success: true,
		Data:    DriveData{Drives: drives},
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	runtimeStats := system.CollectRuntimeStats()
	scanStats := s.scanner.Stats()
	dupStats := s.dupScanner.Stats()

	elapsedMs := int64(0)
	if scanStats.Scanning && !scanStats.StartedAt.IsZero() {
		elapsedMs = time.Since(scanStats.StartedAt).Milliseconds()
	}

	dupElapsedMs := int64(0)
	if dupStats.Scanning && !dupStats.StartedAt.IsZero() {
		dupElapsedMs = time.Since(dupStats.StartedAt).Milliseconds()
	}

	data := StatsData{
		CPUPercent:            runtimeStats.CPUPercent,
		ProcessCPUPercent:     runtimeStats.ProcessCPUPercent,
		Cores:                 runtimeStats.Cores,
		Threads:               runtimeStats.Threads,
		Goroutines:            runtimeStats.Goroutines,
		FilesScanned:          scanStats.FilesScanned,
		DirsScanned:           scanStats.DirsScanned,
		Scanning:              scanStats.Scanning,
		StartedAt:             scanStats.StartedAt,
		ElapsedMs:             elapsedMs,
		DuplicateFilesScanned: dupStats.FilesScanned,
		DuplicateDirsScanned:  dupStats.DirsScanned,
		DuplicateHashed:       dupStats.HashedFiles,
		DuplicateScanning:     dupStats.Scanning,
		DuplicateStartedAt:    dupStats.StartedAt,
		DuplicateElapsedMs:    dupElapsedMs,
	}

	writeJSON(w, http.StatusOK, APIResponse[StatsData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path, err := util.NormalizePath(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	depth := parseInt(r.URL.Query().Get("depth"), s.cfg.DefaultDepth)
	useCache := parseBool(r.URL.Query().Get("useCache"), true)

	result, err := s.scanner.Scan(r.Context(), path, scanner.ScanOptions{
		MaxDepth:      depth,
		UseCache:      useCache,
		CacheStrategy: s.cfg.CacheStrategy,
		ResetStats:    true,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := ScanData{
		Node:      result.Node,
		FromCache: result.FromCache,
		ModTime:   result.ModTime,
		ScannedAt: result.ScannedAt,
		Warnings:  result.Warnings,
	}

	writeJSON(w, http.StatusOK, APIResponse[ScanData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleChildren(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path, err := util.NormalizePath(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	depth := parseInt(r.URL.Query().Get("depth"), 1)
	useCache := parseBool(r.URL.Query().Get("useCache"), true)

	result, err := s.scanner.Scan(r.Context(), path, scanner.ScanOptions{
		MaxDepth:      depth,
		UseCache:      useCache,
		CacheStrategy: s.cfg.CacheStrategy,
		ResetStats:    false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := ScanData{
		Node:      result.Node,
		FromCache: result.FromCache,
		ModTime:   result.ModTime,
		ScannedAt: result.ScannedAt,
		Warnings:  result.Warnings,
	}

	writeJSON(w, http.StatusOK, APIResponse[ScanData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path, err := util.NormalizePath(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	minSize := parseInt64(r.URL.Query().Get("minSizeBytes"), 0)

	result, err := s.dupScanner.Scan(r.Context(), path, minSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	groups := make([]DuplicateGroup, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, DuplicateGroup{
			Hash:      group.Hash,
			SizeBytes: group.SizeBytes,
			Paths:     group.Paths,
		})
	}

	data := DuplicateData{
		Groups:         groups,
		FileCount:      result.FileCount,
		DuplicateFiles: result.DuplicateFiles,
		ScannedAt:      result.ScannedAt,
		Warnings:       result.Warnings,
	}

	writeJSON(w, http.StatusOK, APIResponse[DuplicateData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleCopy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req OpRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	source, isDir, err := s.paths.CheckSource(req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	destination, err := s.paths.CheckDestination(req.Destination)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if isDir && security.IsWithin(source, destination) {
		writeError(w, http.StatusBadRequest, "destination cannot be inside source")
		return
	}
	result, err := s.ops.Copy(r.Context(), source, destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := OperationData{
		Mode:        "copy",
		Source:      req.Source,
		Destination: req.Destination,
		ExitCode:    result.ExitCode,
		DurationMs:  result.Duration.Milliseconds(),
		Output:      result.Output,
	}

	writeJSON(w, http.StatusOK, APIResponse[OperationData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req OpRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	source, isDir, err := s.paths.CheckSource(req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	destination, err := s.paths.CheckDestination(req.Destination)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if isDir && security.IsWithin(source, destination) {
		writeError(w, http.StatusBadRequest, "destination cannot be inside source")
		return
	}
	result, err := s.ops.Move(r.Context(), source, destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := OperationData{
		Mode:        "move",
		Source:      req.Source,
		Destination: req.Destination,
		ExitCode:    result.ExitCode,
		DurationMs:  result.Duration.Milliseconds(),
		Output:      result.Output,
	}

	writeJSON(w, http.StatusOK, APIResponse[OperationData]{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req DeleteRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.paths.Delete(req.Path); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"status": "deleted"},
	})
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	publicRoot := filepath.Clean(s.cfg.PublicDir)
	requestPath := filepath.Clean(filepath.Join(publicRoot, filepath.FromSlash(r.URL.Path)))
	publicRootWithSep := publicRoot + string(os.PathSeparator)
	if requestPath != publicRoot && !strings.HasPrefix(requestPath, publicRootWithSep) {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	fileInfo, err := os.Stat(requestPath)
	if err == nil && !fileInfo.IsDir() {
		setStaticCacheHeaders(w, requestPath)
		http.ServeFile(w, r, requestPath)
		return
	}

	indexPath := filepath.Join(publicRoot, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		setStaticCacheHeaders(w, indexPath)
		http.ServeFile(w, r, indexPath)
		return
	}

	writeError(w, http.StatusNotFound, "asset not found")
}

// withSecurity authenticates operations and, in remote mode, the entire data API.
// No cookie or ambient browser credential is accepted.
func (s *Server) withSecurity(next http.Handler) http.Handler {
	csrf := http.NewCrossOriginProtection()
	for _, origin := range s.cfg.AllowedOrigins {
		_ = csrf.AddTrustedOrigin(origin)
	}
	guarded := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isOp := strings.HasPrefix(r.URL.Path, "/api/ops/")
		protected := isOp || (s.cfg.AllowRemoteBind && strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health")
		if protected {
			if s.cfg.OpsAuthToken == "" || s.pathErr != nil || !s.paths.Enabled() {
				writeError(w, http.StatusServiceUnavailable, "file operations are disabled; configure OPS_AUTH_TOKEN and OPS_ALLOWED_ROOTS")
				return
			}
			constPrefix := "Bearer "
			value := r.Header.Get("Authorization")
			if !strings.HasPrefix(value, constPrefix) ||
				subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(value, constPrefix)), []byte(s.cfg.OpsAuthToken)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="storage-analysis"`)
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
	return s.withCors(csrf.Handler(guarded))
}

func (s *Server) withCors(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(s.cfg.AllowedOrigins))
	for _, origin := range s.cfg.AllowedOrigins {
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				writeError(w, http.StatusForbidden, "untrusted origin")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func readJSON(r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	deco := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
	deco.DisallowUnknownFields()
	if err := deco.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := deco.Decode(&trailing); err != io.EOF {
		return errors.New("unexpected trailing json")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIResponse[struct{}]{
		Success: false,
		Error:   message,
	})
}

func parseInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBool(value string, fallback bool) bool {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return fallback
	}
}

func parseInt64(value string, fallback int64) int64 {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func setStaticCacheHeaders(w http.ResponseWriter, filePath string) {
	if strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".css") || strings.HasSuffix(filePath, ".svg") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}

	if strings.HasSuffix(filePath, ".html") {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
}