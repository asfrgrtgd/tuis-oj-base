import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { Alert } from '@/components/ui/Alert'

export function LoginPage() {
  const navigate = useNavigate()
  const { login, isLoginLoading, user, isLoading } = useAuth()
  const [userid, setUserid] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  // 既にログイン済みならリダイレクト（useEffect内で副作用として実行）
  useEffect(() => {
    if (!isLoading && user) {
      navigate('/problems', { replace: true })
    }
  }, [user, isLoading, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!userid.trim() || !password.trim()) {
      setError('ユーザーIDとパスワードを入力してください')
      return
    }

    try {
      await login({ userid: userid.trim(), password })
      navigate('/problems', { replace: true })
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as { response?: { data?: { error?: { message?: string } } } }
        setError(
          axiosError.response?.data?.error?.message || 'ログインに失敗しました'
        )
      } else {
        setError('ログインに失敗しました')
      }
    }
  }

  return (
    <div className="min-h-[calc(100vh-200px)] flex items-center justify-center">
      <div className="w-full max-w-md">
        <div className="card">
          <div className="card-header text-center">
            <div className="flex items-center justify-center gap-2 mb-2">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-primary">
                <path d="M12 2L2 7l10 5 10-5-10-5z" />
                <path d="M2 17l10 5 10-5" />
                <path d="M2 12l10 5 10-5" />
              </svg>
            </div>
            <h1 className="text-xl font-bold">TUIS Online Judge</h1>
            <p className="text-sm text-muted mt-1">ログインして続行</p>
          </div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              {error && (
                <Alert variant="error" className="mb-4">
                  {error}
                </Alert>
              )}

              <div className="form-group">
                <label htmlFor="userid" className="label">
                  ユーザーID
                </label>
                <input
                  id="userid"
                  type="text"
                  value={userid}
                  onChange={(e) => setUserid(e.target.value)}
                  className="input"
                  placeholder="ユーザーIDを入力"
                  autoComplete="username"
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label htmlFor="password" className="label">
                  パスワード
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="input"
                  placeholder="パスワードを入力"
                  autoComplete="current-password"
                />
              </div>

              <button
                type="submit"
                disabled={isLoginLoading}
                className="btn btn-primary w-full mt-2"
              >
                {isLoginLoading ? (
                  <>
                    <span className="loading-spinner"></span>
                    ログイン中...
                  </>
                ) : (
                  'ログイン'
                )}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  )
}
