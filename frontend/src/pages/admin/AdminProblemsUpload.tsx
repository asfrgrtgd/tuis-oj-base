import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Alert } from '@/components/ui/Alert'
import { BackLink } from '@/components/common'
import { CloudDownload, Upload } from 'lucide-react'

export function AdminProblemsUpload() {
  const queryClient = useQueryClient()
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null)

  const importMutation = useMutation({
    mutationFn: () => {
      if (!file) throw new Error('ファイルを選択してください')
      return api.admin.importProblem(file)
    },
    onSuccess: (data) => {
      setResult({ success: true, message: `問題をインポートしました: ${JSON.stringify(data, null, 2)}` })
      // 問題一覧のキャッシュを最新化
      queryClient.invalidateQueries({ queryKey: ['admin-problems'], exact: false })
      queryClient.invalidateQueries({ queryKey: ['problems'], exact: false })
      setFile(null)
    },
    onError: (err: Error) => {
      setResult({ success: false, message: err.message || 'インポートに失敗しました' })
    },
  })

  const handleDownloadTemplate = async () => {
    try {
      const blob = await api.admin.downloadTemplate()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'two-string.zip'
      a.click()
      URL.revokeObjectURL(url)
      setResult({ success: true, message: 'テンプレートをダウンロードしました' })
    } catch {
      setResult({ success: false, message: 'テンプレートのダウンロードに失敗しました' })
    }
  }

  return (
    <div className="py-8 max-w-4xl mx-auto space-y-6">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>
      
      <h1 className="page-title">問題アップロード</h1>

      <div className="space-y-6">
        {/* テンプレートダウンロード */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold">テンプレートダウンロード</h2>
          </div>
          <div className="card-body space-y-3">
            <p className="text-sm text-muted">
              問題作成用のテンプレートをダウンロードします。
              テンプレートは slug と同名フォルダ（例: two-string/）配下に problem.yaml / statement.md / data が入っています。そのまま再圧縮せずアップロードできます。
            </p>
            <button
              onClick={handleDownloadTemplate}
              className="btn btn-secondary w-full"
            >
              <CloudDownload size={16} />
              テンプレートをダウンロード
            </button>
          </div>
        </div>

        {/* 問題アップロード */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold">問題アップロード</h2>
          </div>
          <div className="card-body space-y-3">
            <p className="text-sm text-muted">
              問題パッケージ（ZIP形式）をアップロードして問題を登録します。
            </p>
            
            <div className="form-group">
              <label className="label">ZIPファイル</label>
              <input
                id="problem-file"
                type="file"
                accept=".zip"
                onChange={(e) => {
                  setFile(e.target.files?.[0] ?? null)
                  setResult(null)
                }}
                className="hidden"
              />
              <label
                htmlFor="problem-file"
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
              onClick={() => importMutation.mutate()}
              disabled={importMutation.isPending || !file}
              className="btn btn-primary w-full"
            >
              {importMutation.isPending ? (
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
          </div>
        </div>

        {/* 結果表示 */}
        {result && (
          <Alert variant={result.success ? 'success' : 'error'}>
            <pre className="whitespace-pre-wrap break-all">{result.message}</pre>
          </Alert>
        )}
      </div>
    </div>
  )
}
