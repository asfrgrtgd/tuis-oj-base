import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/useAuth'
import { BackLink } from '@/components/common'
import { formatDateOnly } from '@/lib/utils'
import { User, Send, Calendar, CheckCircle } from 'lucide-react'

export function UserProfilePage() {
  const params = useParams()
  const { user: currentUser } = useAuth()
  const userid = params.userid as string

  const { data: profile, isLoading, error } = useQuery({
    queryKey: ['user-profile', userid],
    queryFn: () => api.users.profile(userid),
    enabled: !!userid,
  })

  const isOwnProfile = currentUser?.userid === userid

  if (isLoading) {
    return (
      <div className="py-8">
        <div className="skeleton h-8 w-48 mb-4" />
        <div className="skeleton h-64 w-full" />
      </div>
    )
  }

  if (error || !profile) {
    return (
      <div className="py-8">
        <div className="card">
          <div className="empty-state">
            <User size={48} className="text-muted opacity-50 mb-4" />
            <h2 className="empty-state-title">ユーザーが見つかりません</h2>
            <p className="empty-state-description">
              指定されたユーザーは存在しないか、アクセス権限がありません
            </p>
            <Link to="/problems" className="btn btn-primary mt-4">
              問題一覧に戻る
            </Link>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="py-8">
      {/* 戻るリンク */}
      <div className="mb-4">
        <BackLink to="/problems">問題一覧に戻る</BackLink>
      </div>

      {/* プロフィールヘッダー */}
      <div className="card mb-6">
        <div className="card-body">
          <div className="flex items-center gap-6">
            {/* アバター */}
            <div className="w-20 h-20 rounded-full bg-gradient-to-br from-primary/20 to-primary/40 flex items-center justify-center text-primary">
              <User size={40} />
            </div>
            
            {/* ユーザー情報 */}
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl font-bold">{profile.userid}</h1>
                {isOwnProfile && (
                  <span className="badge badge-success">あなた</span>
                )}
              </div>
              <div className="flex items-center gap-2 text-muted text-sm">
                <Calendar size={14} />
                <span>{formatDateOnly(profile.created_at)} に登録</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 統計カード */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* 正解数 */}
        <div className="card">
          <div className="card-body">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-lg bg-success/10 flex items-center justify-center text-success">
                <CheckCircle size={24} />
              </div>
              <div>
                <p className="text-sm text-muted mb-1">正解した問題数</p>
                <p className="text-3xl font-bold text-success">{profile.solved_count}</p>
              </div>
            </div>
          </div>
        </div>

        {/* 提出数 */}
        <div className="card">
          <div className="card-body">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <Send size={24} />
              </div>
              <div>
                <p className="text-sm text-muted mb-1">総提出数</p>
                <p className="text-3xl font-bold text-primary">{profile.submission_count}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
