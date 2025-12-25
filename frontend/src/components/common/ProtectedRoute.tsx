import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'

/**
 * 認証が必要なページを保護するコンポーネント
 * 未ログインの場合はログインページにリダイレクトする
 */
export function ProtectedRoute() {
  const { user, isLoading } = useAuth()

  // 認証状態の読み込み中はローディング表示
  if (isLoading) {
    return (
      <div className="min-h-[calc(100vh-200px)] flex items-center justify-center">
        <div className="text-center">
          <div className="loading-spinner mx-auto mb-4"></div>
          <p className="text-muted">読み込み中...</p>
        </div>
      </div>
    )
  }

  // 未ログインの場合はログインページにリダイレクト
  if (!user) {
    return <Navigate to="/login" replace />
  }

  // ログイン済みの場合は子コンポーネントをレンダリング
  return <Outlet />
}

/**
 * 管理者専用ページを保護するコンポーネント
 * 未ログインの場合はログインページに、一般ユーザーの場合は問題一覧にリダイレクト
 */
export function AdminRoute() {
  const { user, isLoading, isAdmin } = useAuth()

  // 認証状態の読み込み中はローディング表示
  if (isLoading) {
    return (
      <div className="min-h-[calc(100vh-200px)] flex items-center justify-center">
        <div className="text-center">
          <div className="loading-spinner mx-auto mb-4"></div>
          <p className="text-muted">読み込み中...</p>
        </div>
      </div>
    )
  }

  // 未ログインの場合はログインページにリダイレクト
  if (!user) {
    return <Navigate to="/login" replace />
  }

  // 管理者でない場合は問題一覧にリダイレクト
  if (!isAdmin) {
    return <Navigate to="/problems" replace />
  }

  // 管理者の場合は子コンポーネントをレンダリング
  return <Outlet />
}

