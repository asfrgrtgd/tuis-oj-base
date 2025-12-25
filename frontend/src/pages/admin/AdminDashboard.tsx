import { Link } from 'react-router-dom'
import { Upload, Eye, Users, Activity, Bell, FlaskConical, UserCog } from 'lucide-react'

const menuItems = [
  {
    title: 'お知らせ管理',
    description: 'お知らせの追加・編集・削除',
    icon: Bell,
    path: '/admin/notices',
  },
  {
    title: '問題アップロード',
    description: '問題テンプレートのダウンロードと問題のアップロード',
    icon: Upload,
    path: '/admin/problems/upload',
  },
  {
    title: '問題公開設定',
    description: '問題の公開/非公開を管理',
    icon: Eye,
    path: '/admin/problems/visibility',
  },
  {
    title: 'ユーザー管理',
    description: 'ユーザー一覧の閲覧、提出数・AC数の確認',
    icon: UserCog,
    path: '/admin/users',
  },
  {
    title: 'ユーザー追加',
    description: 'ユーザーの単体追加とCSVによる一括追加',
    icon: Users,
    path: '/admin/users/bulk',
  },
  {
    title: '提出テスト',
    description: 'ジャッジの動作確認・負荷テスト用の一括提出',
    icon: FlaskConical,
    path: '/admin/submissions/test',
  },
  {
    title: 'システム状態',
    description: 'ワーカー、キュー、メモリ使用状況の監視',
    icon: Activity,
    path: '/admin/system',
  },
]

export function AdminDashboard() {
  return (
    <div className="py-8">
      <h1 className="page-title">管理画面</h1>
      
      <div className="grid gap-4 md:grid-cols-2">
        {menuItems.map((item) => (
          <Link
            key={item.path}
            to={item.path}
            className="card hover:border-primary transition-colors"
          >
            <div className="card-body flex items-start gap-4">
              <div className="p-3 bg-secondary rounded-lg">
                <item.icon size={24} className="text-primary" />
              </div>
              <div>
                <h2 className="font-semibold text-lg mb-1">{item.title}</h2>
                <p className="text-sm text-muted">{item.description}</p>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}
