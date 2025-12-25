export interface QueueMetrics {
  pending: number
  processing: number
  expired_candidate: number
}

export interface WorkerHeartbeat {
  worker_id: string
  hostname: string
  pid: number
  concurrency: number
  uptime_seconds: number
  status: string
  running_count: number
  current_job?: string
  running_jobs?: string[]
  processed_total: number
  failed_total: number
  last_error?: string
  memory_rss_bytes: number
  num_goroutine: number
  started_at: string
  updated_at: string
}

export interface MetricsOverview {
  queues: QueueMetrics
  workers: WorkerHeartbeat[]
}

