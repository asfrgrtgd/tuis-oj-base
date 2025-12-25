import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Alert } from '@/components/ui/Alert'
import { BackLink } from '@/components/common'
import { Upload, CheckCircle, FileText, UserPlus } from 'lucide-react'

interface BulkResult {
  created_count: number
  failed_count: number
  failed_rows: Array<{
    row_number: number
    userid: string
    reason: string
  }>
}

export function AdminUsersBulk() {
  const queryClient = useQueryClient()

  // 単体追加
  const [userid, setUserid] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<'user' | 'admin'>('user')
  const [singleResult, setSingleResult] = useState<{ success: boolean; message: string } | null>(null)

  // 一括追加
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<{ success: boolean; data?: BulkResult; message?: string } | null>(null)

  const createMutation = useMutation({
    mutationFn: () => api.admin.createUser({ userid, password, role }),
    onSuccess: (data) => {
      setSingleResult({ success: true, message: `ユーザーを作成しました: ${data.userid}` })
      queryClient.invalidateQueries({ queryKey: ['admin-users'], exact: false })
      setUserid('')
      setPassword('')
      setRole('user')
    },
    onError: (err: Error) => {
      setSingleResult({ success: false, message: err.message || 'ユーザー作成に失敗しました' })
    },
  })

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!file) throw new Error('ファイルを選択してください')
      return api.admin.bulkCreateUsers(file)
    },
    onSuccess: (data) => {
      const normalized: BulkResult = {
        created_count: data?.created_count ?? 0,
        failed_count: data?.failed_count ?? 0,
        failed_rows: data?.failed_rows ?? [],
      }
      setResult({ success: true, data: normalized })
      // ユーザー一覧キャッシュを最新化
      queryClient.invalidateQueries({ queryKey: ['admin-users'], exact: false })
      setFile(null)
    },
    onError: (err: Error) => {
      setResult({ success: false, message: err.message || 'アップロードに失敗しました' })
    },
  })

  return (
    <div className="py-8 max-w-4xl mx-auto space-y-6">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>

      <h1 className="page-title">ユーザー追加</h1>

      <div className="grid gap-4 md:grid-cols-2">
        {/* 単体追加 */}
        <div className="card h-full flex flex-col">
          <div className="card-header">
            <h2 className="font-semibold">単体ユーザー追加</h2>
          </div>
          <div className="card-body space-y-3">
            <div className="form-group">
              <label htmlFor="userid" className="label">ユーザーID</label>
              <input
                id="userid"
                type="text"
                value={userid}
                onChange={(e) => setUserid(e.target.value)}
                className="input"
                placeholder="例: user001"
              />
            </div>

            <div className="form-group">
              <label htmlFor="password" className="label">パスワード</label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input"
                placeholder="初期パスワード"
              />
            </div>

            <div className="form-group">
              <label htmlFor="role" className="label">ロール</label>
              <select
                id="role"
                value={role}
                onChange={(e) => setRole(e.target.value as 'user' | 'admin')}
                className="input"
              >
                <option value="user">一般ユーザー</option>
                <option value="admin">管理者</option>
              </select>
            </div>

            <button
              type="button"
              disabled={createMutation.isPending}
              className="btn btn-primary w-full"
              onClick={() => {
                setSingleResult(null)
                if (!userid.trim()) {
                  setSingleResult({ success: false, message: 'ユーザーIDを入力してください' })
                  return
                }
                if (!password.trim()) {
                  setSingleResult({ success: false, message: 'パスワードを入力してください' })
                  return
                }
                createMutation.mutate()
              }}
            >
              {createMutation.isPending ? (
                <>
                  <span className="loading-spinner" />
                  作成中...
                </>
              ) : (
                <>
                  <UserPlus size={16} />
                  ユーザーを作成
                </>
              )}
            </button>

            {singleResult && (
              <Alert variant={singleResult.success ? 'success' : 'error'}>
                {singleResult.message}
              </Alert>
            )}
          </div>
        </div>

        {/* CSV一括追加 */}
        <div className="card h-full flex flex-col">
          <div className="card-header flex items-center gap-2">
            <FileText size={16} />
            <h2 className="font-semibold">CSV一括追加</h2>
          </div>
          <div className="card-body space-y-3">
            <div>
              <p className="text-sm text-muted mb-3">
                以下の形式のCSVファイルをアップロードしてください。1行目はヘッダーです。
              </p>
              <pre className="code text-sm">
{`userid,password
user001,password123
user002,password456
user003,password789`}
              </pre>
            </div>

            <div className="form-group">
              <label htmlFor="csv-file" className="label">CSVファイル</label>
              <input
                id="csv-file"
                type="file"
                accept=".csv"
                onChange={(e) => {
                  setFile(e.target.files?.[0] ?? null)
                  setResult(null)
                }}
                className="hidden"
              />
              <label
                htmlFor="csv-file"
                className="btn btn-secondary w-full justify-between"
              >
                <span>{file?.name ? file.name : 'ファイルを選択'}</span>
                <span className="text-sm text-muted">参照</span>
              </label>
              <p className="text-sm text-muted mt-2">
                {file ? file.name : 'ファイルが選択されていません。'}
              </p>
            </div>

            <button
              onClick={() => uploadMutation.mutate()}
              disabled={uploadMutation.isPending || !file}
              className="btn btn-primary w-full"
            >
              {uploadMutation.isPending ? (
                <>
                  <span className="loading-spinner" />
                  アップロード中...
                </>
              ) : (
                <>
                  <Upload size={16} />
                  アップロード
                </>
              )}
            </button>

            {result && (
              result.success && result.data ? (
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <CheckCircle size={20} className="text-success" />
                    <span className="font-semibold">処理完了</span>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-success/10 rounded">
                      <p className="text-sm text-muted">成功</p>
                      <p className="text-2xl font-bold text-success">{result.data.created_count}</p>
                    </div>
                    <div className="p-3 bg-destructive/10 rounded">
                      <p className="text-sm text-muted">失敗</p>
                      <p className="text-2xl font-bold text-destructive">{result.data.failed_count}</p>
                    </div>
                  </div>
                  {(result.data.failed_rows?.length ?? 0) > 0 && (
                    <div>
                      <p className="text-sm font-medium mb-2">失敗した行:</p>
                      <div className="max-h-40 overflow-auto">
                        <table className="table text-sm">
                          <thead>
                            <tr>
                              <th>行番号</th>
                              <th>ユーザーID</th>
                              <th>理由</th>
                            </tr>
                          </thead>
                          <tbody>
                            {result.data.failed_rows.map((row, i) => (
                              <tr key={i}>
                                <td>{row.row_number}</td>
                                <td>{row.userid}</td>
                                <td className="text-destructive">{row.reason}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <Alert variant="error">
                  {result.message}
                </Alert>
              )
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
