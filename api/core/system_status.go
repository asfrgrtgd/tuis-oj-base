package core

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// SystemStatus は管理ダッシュボード向けの集約ステータス。
type SystemStatus struct {
	Queue struct {
		Pending    int64 `json:"pending"`
		Processing int64 `json:"processing"`
	} `json:"queue"`
	Workers struct {
		Active int `json:"active"`
		Total  int `json:"total"`
	} `json:"workers"`
	Memory struct {
		UsedBytes  uint64 `json:"used_bytes"`
		TotalBytes uint64 `json:"total_bytes"`
	} `json:"memory"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}

// CollectSystemStatus で現在のステータスを集約する。
func CollectSystemStatus(ctx context.Context, metrics *MetricsService, startedAt time.Time) (SystemStatus, error) {
	var st SystemStatus

	// Queue
	if metrics != nil {
		if qm, err := metrics.Queue(ctx); err == nil {
			st.Queue.Pending = qm.Pending
			st.Queue.Processing = qm.Processing
		}
		workers, _ := metrics.Workers(ctx) // ignore error to keep best-effort
		st.Workers.Total = len(workers)
		active := 0
		for _, w := range workers {
			if w.Status != "starting" {
				active++
			}
		}
		st.Workers.Active = active
	}

	// Memory (best-effort from /proc/meminfo)
	used, total := readMemInfo()
	st.Memory.UsedBytes = used
	st.Memory.TotalBytes = total

	// Uptime
	if !startedAt.IsZero() {
		st.UptimeSeconds = int64(time.Since(startedAt).Seconds())
	}

	return st, nil
}

// readMemInfo returns used and total bytes using /proc/meminfo.
// If unavailable, returns zeros.
func readMemInfo() (used, total uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseKiBLine(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseKiBLine(line)
		}
	}
	if memTotal > 0 {
		total = memTotal
		if memAvailable <= memTotal {
			used = memTotal - memAvailable
		}
		// convert KiB -> bytes
		used *= 1024
		total *= 1024
	}
	return used, total
}

func parseKiBLine(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v
}
