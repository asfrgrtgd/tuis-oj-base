package core

import (
	"context"
	"encoding/json"
	"runtime"
	"time"
)

const (
	WorkerHeartbeatPrefix = "worker:heartbeat:"
	WorkerHeartbeatTTL    = 45 * time.Second
)

// WorkerHeartbeatKey returns Redis key for given worker ID.
func WorkerHeartbeatKey(id string) string {
	return WorkerHeartbeatPrefix + id
}

// SaveHeartbeat stores heartbeat JSON with TTL.
func SaveHeartbeat(ctx context.Context, client RedisClientRaw, hb WorkerHeartbeat) error {
	hb.UpdatedAt = time.Now()
	data, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	return client.Set(ctx, WorkerHeartbeatKey(hb.WorkerID), data, WorkerHeartbeatTTL).Err()
}

// WorkerHeartbeat はワーカーが Redis に定期送信する稼働情報。
// JSON で保存し API から参照する。
type WorkerHeartbeat struct {
	WorkerID       string    `json:"worker_id"`
	Hostname       string    `json:"hostname"`
	PID            int       `json:"pid"`
	Version        string    `json:"version"` // 予備: ビルドバージョンやGit SHA
	Concurrency    int       `json:"concurrency"`
	UptimeSeconds  int64     `json:"uptime_seconds"`
	Status         string    `json:"status"` // idle|busy|starting
	RunningCount   int       `json:"running_count"`
	CurrentJob     string    `json:"current_job,omitempty"`
	RunningJobs    []string  `json:"running_jobs,omitempty"`
	ProcessedTotal int64     `json:"processed_total"`
	FailedTotal    int64     `json:"failed_total"`
	LastError      string    `json:"last_error,omitempty"`
	MemoryRSSBytes uint64    `json:"memory_rss_bytes"`
	NumGoroutine   int       `json:"num_goroutine"`
	StartedAt      time.Time `json:"started_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UpdateRuntimeStats はメモリ/Goroutine を現在値で上書きするヘルパー。
func (h *WorkerHeartbeat) UpdateRuntimeStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	h.MemoryRSSBytes = ms.Sys // 近似。必要なら procfs で置換。
	h.NumGoroutine = runtime.NumGoroutine()
}
