import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { formatTimeLimit, formatMemoryLimit } from '@/lib/utils'
import type { Problem, Submission } from '@/types'
import { CheckCircle2, Circle, Clock, HardDrive, FileText } from 'lucide-react'

export function ProblemsPage() {
  const navigate = useNavigate()

  // 問題一覧を取得
  const { data: problems, isLoading: problemsLoading } = useQuery({
    queryKey: ['problems'],
    queryFn: () => api.problems.list(),
  })

  // 自分の提出一覧を取得（ACの問題IDを特定するため）
  const { data: submissions } = useQuery({
    queryKey: ['submissions', 'mine', 'all'],
    queryFn: async () => {
      // 全ての提出を取得するために大きなper_pageを指定
      const res = await api.submissions.mine(1, 1000)
      return res.items
    },
  })

  const handleRowClick = (problemId: number) => {
    navigate(`/problems/${problemId}`)
  }

  // ACした問題のIDを集める
  const solvedProblemIds = new Set<number>()
  if (submissions) {
    submissions.forEach((s: Submission) => {
      if (s.verdict === 'AC') {
        solvedProblemIds.add(s.problem_id)
      }
    })
  }

  if (problemsLoading) {
    return (
      <div className="py-8">
        <h1 className="page-title">問題一覧</h1>
        <div className="card">
          <div className="table-container overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th style={{ width: '50px' }}></th>
                  <th style={{ width: '60px' }}>ID</th>
                  <th>問題名</th>
                  <th style={{ width: '100px' }} className="hidden sm:table-cell">時間</th>
                  <th style={{ width: '100px' }} className="hidden sm:table-cell">メモリ</th>
                </tr>
              </thead>
              <tbody>
                {[...Array(10)].map((_, i) => (
                  <tr key={i}>
                    <td><div className="skeleton h-6 w-6 rounded-full" /></td>
                    <td><div className="skeleton h-4 w-10" /></td>
                    <td><div className="skeleton h-4 w-32 sm:w-48" /></td>
                    <td className="hidden sm:table-cell"><div className="skeleton h-4 w-16" /></td>
                    <td className="hidden sm:table-cell"><div className="skeleton h-4 w-16" /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    )
  }

  if (!problems || problems.length === 0) {
    return (
      <div className="py-8">
        <h1 className="page-title">問題一覧</h1>
        <div className="card">
          <div className="empty-state">
            <FileText size={48} className="text-muted opacity-50 mb-4" />
            <h2 className="empty-state-title">問題がありません</h2>
            <p className="empty-state-description">
              まだ公開されている問題がありません
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="py-8">
      <h1 className="page-title">問題一覧</h1>
      
      <div className="card">
        <div className="table-container overflow-x-auto">
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: '50px', textAlign: 'center' }}></th>
                <th style={{ width: '60px' }}>ID</th>
                <th>問題名</th>
                <th style={{ width: '100px' }} className="hidden sm:table-cell">
                  <span className="flex items-center gap-1">
                    <Clock size={14} />
                    時間
                  </span>
                </th>
                <th style={{ width: '100px' }} className="hidden sm:table-cell">
                  <span className="flex items-center gap-1">
                    <HardDrive size={14} />
                    メモリ
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              {problems.map((problem: Problem) => {
                const isSolved = solvedProblemIds.has(problem.id)
                return (
                  <tr
                    key={problem.id}
                    className="cursor-pointer"
                    onClick={() => handleRowClick(problem.id)}
                  >
                    <td style={{ textAlign: 'center' }}>
                      {isSolved ? (
                        <span className="solved-check" title="正解済み">
                          <CheckCircle2 size={14} />
                        </span>
                      ) : (
                        <span className="unsolved-check" title="未回答">
                          <Circle size={14} />
                        </span>
                      )}
                    </td>
                    <td className="mono text-muted">{problem.id}</td>
                    <td>
                      <Link
                        to={`/problems/${problem.id}`}
                        className="link font-medium"
                      >
                        {problem.title}
                      </Link>
                    </td>
                    <td className="mono text-sm hidden sm:table-cell">
                      {formatTimeLimit(problem.time_limit_ms)}
                    </td>
                    <td className="mono text-sm hidden sm:table-cell">
                      {formatMemoryLimit(problem.memory_limit_kb)}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
