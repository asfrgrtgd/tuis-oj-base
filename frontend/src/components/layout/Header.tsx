import { useState, useRef, useEffect } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { ChevronDown } from 'lucide-react'

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout, isLogoutLoading } = useAuth()
  const [isSubmissionsOpen, setIsSubmissionsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // 現在のパスから問題IDを抽出（/problems/:id または /problems/:id/submissions）
  const problemMatch = location.pathname.match(/^\/problems\/(\d+)/)
  const currentProblemId = problemMatch ? problemMatch[1] : null

  const isActive = (path: string) => location.pathname === path
  const isProblemsActive = location.pathname === '/problems'
  const isSubmissionsActive = currentProblemId && location.pathname.startsWith(`/problems/${currentProblemId}/submissions`)

  // ドロップダウン外をクリックしたら閉じる
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsSubmissionsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // ページ遷移したらドロップダウンを閉じる
  useEffect(() => {
    setIsSubmissionsOpen(false)
  }, [location.pathname])

  const handleLogout = async () => {
    try {
      await logout()
      window.location.href = '/login'
    } catch {
      // エラー処理
    }
  }

  const handleSubmissionSelect = (tab: 'mine' | 'all') => {
    setIsSubmissionsOpen(false)
    navigate(`/problems/${currentProblemId}/submissions?tab=${tab}`)
  }

  return (
    <header className="header">
      <div className="container">
        <div className="header-inner">
          <div className="flex items-center gap-8">
            <Link to="/" className="logo">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2L2 7l10 5 10-5-10-5z" />
                <path d="M2 17l10 5 10-5" />
                <path d="M2 12l10 5 10-5" />
              </svg>
              TUIS OJ
            </Link>

            {user && (
              <nav className="nav">
                <Link
                  to="/problems"
                  className={`nav-link ${isProblemsActive ? 'active' : ''}`}
                >
                  問題一覧
                </Link>
                <Link
                  to="/notices"
                  className={`nav-link ${location.pathname === '/notices' ? 'active' : ''}`}
                >
                  お知らせ
                </Link>
                {/* 問題ページを開いている時だけ「提出一覧」を表示 */}
                {currentProblemId && (
                  <div className="relative" ref={dropdownRef}>
                    <button
                      onClick={() => setIsSubmissionsOpen(!isSubmissionsOpen)}
                      className={`nav-link flex items-center gap-1 ${isSubmissionsActive ? 'active' : ''}`}
                    >
                      提出一覧
                      <ChevronDown size={14} className={`transition-transform ${isSubmissionsOpen ? 'rotate-180' : ''}`} />
                    </button>
                    {isSubmissionsOpen && (
                      <div className="absolute top-full left-0 mt-1 bg-white border border-border rounded-md shadow-lg py-1 min-w-[160px] z-50">
                        <button
                          onClick={() => handleSubmissionSelect('mine')}
                          className="dropdown-item"
                        >
                          自分の提出
                        </button>
                        <button
                          onClick={() => handleSubmissionSelect('all')}
                          className="dropdown-item"
                        >
                          全員の提出
                        </button>
                      </div>
                    )}
                  </div>
                )}
                {user.role === 'admin' && (
                  <Link
                    to="/admin"
                    className={`nav-link ${isActive('/admin') ? 'active' : ''}`}
                  >
                    管理
                  </Link>
                )}
              </nav>
            )}
          </div>

          <div className="flex items-center gap-4">
            {user ? (
              <>
                <span className="text-sm text-muted">
                  <Link
                    to={`/users/${user.userid}`}
                    className="font-medium text-foreground hover:text-primary transition-colors"
                  >
                    {user.userid}
                  </Link>
                  {user.role === 'admin' && (
                    <span className="ml-2 badge badge-info">Admin</span>
                  )}
                </span>
                <button
                  onClick={handleLogout}
                  disabled={isLogoutLoading}
                  className="btn btn-secondary btn-sm"
                >
                  {isLogoutLoading ? 'ログアウト中...' : 'ログアウト'}
                </button>
              </>
            ) : (
              <Link to="/login" className="btn btn-primary btn-sm">
                ログイン
              </Link>
            )}
          </div>
        </div>
      </div>
    </header>
  )
}
