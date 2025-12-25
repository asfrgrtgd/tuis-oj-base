import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="min-h-[calc(100vh-200px)] flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-muted mb-4">404</h1>
        <h2 className="text-xl font-semibold mb-2">ページが見つかりません</h2>
        <p className="text-muted mb-6">
          お探しのページは存在しないか、移動した可能性があります
        </p>
        <Link to="/problems" className="btn btn-primary">
          問題一覧に戻る
        </Link>
      </div>
    </div>
  )
}
