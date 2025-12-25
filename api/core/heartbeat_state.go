package core

import (
	"context"
	"os"
	"sync"
	"time"
)

// HeartbeatState は単一 worker プロセスの集約メトリクスを保持する。
type HeartbeatState struct {
	mu       sync.Mutex
	hb       WorkerHeartbeat
	running  map[string]time.Time
	ticker   *time.Ticker
	stopOnce sync.Once
}

func NewHeartbeatState(workerID, hostname string, concurrency int) *HeartbeatState {
	return &HeartbeatState{
		hb: WorkerHeartbeat{
			WorkerID:     workerID,
			Hostname:     hostname,
			PID:          os.Getpid(),
			Concurrency:  concurrency,
			Status:       "starting",
			RunningCount: 0,
			StartedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			RunningJobs:  []string{},
		},
		running: make(map[string]time.Time),
		ticker:  time.NewTicker(5 * time.Second),
	}
}

// Start を呼ぶとバックグラウンドで TTL 更新を行う。
func (s *HeartbeatState) Start(ctx context.Context, client RedisClientRaw) {
	// 直ちに 1 回送信
	s.flush(ctx, client)
	defer s.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			s.flush(ctx, client)
		}
	}
}

// JobStarted は実行中ジョブを追加し、状態を busy にする。
func (s *HeartbeatState) JobStarted(job string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hb.Status = "busy"
	s.running[job] = time.Now()
	s.updateRunningFieldsLocked()
}

// JobFinished はジョブ終了時のカウンタ更新を行う。
func (s *HeartbeatState) JobFinished(job string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, job)
	s.hb.ProcessedTotal++
	if err != nil {
		s.hb.FailedTotal++
		s.hb.LastError = err.Error()
	}
	if len(s.running) == 0 {
		s.hb.Status = "idle"
	} else {
		s.hb.Status = "busy"
	}
	s.updateRunningFieldsLocked()
}

func (s *HeartbeatState) updateRunningFieldsLocked() {
	s.hb.RunningCount = len(s.running)
	s.hb.RunningJobs = s.hb.RunningJobs[:0]
	for job := range s.running {
		if len(s.hb.RunningJobs) >= 3 {
			break
		}
		s.hb.RunningJobs = append(s.hb.RunningJobs, job)
	}
	if s.hb.RunningCount == 0 {
		s.hb.CurrentJob = ""
	} else {
		s.hb.CurrentJob = s.hb.RunningJobs[0]
	}
}

func (s *HeartbeatState) flush(ctx context.Context, client RedisClientRaw) {
	s.mu.Lock()
	s.hb.UptimeSeconds = int64(time.Since(s.hb.StartedAt).Seconds())
	s.hb.UpdateRuntimeStats()
	hbCopy := s.hb
	s.mu.Unlock()
	_ = SaveHeartbeat(ctx, client, hbCopy)
}
