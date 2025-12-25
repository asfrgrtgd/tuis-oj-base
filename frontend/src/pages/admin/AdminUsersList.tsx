import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { BackLink } from '@/components/common'
import { formatDateShort } from '@/lib/utils'
import { Search, User, ExternalLink } from 'lucide-react'

export function AdminUsersList() {
  const [page, setPage] = useState(1)
  const [searchQuery, setSearchQuery] = useState('')
  const perPage = 20

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-users', page, perPage],
    queryFn: () => api.admin.users(page, perPage),
    staleTime: 0,
    refetchOnMount: 'always',
  })

  // 検索フィルタリング
  const filteredUsers = data?.items?.filter(u => 
    u.userid.toLowerCase().includes(searchQuery.toLowerCase())
  ) ?? []

  if (isLoading) {
    return (
      <div className="py-8">
        <div className="skeleton h-8 w-48 mb-4" />
        <div className="skeleton h-64 w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="py-8">
        <div className="card">
          <div className="card-body text-center text-destructive">
            ユーザー一覧の取得に失敗しました
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="py-8">
      {/* ヘッダー */}
      <div className="mb-6">
        <div className="mb-4">
          <BackLink to="/admin">管理画面に戻る</BackLink>
        </div>
        
        <h1 className="page-title">ユーザー管理</h1>
        <p className="text-muted">登録されているユーザーの一覧と統計情報を確認できます</p>
      </div>

      {/* 検索・統計 */}
      <div className="card mb-6">
        <div className="card-body">
          <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
            <div className="relative w-full sm:w-80">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input
                type="text"
                placeholder="ユーザーIDで検索..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input pl-9 w-full"
              />
            </div>
            <div className="flex items-center gap-4 text-sm text-muted">
              <span>総ユーザー数: <strong className="text-foreground">{data?.total_items ?? 0}</strong></span>
            </div>
          </div>
        </div>
      </div>

      {/* ユーザー一覧 */}
      <div className="card">
        <div className="card-header">
          <h2 className="font-semibold">ユーザー一覧</h2>
        </div>
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>ユーザーID</th>
                <th style={{ width: '100px' }}>役割</th>
                <th style={{ width: '160px' }}>登録日時</th>
                <th style={{ width: '80px' }}>詳細</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.length === 0 ? (
                <tr>
                  <td colSpan={4} className="text-center text-muted py-8">
                    {searchQuery ? '該当するユーザーが見つかりません' : 'ユーザーがいません'}
                  </td>
                </tr>
              ) : (
                filteredUsers.map((u) => (
                  <tr key={u.id}>
                    <td>
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                          <User size={16} />
                        </div>
                        <span className="font-medium">{u.userid}</span>
                      </div>
                    </td>
                    <td>
                      {u.role === 'admin' ? (
                        <span className="badge badge-info">Admin</span>
                      ) : (
                        <span className="badge badge-neutral">User</span>
                      )}
                    </td>
                    <td className="text-muted">
                      {formatDateShort(u.created_at)}
                    </td>
                    <td>
                      <Link
                        to={`/users/${u.userid}`}
                        className="btn btn-ghost btn-sm inline-flex items-center gap-1"
                      >
                        <ExternalLink size={14} />
                        詳細
                      </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* ページネーション */}
        {data && data.total_pages > 1 && (
          <div className="card-body border-t border-border">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted">
                {data.total_items} 件中 {(page - 1) * perPage + 1} - {Math.min(page * perPage, data.total_items)} 件を表示
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="btn btn-secondary btn-sm"
                >
                  前へ
                </button>
                <span className="flex items-center px-3 text-sm">
                  {page} / {data.total_pages}
                </span>
                <button
                  onClick={() => setPage(p => Math.min(data.total_pages, p + 1))}
                  disabled={page === data.total_pages}
                  className="btn btn-secondary btn-sm"
                >
                  次へ
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
