package ops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Threads int
	Timeout time.Duration
}

type Result struct {
	ExitCode int
	Output   string
	Duration time.Duration
}

type Robocopy struct {
	threads int
	timeout time.Duration
}

func NewRobocopy(opts Options) *Robocopy {
	threads := opts.Threads
	if threads <= 0 {
		threads = 16
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	return &Robocopy{
		threads: threads,
		timeout: timeout,
	}
}

func (r *Robocopy) Copy(ctx context.Context, source, destination string) (Result, error) {
	return r.run(ctx, "copy", source, destination)
}

func (r *Robocopy) Move(ctx context.Context, source, destination string) (Result, error) {
	return r.run(ctx, "move", source, destination)
}

func (r *Robocopy) run(ctx context.Context, mode, source, destination string) (Result, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(destination) == "" {
		return Result{}, errors.New("source and destination are required")
	}

	info, err := os.Stat(source)
	if err != nil {
		return Result{}, err
	}

	args, err := buildArgs(mode, source, destination, info.IsDir(), r.threads)
	if err != nil {
		return Result{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "robocopy", args...)
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	exitCode := exitCode(err)
	result := Result{
		ExitCode: exitCode,
		Output:   string(output),
		Duration: elapsed,
	}

	if exitCode > 7 {
		return result, fmt.Errorf("robocopy failed with code %d", exitCode)
	}

	return result, nil
}

func buildArgs(mode, source, destination string, isDir bool, threads int) ([]string, error) {
	if threads <= 0 {
		threads = 16
	}

	threadArg := fmt.Sprintf("/MT:%d", threads)
	baseArgs := []string{"/R:1", "/W:1", threadArg, "/NFL", "/NDL", "/NJH", "/NJS", "/NP", "/XJ", "/SL"}

	if isDir {
		destDir := destination
		if filepath.Base(destination) != filepath.Base(source) {
			destDir = filepath.Join(destination, filepath.Base(source))
		}

		args := []string{source, destDir, "/E", "/COPY:DAT"}
		if mode == "move" {
			args = append(args, "/MOVE")
		}
		return append(args, baseArgs...), nil
	}

	sourceDir := filepath.Dir(source)
	fileName := filepath.Base(source)
	args := []string{sourceDir, destination, fileName, "/COPY:DAT"}
	if mode == "move" {
		args = append(args, "/MOV")
	}

	return append(args, baseArgs...), nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 16
}