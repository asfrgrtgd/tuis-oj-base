import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { Bell, RefreshCw } from 'lucide-react'

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

export function NoticesPage() {
  const noticesQuery = useQuery({
    queryKey: ['notices'],
    queryFn: async (): Promise<NoticesResponse> => {
      return api.notices.list(1, 50)
    },
  })

  const notices = noticesQuery.data?.items ?? []

  return (
    <div className="py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="page-title mb-0">お知らせ</h1>
        <button
          onClick={() => noticesQuery.refetch()}
          disabled={noticesQuery.isFetching}
          className="btn btn-secondary btn-sm"
        >
          <RefreshCw size={14} className={noticesQuery.isFetching ? 'animate-spin' : ''} />
          更新
        </button>
      </div>

      {noticesQuery.isLoading ? (
        <div className="space-y-4">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="card">
              <div className="card-header">
                <div className="skeleton h-6 w-48" />
              </div>
              <div className="card-body">
                <div className="skeleton h-4 w-full mb-2" />
                <div className="skeleton h-4 w-3/4" />
              </div>
            </div>
          ))}
        </div>
      ) : notices.length > 0 ? (
        <div className="space-y-4">
          {notices.map((notice) => (
            <article key={notice.id} className="card">
              <div className="card-header">
                <div className="flex items-start justify-between gap-4">
                  <h2 className="font-semibold">{notice.title}</h2>
                  <div className="text-xs text-muted whitespace-nowrap">
                    <div>{formatDate(notice.created_at)}</div>
                    {notice.updated_at !== notice.created_at && (
                      <div className="text-right">（更新: {formatDate(notice.updated_at)}）</div>
                    )}
                  </div>
                </div>
              </div>
              <div className="card-body">
                <div className="whitespace-pre-wrap text-sm leading-relaxed">{notice.body}</div>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="card">
          <div className="card-body">
            <div className="empty-state">
              <Bell size={48} className="mx-auto text-muted mb-4 opacity-50" />
              <h2 className="empty-state-title">お知らせはありません</h2>
              <p className="empty-state-description">
                新しいお知らせが投稿されるとここに表示されます
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
