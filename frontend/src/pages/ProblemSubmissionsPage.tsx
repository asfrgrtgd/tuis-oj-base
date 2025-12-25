import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { api } from '@/lib/api'
import { BackLink, VerdictBadge } from '@/components/common'
import { formatRelativeTime } from '@/lib/utils'
import type { Submission } from '@/types'
import { RefreshCw, ChevronLeft, ChevronRight, FileText } from 'lucide-react'

type SubmissionTab = 'mine' | 'all'

export function ProblemSubmissionsPage() {
  const params = useParams()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const problemId = Number(params.id)
  
  // URLのクエリパラメータからタブを取得
  const tabParam = searchParams.get('tab')
  const activeTab: SubmissionTab = tabParam === 'all' ? 'all' : 'mine'
  const [pageMine, setPageMine] = useState(1)
  const [pageAll, setPageAll] = useState(1)
  const page = activeTab === 'all' ? pageAll : pageMine
  const setPageForActive = activeTab === 'all' ? setPageAll : setPageMine

  // 問題情報を取得
  const problemQuery = useQuery({
    queryKey: ['problem', problemId],
    queryFn: () => api.problems.get(problemId),
    enabled: Number.isFinite(problemId),
  })

  // 自分の提出
  const mySubmissionsQuery = useQuery({
    queryKey: ['my-submissions', problemId, page],
    queryFn: () => api.submissions.mine(page, 20, problemId),
    enabled: Number.isFinite(problemId) && activeTab === 'mine',
  })

  // 全員の提出
  const allSubmissionsQuery = useQuery({
    queryKey: ['problem-submissions', problemId, page],
    queryFn: () => api.problems.submissions(problemId, page, 20),
    enabled: Number.isFinite(problemId) && activeTab === 'all',
  })

  const handleRowClick = (submissionId: number) => {
    navigate(`/submissions/${submissionId}`)
  }

  const currentQuery = activeTab === 'mine' ? mySubmissionsQuery : allSubmissionsQuery
  const items = currentQuery.data?.items ?? []
  const totalPages = currentQuery.data?.total_pages ?? 1
  const isShowingAll = activeTab === 'all'


  return (
    <div className="py-8">
      {/* ヘッダー */}
      <div className="mb-6">
        <div className="mb-4">
          <BackLink to={`/problems/${problemId}`}>問題に戻る</BackLink>
        </div>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="page-title mb-1">
              {isShowingAll ? '全員の提出' : '自分の提出'}
            </h1>
            {problemQuery.data && (
              <Link to={`/problems/${problemId}`} className="link">
                {problemQuery.data.title}
              </Link>
            )}
          </div>
          <button
            onClick={() => currentQuery.refetch()}
            className="btn btn-secondary btn-sm"
            disabled={currentQuery.isFetching}
          >
            <RefreshCw size={14} className={currentQuery.isFetching ? 'animate-spin' : ''} />
            更新
          </button>
        </div>
      </div>

      {/* テーブル */}
      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: '80px' }}>ID</th>
                {isShowingAll && <th style={{ width: '120px' }}>ユーザー</th>}
                <th style={{ width: '80px' }}>結果</th>
                <th style={{ width: '100px' }}>実行時間</th>
                <th style={{ width: '100px' }}>メモリ</th>
                <th style={{ width: '120px' }}>言語</th>
                <th>提出日時</th>
              </tr>
            </thead>
            <tbody>
              {currentQuery.isLoading ? (
                [...Array(5)].map((_, i) => (
                  <tr key={i}>
                    <td><div className="skeleton h-4 w-12" /></td>
                    {isShowingAll && <td><div className="skeleton h-4 w-20" /></td>}
                    <td><div className="skeleton h-6 w-10" /></td>
                    <td><div className="skeleton h-4 w-16" /></td>
                    <td><div className="skeleton h-4 w-16" /></td>
                    <td><div className="skeleton h-4 w-20" /></td>
                    <td><div className="skeleton h-4 w-24" /></td>
                  </tr>
                ))
              ) : items.length > 0 ? (
                items.map((sub: Submission) => (
                  <tr
                    key={sub.id}
                    className="cursor-pointer"
                    onClick={() => handleRowClick(sub.id)}
                  >
                    <td className="mono">{sub.id}</td>
                    {isShowingAll && <td>{sub.userid}</td>}
                    <td><VerdictBadge verdict={sub.verdict} status={sub.status} /></td>
                    <td className="mono text-sm">{sub.time_ms ?? '-'} ms</td>
                    <td className="mono text-sm">{sub.memory_kb ?? '-'} KB</td>
                    <td className="text-sm">{sub.language}</td>
                    <td className="text-sm text-muted">{formatRelativeTime(sub.created_at)}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={isShowingAll ? 7 : 6}>
                    <div className="empty-state">
                      <FileText size={48} className="text-muted opacity-50 mb-4" />
                      <h2 className="empty-state-title">提出がありません</h2>
                      <p className="empty-state-description">
                        {isShowingAll 
                          ? 'この問題にはまだ提出がありません'
                          : 'この問題にまだ提出していません'}
                      </p>
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* ページネーション */}
        {totalPages > 1 && (
          <div className="pagination border-t border-border pt-4 mt-0">
            <button
              className="pagination-btn"
              disabled={page <= 1}
              onClick={() => setPageForActive((p) => Math.max(1, p - 1))}
            >
              <ChevronLeft size={16} />
            </button>
            
            {[...Array(Math.min(totalPages, 5))].map((_, i) => {
              let pageNum: number
              if (totalPages <= 5) {
                pageNum = i + 1
              } else if (page <= 3) {
                pageNum = i + 1
              } else if (page >= totalPages - 2) {
                pageNum = totalPages - 4 + i
              } else {
                pageNum = page - 2 + i
              }
              
              return (
                <button
                  key={pageNum}
                  className={`pagination-btn ${page === pageNum ? 'active' : ''}`}
                  onClick={() => setPageForActive(pageNum)}
                >
                  {pageNum}
                </button>
              )
            })}
            
            <button
              className="pagination-btn"
              disabled={page >= totalPages}
              onClick={() => setPageForActive((p) => p + 1)}
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
