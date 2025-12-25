import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/lib/api'
import { BackLink, CopyButton, VerdictBadge } from '@/components/common'
import { formatDateWithSeconds } from '@/lib/utils'
import type { JudgeDetail } from '@/types'
import { RefreshCw, Search } from 'lucide-react'

function TestCaseResult({ detail }: { detail: JudgeDetail }) {
  const showTime = detail.status !== 'TLE' && detail.time_ms !== undefined
  const showMem = detail.status !== 'MLE' && detail.memory_kb !== undefined

  return (
    <div className="grid grid-cols-[1fr_60px_70px_80px] sm:grid-cols-[1fr_60px_70px_80px] items-center gap-2 px-3 py-2 border-b border-border last:border-b-0 text-sm">
      <span className="mono truncate pl-1">{detail.testcase}</span>
      <span className="text-center">
        <VerdictBadge verdict={detail.status} />
      </span>
      <span className="mono text-right">
        {showTime ? `${detail.time_ms} ms` : '-'}
      </span>
      <span className="mono text-right hidden sm:block">
        {showMem ? `${detail.memory_kb} KB` : '-'}
      </span>
    </div>
  )
}

export function SubmissionDetailPage() {
  const params = useParams()
  const submissionId = Number(params.id)

  const { data: submission, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['submission', submissionId],
    queryFn: () => api.submissions.detail(submissionId),
    enabled: Number.isFinite(submissionId),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'pending' || status === 'running' ? 1500 : false
    },
  })

  if (isLoading) {
    return (
      <div className="py-8">
        <div className="skeleton h-8 w-48 mb-4" />
        <div className="skeleton h-64 w-full" />
      </div>
    )
  }

  if (!submission) {
    return (
      <div className="py-8">
        <div className="card">
          <div className="empty-state">
            <Search size={48} className="text-muted opacity-50 mb-4" />
            <h2 className="empty-state-title">提出が見つかりません</h2>
            <p className="empty-state-description">
              指定された提出は存在しないか、アクセス権限がありません
            </p>
            <Link to="/problems" className="btn btn-primary mt-4">
              問題一覧に戻る
            </Link>
          </div>
        </div>
      </div>
    )
  }

  const isPending = submission.status === 'pending' || submission.status === 'running'

  return (
    <div className="py-8 max-w-5xl mx-auto space-y-6">
      {/* ヘッダー */}
      <div className="mb-6">
        <div className="mb-4">
          <BackLink to={`/problems/${submission.problem_id}`}>問題に戻る</BackLink>
        </div>
        
        <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
          <div>
            <h1 className="page-title mb-2">提出 #{submission.id}</h1>
            <Link
              to={`/problems/${submission.problem_id}`}
              className="link text-lg font-medium"
            >
              {submission.problem_title || `問題 ${submission.problem_id}`}
            </Link>
          </div>
          <div className="flex items-center gap-3">
            {isPending && (
              <span className="text-sm text-muted flex items-center gap-1">
                <span className="loading-spinner" />
                ジャッジ中...
              </span>
            )}
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="btn btn-secondary btn-sm"
            >
              <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
              更新
            </button>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-6">
        {/* 結果詳細 */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold">ジャッジ結果</h2>
          </div>
          <div className="card-body p-0">
            <table className="w-full text-sm">
              <tbody>
                <tr className="border-b border-border">
                  <th className="px-4 py-2 text-left text-muted font-medium bg-secondary w-28">結果</th>
                  <td className="px-4 py-2">
                    <VerdictBadge verdict={submission.verdict} status={submission.status} />
                  </td>
                </tr>
                <tr className="border-b border-border">
          <th className="px-4 py-2 text-left text-muted font-medium bg-secondary">実行時間</th>
          <td className="px-4 py-2 mono">
            {submission.verdict === 'AC' && submission.time_ms !== undefined
              ? `${submission.time_ms} ms`
              : '-'}
          </td>
        </tr>
        <tr className="border-b border-border">
          <th className="px-4 py-2 text-left text-muted font-medium bg-secondary">メモリ</th>
          <td className="px-4 py-2 mono">
            {submission.verdict === 'AC' && submission.memory_kb !== undefined
              ? `${submission.memory_kb} KB`
              : '-'}
          </td>
        </tr>
                <tr className="border-b border-border">
                  <th className="px-4 py-2 text-left text-muted font-medium bg-secondary">ユーザー</th>
                  <td className="px-4 py-2">{submission.userid}</td>
                </tr>
                <tr className="border-b border-border">
                  <th className="px-4 py-2 text-left text-muted font-medium bg-secondary">言語</th>
                  <td className="px-4 py-2">{submission.language}</td>
                </tr>
                <tr>
                  <th className="px-4 py-2 text-left text-muted font-medium bg-secondary">提出日時</th>
                  <td className="px-4 py-2">{formatDateWithSeconds(submission.created_at)}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* テストケース結果 */}
        {submission.judge_details && submission.judge_details.length > 0 && (
          <div className="card">
            <div className="card-header flex items-center justify-between">
              <h2 className="font-semibold">テストケース結果</h2>
              <span className="text-sm text-muted">
                {submission.judge_details.filter(d => d.status === 'AC').length} / {submission.judge_details.length} 通過
              </span>
            </div>
            <div className="divide-y divide-border">
              {/* ヘッダー */}
              <div className="grid grid-cols-[1fr_60px_70px_80px] sm:grid-cols-[1fr_60px_70px_80px] items-center gap-2 px-3 py-2 bg-secondary text-sm font-medium text-muted">
                <span className="pl-1">ケース名</span>
                <span className="text-center">結果</span>
                <span className="text-right">時間</span>
                <span className="text-right hidden sm:block">メモリ</span>
              </div>
              {/* ケース一覧 */}
              {submission.judge_details.map((detail, idx) => (
                <TestCaseResult key={idx} detail={detail} />
              ))}
            </div>
          </div>
        )}

        {/* ソースコード */}
        <div className="card">
          <div className="card-header flex items-center justify-between">
            <h2 className="font-semibold">ソースコード</h2>
            {submission.source_code && (
              <CopyButton text={submission.source_code} showLabel />
            )}
          </div>
          <div className="card-body p-0">
            <pre className="code text-sm overflow-auto max-h-[600px] p-4 rounded-none border-0">
              {submission.source_code || '（ソースコードを表示できません）'}
            </pre>
          </div>
        </div>

        {/* エラーメッセージ */}
        {submission.error_message && (
          <div className="card">
            <div className="card-header">
              <h2 className="font-semibold text-destructive">エラーメッセージ</h2>
            </div>
            <div className="card-body">
              <pre className="code text-sm text-destructive whitespace-pre-wrap">
                {submission.error_message}
              </pre>
            </div>
          </div>
        )}

        {/* ジャッジ中の場合 */}
        {isPending && (!submission.judge_details || submission.judge_details.length === 0) && (
          <div className="card">
            <div className="card-body">
              <div className="empty-state empty-state-sm">
                <span className="loading-spinner mb-4" style={{ width: 32, height: 32 }} />
                <h2 className="empty-state-title">ジャッジ中...</h2>
                <p className="empty-state-description">
                  結果が出るまでお待ちください。自動的に更新されます。
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
