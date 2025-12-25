package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// QueueMetrics はキューの現在値を表す。
type QueueMetrics struct {
	Pending          int64 `json:"pending"`
	Processing       int64 `json:"processing"`
	ExpiredCandidate int64 `json:"expired_candidate"`
}

// MetricsService は Redis からキュー長とワーカーハートビートを取得する。
type MetricsService struct {
	redis RedisClientRaw
}

func NewMetricsService(redis RedisClientRaw) *MetricsService {
	return &MetricsService{redis: redis}
}

// Overview はキューと全ワーカーの簡易情報を返す。
func (s *MetricsService) Overview(ctx context.Context) (QueueMetrics, []WorkerHeartbeat, error) {
	queue, err := s.Queue(ctx)
	if err != nil {
		return QueueMetrics{}, nil, err
	}
	workers, err := s.Workers(ctx)
	if err != nil {
		return queue, nil, err
	}
	return queue, workers, nil
}

// Queue は pending / processing の件数と期限切れ候補数を返す。
func (s *MetricsService) Queue(ctx context.Context) (QueueMetrics, error) {
	now := time.Now().UnixMilli()
	pending, err := s.redis.LLen(ctx, PendingQueueKey).Result()
	if err != nil {
		return QueueMetrics{}, err
	}
	processing, err := s.redis.ZCard(ctx, ProcessingQueueKey).Result()
	if err != nil {
		return QueueMetrics{}, err
	}
	expired, err := s.redis.ZCount(ctx, ProcessingQueueKey, "-inf", fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return QueueMetrics{}, err
	}
	return QueueMetrics{Pending: pending, Processing: processing, ExpiredCandidate: expired}, nil
}

// Workers は Redis に残っているハートビートをすべて返す。
func (s *MetricsService) Workers(ctx context.Context) ([]WorkerHeartbeat, error) {
	iter := s.redis.Scan(ctx, 0, WorkerHeartbeatPrefix+"*", 100).Iterator()
	var res []WorkerHeartbeat
	for iter.Next(ctx) {
		key := iter.Val()
		val, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var hb WorkerHeartbeat
		if err := json.Unmarshal([]byte(val), &hb); err != nil {
			continue
		}
		res = append(res, hb)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// WorkerByID は特定ワーカーのハートビートを返す。
func (s *MetricsService) WorkerByID(ctx context.Context, id string) (*WorkerHeartbeat, error) {
	val, err := s.redis.Get(ctx, WorkerHeartbeatKey(id)).Result()
	if err != nil {
		return nil, err
	}
	var hb WorkerHeartbeat
	if err := json.Unmarshal([]byte(val), &hb); err != nil {
		return nil, err
	}
	return &hb, nil
}
