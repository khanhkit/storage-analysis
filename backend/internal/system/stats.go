package system

import (
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
)

type RuntimeStats struct {
	CPUPercent       float64
	ProcessCPUPercent float64
	Cores            int
	Threads          int32
	Goroutines       int
}

func CollectRuntimeStats() RuntimeStats {
	stats := RuntimeStats{
		Cores:      runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
	}

	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		stats.CPUPercent = values[0]
	}

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err == nil {
		if threads, err := proc.NumThreads(); err == nil {
			stats.Threads = threads
		}
		if percent, err := proc.CPUPercent(); err == nil {
			stats.ProcessCPUPercent = percent
		}
	}

	return stats
}
