import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { BackLink } from '@/components/common'
import { formatDateShort } from '@/lib/utils'
import { RefreshCw, Plus, Pencil, Trash2, X, Check } from 'lucide-react'

interface Notice {
  id: number
  title: string
  body: string
  created_at: string
  updated_at: string
}

interface NoticesResponse {
  items: Notice[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

export function AdminNotices() {
  const queryClient = useQueryClient()
  
  const [isAdding, setIsAdding] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')

  const noticesQuery = useQuery({
    queryKey: ['admin-notices'],
    queryFn: async (): Promise<NoticesResponse> => {
      return api.admin.notices(1, 100)
    },
    staleTime: 0,
    refetchOnMount: 'always',
  })

  const createMutation = useMutation({
    mutationFn: async () => {
      return api.admin.createNotice({ title, body })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-notices'] })
      queryClient.invalidateQueries({ queryKey: ['notices'] })
      setIsAdding(false)
      setTitle('')
      setBody('')
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (id: number) => {
      return api.admin.updateNotice(id, { title, body })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-notices'] })
      queryClient.invalidateQueries({ queryKey: ['notices'] })
      setEditingId(null)
      setTitle('')
      setBody('')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      return api.admin.deleteNotice(id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-notices'] })
      queryClient.invalidateQueries({ queryKey: ['notices'] })
    },
  })

  const handleEdit = (notice: Notice) => {
    setEditingId(notice.id)
    setTitle(notice.title)
    setBody(notice.body)
    setIsAdding(false)
  }

  const handleCancelEdit = () => {
    setEditingId(null)
    setIsAdding(false)
    setTitle('')
    setBody('')
  }

  const handleDelete = (id: number) => {
    if (confirm('このお知らせを削除しますか？')) {
      deleteMutation.mutate(id)
    }
  }

  const handleStartAdd = () => {
    setIsAdding(true)
    setEditingId(null)
    setTitle('')
    setBody('')
  }

  const notices = noticesQuery.data?.items ?? []

  return (
    <div className="py-8">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>

      <div className="flex items-center justify-between mb-6">
        <h1 className="page-title mb-0">お知らせ管理</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={() => noticesQuery.refetch()}
            disabled={noticesQuery.isFetching}
            className="btn btn-secondary btn-sm"
          >
            <RefreshCw size={14} className={noticesQuery.isFetching ? 'animate-spin' : ''} />
          </button>
          {!isAdding && editingId === null && (
            <button
              onClick={handleStartAdd}
              className="btn btn-primary btn-sm"
            >
              <Plus size={14} />
              新規追加
            </button>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {/* 新規追加フォーム */}
        {isAdding && (
          <div className="card border-primary">
            <div className="card-header">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold">新規お知らせ</h2>
                <button onClick={handleCancelEdit} className="btn btn-ghost btn-sm">
                  <X size={14} />
                  キャンセル
                </button>
              </div>
            </div>
            <div className="card-body">
              <div className="form-group">
                <label htmlFor="new-title" className="label">タイトル</label>
                <input
                  id="new-title"
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="input"
                  placeholder="お知らせのタイトル"
                />
              </div>
              <div className="form-group">
                <label htmlFor="new-body" className="label">本文</label>
                <textarea
                  id="new-body"
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  className="input"
                  rows={5}
                  placeholder="お知らせの内容"
                />
              </div>
              <button
                onClick={() => createMutation.mutate()}
                disabled={createMutation.isPending || !title.trim() || !body.trim()}
                className="btn btn-primary"
              >
                {createMutation.isPending ? (
                  <>
                    <span className="loading-spinner" />
                    追加中...
                  </>
                ) : (
                  <>
                    <Check size={14} />
                    追加する
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        {/* お知らせ一覧 */}
        {noticesQuery.isLoading ? (
          [...Array(3)].map((_, i) => (
            <div key={i} className="card">
              <div className="card-header">
                <div className="skeleton h-6 w-48" />
              </div>
              <div className="card-body">
                <div className="skeleton h-4 w-full mb-2" />
                <div className="skeleton h-4 w-3/4" />
              </div>
            </div>
          ))
        ) : notices.length > 0 ? (
          notices.map((notice) => (
            <div key={notice.id} className={`card ${editingId === notice.id ? 'border-primary' : ''}`}>
              {editingId === notice.id ? (
                // 編集モード
                <>
                  <div className="card-header">
                    <div className="flex items-center justify-between">
                      <h2 className="font-semibold">編集中</h2>
                      <button onClick={handleCancelEdit} className="btn btn-ghost btn-sm">
                        <X size={14} />
                        キャンセル
                      </button>
                    </div>
                  </div>
                  <div className="card-body">
                    <div className="form-group">
                      <label htmlFor={`edit-title-${notice.id}`} className="label">タイトル</label>
                      <input
                        id={`edit-title-${notice.id}`}
                        type="text"
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        className="input"
                      />
                    </div>
                    <div className="form-group">
                      <label htmlFor={`edit-body-${notice.id}`} className="label">本文</label>
                      <textarea
                        id={`edit-body-${notice.id}`}
                        value={body}
                        onChange={(e) => setBody(e.target.value)}
                        className="input"
                        rows={5}
                      />
                    </div>
                    <button
                      onClick={() => updateMutation.mutate(notice.id)}
                      disabled={updateMutation.isPending || !title.trim() || !body.trim()}
                      className="btn btn-primary"
                    >
                      {updateMutation.isPending ? (
                        <>
                          <span className="loading-spinner" />
                          保存中...
                        </>
                      ) : (
                        <>
                          <Check size={14} />
                          保存する
                        </>
                      )}
                    </button>
                  </div>
                </>
              ) : (
                // 表示モード
                <>
                  <div className="card-header">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 min-w-0">
                        <h2 className="font-semibold">{notice.title}</h2>
                        <div className="text-xs text-muted mt-1">
                          作成: {formatDateShort(notice.created_at)}
                          {notice.updated_at !== notice.created_at && (
                            <span className="ml-3">更新: {formatDateShort(notice.updated_at)}</span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-1 flex-shrink-0">
                        <button
                          onClick={() => handleEdit(notice)}
                          className="btn btn-ghost btn-sm"
                          title="編集"
                        >
                          <Pencil size={14} />
                        </button>
                        <button
                          onClick={() => handleDelete(notice.id)}
                          disabled={deleteMutation.isPending}
                          className="btn btn-ghost btn-sm text-destructive hover:bg-destructive/10"
                          title="削除"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                  </div>
                  <div className="card-body">
                    <div className="whitespace-pre-wrap text-sm leading-relaxed">{notice.body}</div>
                  </div>
                </>
              )}
            </div>
          ))
        ) : !isAdding ? (
          <div className="card">
            <div className="card-body">
              <div className="empty-state">
                <h2 className="empty-state-title">お知らせがありません</h2>
                <p className="empty-state-description">
                  「新規追加」ボタンからお知らせを作成できます
                </p>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )
}
