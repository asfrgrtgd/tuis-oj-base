package core

import "time"

// Queue/Redis キーと可視タイムアウトのデフォルト値をまとめた定数。
const (
	PendingQueueKey    = "pending_submissions"
	ProcessingQueueKey = "processing_submissions"
	// DefaultVisibilityTimeout はワーカーがジョブを保持する可視タイムアウト。
	DefaultVisibilityTimeout = 30 * time.Second
)
