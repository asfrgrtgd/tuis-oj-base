import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Alert } from '@/components/ui/Alert'
import { BackLink } from '@/components/common'
import { RefreshCw, Server, Database, Activity, Clock, HardDrive } from 'lucide-react'

interface SystemStatus {
  queue: {
    pending: number
    processing: number
  }
  workers?: {
    active: number
    total: number
  }
  memory?: {
    used_bytes: number
    total_bytes: number
  }
  uptime_seconds?: number
}

interface WorkerInfo {
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
  memory_rss_bytes?: number
  num_goroutine?: number
  started_at: string
  updated_at: string
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  const parts = []
  if (days > 0) parts.push(`${days}日`)
  if (hours > 0) parts.push(`${hours}時間`)
  if (minutes > 0) parts.push(`${minutes}分`)
  
  return parts.join(' ') || '0分'
}

function StatCard({ 
  icon: Icon, 
  title, 
  value, 
  subValue,
  color = 'primary'
}: { 
  icon: React.ComponentType<{ size?: number | string; className?: string }>
  title: string
  value: string | number
  subValue?: string
  color?: 'primary' | 'success' | 'warning' | 'danger'
}) {
  const colorClasses = {
    primary: 'text-primary bg-primary/10',
    success: 'text-success bg-success/10',
    warning: 'text-warning bg-warning/10',
    danger: 'text-destructive bg-destructive/10',
  }

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-center gap-3 mb-3">
          <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
            <Icon size={20} />
          </div>
          <span className="text-sm text-muted">{title}</span>
        </div>
        <p className="text-2xl font-bold">{value}</p>
        {subValue && <p className="text-sm text-muted mt-1">{subValue}</p>}
      </div>
    </div>
  )
}

function ProgressBar({ value, max }: { value: number; max: number }) {
  const percentage = max > 0 ? (value / max) * 100 : 0
  
  return (
    <div className="h-2 bg-secondary rounded-full overflow-hidden">
      <div 
        className="h-full bg-primary transition-all"
        style={{ width: `${Math.min(percentage, 100)}%` }}
      />
    </div>
  )
}

export function AdminSystem() {
  // システム状態を取得
  const systemQuery = useQuery({
    queryKey: ['admin-system-status'],
    queryFn: async (): Promise<SystemStatus> => {
      const data = await api.admin.systemStatus()
      return data
    },
    refetchInterval: 5000, // 5秒ごとに自動更新
  })

  // メトリクス詳細を取得（ワーカー情報など）
  const metricsQuery = useQuery({
    queryKey: ['admin-metrics-overview'],
    queryFn: async () => {
      const data = await api.admin.metricsOverview()
      return data
    },
    refetchInterval: 5000,
  })

  const status = systemQuery.data
  const metrics = metricsQuery.data
  const workers: WorkerInfo[] = metrics?.workers ?? []
  const isLoading = systemQuery.isLoading || metricsQuery.isLoading
  const isFetching = systemQuery.isFetching || metricsQuery.isFetching
  const hasError = systemQuery.isError || metricsQuery.isError

  const handleRefresh = () => {
    systemQuery.refetch()
    metricsQuery.refetch()
  }

  return (
    <div className="py-8">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>

      <div className="flex items-center justify-between mb-6">
        <h1 className="page-title mb-0">システム状態</h1>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className="btn btn-secondary btn-sm"
        >
          <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
          更新
        </button>
      </div>

      {hasError && (
        <Alert variant="error" className="mb-6">
          システム情報の取得に失敗しました。APIが実装されているか確認してください。
        </Alert>
      )}

      {/* ステータスカード */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <StatCard
          icon={Database}
          title="待機中のジョブ"
          value={isLoading ? '-' : (status?.queue?.pending ?? metrics?.queues?.pending ?? 0)}
          color={status && (status.queue?.pending ?? 0) > 10 ? 'warning' : 'primary'}
        />
        <StatCard
          icon={Activity}
          title="処理中のジョブ"
          value={isLoading ? '-' : (status?.queue?.processing ?? metrics?.queues?.processing ?? 0)}
          color="success"
        />
        <StatCard
          icon={Server}
          title="ワーカー"
          value={isLoading ? '-' : (status?.workers ? `${status.workers.active}/${status.workers.total}` : `${workers.filter(w => w.status === 'busy' || w.status === 'idle').length}/${workers.length}`)}
          subValue="アクティブ/合計"
        />
        <StatCard
          icon={Clock}
          title="稼働時間"
          value={isLoading ? '-' : (status?.uptime_seconds ? formatUptime(status.uptime_seconds) : '-')}
        />
      </div>

      {/* 詳細情報 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* キュー状態 */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold flex items-center gap-2">
              <Database size={16} />
              キュー状態
            </h2>
          </div>
          <div className="card-body space-y-4">
            <div>
              <div className="flex justify-between text-sm mb-1">
                <span>待機中</span>
                <span className="font-medium">{status?.queue?.pending ?? metrics?.queues?.pending ?? 0}</span>
              </div>
              <ProgressBar value={status?.queue?.pending ?? metrics?.queues?.pending ?? 0} max={20} />
            </div>
            <div>
              <div className="flex justify-between text-sm mb-1">
                <span>処理中</span>
                <span className="font-medium">{status?.queue?.processing ?? metrics?.queues?.processing ?? 0}</span>
              </div>
              <ProgressBar value={status?.queue?.processing ?? metrics?.queues?.processing ?? 0} max={10} />
            </div>
            {metrics?.queues?.expired_candidate !== undefined && metrics.queues.expired_candidate > 0 && (
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-warning">期限切れ候補</span>
                  <span className="font-medium text-warning">{metrics.queues.expired_candidate}</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* メモリ使用状況 */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold flex items-center gap-2">
              <HardDrive size={16} />
              メモリ使用状況
            </h2>
          </div>
          <div className="card-body">
            {status?.memory ? (
              <div>
                <div className="flex justify-between text-sm mb-2">
                  <span>使用中</span>
                  <span className="font-medium">
                    {formatBytes(status.memory.used_bytes)} / {formatBytes(status.memory.total_bytes)}
                  </span>
                </div>
                <ProgressBar 
                  value={status.memory.used_bytes} 
                  max={status.memory.total_bytes}
                />
                <p className="text-sm text-muted mt-2">
                  使用率: {((status.memory.used_bytes / status.memory.total_bytes) * 100).toFixed(1)}%
                </p>
              </div>
            ) : (
              <p className="text-muted text-sm">
                メモリ情報は利用できません
              </p>
            )}
          </div>
        </div>

      </div>

      {/* ワーカー一覧 */}
      {workers.length > 0 && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Server size={18} />
            ワーカー一覧
          </h2>
          <div className="grid gap-4 lg:grid-cols-2">
            {workers.map((worker) => (
              <WorkerCard key={worker.worker_id} worker={worker} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function WorkerCard({ worker }: { worker: WorkerInfo }) {
  const statusColor = {
    idle: 'bg-success',
    busy: 'bg-warning',
    offline: 'bg-destructive',
  }[worker.status] ?? 'bg-muted'

  const statusLabel = {
    idle: 'アイドル',
    busy: 'ビジー',
    offline: 'オフライン',
  }[worker.status] ?? worker.status

  return (
    <div className="card">
      <div className="card-header flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={`w-2 h-2 rounded-full ${statusColor}`} />
          <span className="font-medium text-sm">{worker.hostname}:{worker.pid}</span>
        </div>
        <span className={`text-xs px-2 py-0.5 rounded ${
          worker.status === 'idle' ? 'bg-success/10 text-success' :
          worker.status === 'busy' ? 'bg-warning/10 text-warning' :
          'bg-destructive/10 text-destructive'
        }`}>
          {statusLabel}
        </span>
      </div>
      <div className="card-body text-sm space-y-2">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <span className="text-muted">並列数:</span>{' '}
            <span className="font-medium">{worker.concurrency}</span>
          </div>
          <div>
            <span className="text-muted">実行中:</span>{' '}
            <span className="font-medium">{worker.running_count}</span>
          </div>
          <div>
            <span className="text-muted">処理済:</span>{' '}
            <span className="font-medium">{worker.processed_total}</span>
          </div>
          <div>
            <span className="text-muted">失敗:</span>{' '}
            <span className={`font-medium ${worker.failed_total > 0 ? 'text-destructive' : ''}`}>
              {worker.failed_total}
            </span>
          </div>
        </div>
        {worker.memory_rss_bytes !== undefined && (
          <div>
            <span className="text-muted">メモリ:</span>{' '}
            <span className="font-medium">{formatBytes(worker.memory_rss_bytes)}</span>
          </div>
        )}
        {worker.running_jobs && worker.running_jobs.length > 0 && (
          <div>
            <span className="text-muted">実行中ジョブ:</span>{' '}
            <span className="font-mono text-xs">{worker.running_jobs.join(', ')}</span>
          </div>
        )}
        {worker.last_error && (
          <div className="text-destructive text-xs">
            <span className="text-muted">最終エラー:</span> {worker.last_error}
          </div>
        )}
        <div className="text-xs text-muted">
          稼働時間: {formatUptime(worker.uptime_seconds)}
        </div>
      </div>
    </div>
  )
}
