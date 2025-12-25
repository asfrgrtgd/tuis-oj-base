import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { BackLink } from '@/components/common'
import { RefreshCw, Eye, EyeOff } from 'lucide-react'

interface AdminProblem {
  id: number
  title: string
  slug: string
  visibility: 'public' | 'hidden'
  solved_count: number
  submission_count: number
}

interface AdminProblemsResponse {
  items: AdminProblem[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

export function AdminProblemsVisibility() {
  const queryClient = useQueryClient()

  const problemsQuery = useQuery({
    queryKey: ['admin-problems'],
    queryFn: async (): Promise<AdminProblemsResponse> => {
      return api.admin.problems(1, 100)
    },
    staleTime: 0,
    refetchOnMount: 'always',
  })

  const toggleMutation = useMutation({
    mutationFn: async ({ id, isPublic }: { id: number; isPublic: boolean }) => {
      return api.admin.updateProblemVisibility(id, isPublic)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-problems'] })
    },
  })

  const problems = problemsQuery.data?.items ?? []

  return (
    <div className="py-8">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>

      <div className="flex items-center justify-between mb-6">
        <h1 className="page-title mb-0">問題公開設定</h1>
        <button
          onClick={() => problemsQuery.refetch()}
          disabled={problemsQuery.isFetching}
          className="btn btn-secondary btn-sm"
        >
          <RefreshCw size={14} className={problemsQuery.isFetching ? 'animate-spin' : ''} />
          更新
        </button>
      </div>

      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: '80px' }}>ID</th>
                <th>タイトル</th>
                <th style={{ width: '120px' }}>Slug</th>
                <th style={{ width: '100px' }}>提出数</th>
                <th style={{ width: '100px' }}>正解数</th>
                <th style={{ width: '120px' }}>公開状態</th>
                <th style={{ width: '100px' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {problemsQuery.isLoading ? (
                [...Array(5)].map((_, i) => (
                  <tr key={i}>
                    <td><div className="skeleton h-4 w-12" /></td>
                    <td><div className="skeleton h-4 w-32" /></td>
                    <td><div className="skeleton h-4 w-20" /></td>
                    <td><div className="skeleton h-4 w-12" /></td>
                    <td><div className="skeleton h-4 w-12" /></td>
                    <td><div className="skeleton h-6 w-16" /></td>
                    <td><div className="skeleton h-8 w-16" /></td>
                  </tr>
                ))
              ) : problems.length > 0 ? (
                problems.map((problem) => {
                  const isPublic = problem.visibility === 'public'
                  return (
                    <tr key={problem.id}>
                      <td className="mono">{problem.id}</td>
                      <td className="font-medium">{problem.title}</td>
                      <td className="mono text-sm text-muted">{problem.slug}</td>
                      <td className="mono">{problem.submission_count}</td>
                      <td className="mono">{problem.solved_count}</td>
                      <td>
                        {isPublic ? (
                          <span className="badge badge-success flex items-center gap-1 w-fit">
                            <Eye size={12} />
                            公開
                          </span>
                        ) : (
                          <span className="badge badge-neutral flex items-center gap-1 w-fit">
                            <EyeOff size={12} />
                            非公開
                          </span>
                        )}
                      </td>
                      <td>
                        <button
                          onClick={() => toggleMutation.mutate({
                            id: problem.id,
                            isPublic: !isPublic
                          })}
                          disabled={toggleMutation.isPending}
                          className={`btn btn-sm ${isPublic ? 'btn-secondary' : 'btn-primary'}`}
                        >
                          {isPublic ? '非公開に' : '公開する'}
                        </button>
                      </td>
                    </tr>
                  )
                })
              ) : (
                <tr>
                  <td colSpan={7}>
                    <div className="empty-state">
                      <h2 className="empty-state-title">問題がありません</h2>
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
