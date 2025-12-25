import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/lib/api'
import { Alert } from '@/components/ui/Alert'
import { BackLink } from '@/components/common'
import { Play, CheckCircle } from 'lucide-react'

interface BulkTestResponse {
  created: number[]
  count: number
  problem: number
  language: string
}

const LANGUAGES = [
  { value: 'c', label: 'C (GCC)' },
  { value: 'python', label: 'Python 3' },
  { value: 'java', label: 'Java' },
]

const DEFAULT_CODES: Record<string, string> = {
  c: `#include <stdio.h>
int main() {
    int a, b;
    scanf("%d %d", &a, &b);
    printf("%d\\n", a + b);
    return 0;
}`,
  python: `a, b = map(int, input().split())
print(a + b)`,
  java: `import java.util.Scanner;
public class Main {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int a = sc.nextInt();
        int b = sc.nextInt();
        System.out.println(a + b);
    }
}`,
}

export function AdminSubmissionTest() {
  const [problemId, setProblemId] = useState('1')
  const [language, setLanguage] = useState('c')
  const [count, setCount] = useState('10')
  const [sourceCode, setSourceCode] = useState(DEFAULT_CODES['c'])
  const [result, setResult] = useState<BulkTestResponse | null>(null)

  // 問題一覧を取得
  const problemsQuery = useQuery({
    queryKey: ['problems'],
    queryFn: () => api.problems.list(),
  })

  const bulkTestMutation = useMutation({
    mutationFn: async () => {
      return api.admin.bulkSubmit({
        problem_id: parseInt(problemId, 10),
        language,
        count: parseInt(count, 10),
        source_code: sourceCode,
      })
    },
    onSuccess: (data) => {
      setResult(data)
    },
  })

  const handleLanguageChange = (newLang: string) => {
    setLanguage(newLang)
    // デフォルトコードに変更
    if (DEFAULT_CODES[newLang]) {
      setSourceCode(DEFAULT_CODES[newLang])
    }
  }

  return (
    <div className="py-8">
      <div className="mb-4">
        <BackLink to="/admin">管理画面に戻る</BackLink>
      </div>

      <h1 className="page-title">提出テスト</h1>
      <p className="text-muted mb-6">
        指定した問題に対して複数の提出を一括で作成し、ジャッジの動作確認やワーカーの負荷テストを行います。
      </p>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* 設定フォーム */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold">テスト設定</h2>
          </div>
          <div className="card-body space-y-4">
            {/* 問題選択 */}
            <div className="form-group">
              <label className="label">問題</label>
              <select
                value={problemId}
                onChange={(e) => setProblemId(e.target.value)}
                className="input"
              >
                {problemsQuery.data?.map((p) => (
                  <option key={p.id} value={p.id}>
                    #{p.id} - {p.title}
                  </option>
                ))}
              </select>
              <p className="text-xs text-muted mt-1">
                テスト対象の問題を選択してください
              </p>
            </div>

            {/* 言語選択 */}
            <div className="form-group">
              <label className="label">言語</label>
              <select
                value={language}
                onChange={(e) => handleLanguageChange(e.target.value)}
                className="input"
              >
                {LANGUAGES.map((l) => (
                  <option key={l.value} value={l.value}>
                    {l.label}
                  </option>
                ))}
              </select>
            </div>

            {/* 提出数 */}
            <div className="form-group">
              <label className="label">提出数</label>
              <input
                type="number"
                value={count}
                onChange={(e) => setCount(e.target.value)}
                min={1}
                max={100}
                className="input"
              />
              <p className="text-xs text-muted mt-1">
                1〜100の範囲で指定（デフォルト: 10）
              </p>
            </div>

            {/* ソースコード */}
            <div className="form-group">
              <label className="label">ソースコード</label>
              <textarea
                value={sourceCode}
                onChange={(e) => setSourceCode(e.target.value)}
                rows={12}
                className="input mono text-sm"
                placeholder="ソースコードを入力..."
              />
              <p className="text-xs text-muted mt-1">
                空欄の場合は言語に応じたデフォルトコード（A+B解答）が使用されます
              </p>
            </div>

            {/* 実行ボタン */}
            <button
              onClick={() => bulkTestMutation.mutate()}
              disabled={bulkTestMutation.isPending}
              className="btn btn-primary w-full"
            >
              {bulkTestMutation.isPending ? (
                <>
                  <span className="loading-spinner" />
                  提出中...
                </>
              ) : (
                <>
                  <Play size={16} />
                  テスト提出を実行
                </>
              )}
            </button>

            {bulkTestMutation.isError && (
              <Alert variant="error">
                エラーが発生しました: {(bulkTestMutation.error as Error).message}
              </Alert>
            )}
          </div>
        </div>

        {/* 結果表示 */}
        <div className="card">
          <div className="card-header">
            <h2 className="font-semibold">実行結果</h2>
          </div>
          <div className="card-body">
            {result ? (
              <div className="space-y-4">
                <Alert variant="success">
                  <div className="flex items-start gap-2">
                    <CheckCircle size={18} className="flex-shrink-0 mt-0.5" />
                    <div>
                      <p className="font-medium">{result.count}件の提出を作成しました</p>
                      <p className="text-sm mt-1">
                        問題: #{result.problem} / 言語: {result.language}
                      </p>
                    </div>
                  </div>
                </Alert>

                <div>
                  <h3 className="text-sm font-medium mb-2">作成された提出ID</h3>
                  <div className="flex flex-wrap gap-2">
                    {result.created.map((id) => (
                      <Link
                        key={id}
                        to={`/submissions/${id}`}
                        className="px-2 py-1 bg-secondary rounded text-sm hover:bg-primary hover:text-white transition-colors"
                      >
                        #{id}
                      </Link>
                    ))}
                  </div>
                </div>

                <div className="pt-4 border-t border-border">
                  <Link
                    to="/admin/system"
                    className="btn btn-secondary w-full"
                  >
                    システム状態を確認
                  </Link>
                </div>
              </div>
            ) : (
              <div className="empty-state">
                <Play size={48} className="mx-auto mb-4 opacity-30" />
                <p>テストを実行すると結果がここに表示されます</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 注意事項 */}
      <Alert variant="warning" className="mt-6">
        <h3 className="font-medium mb-2">⚠️ 注意事項</h3>
        <ul className="text-sm space-y-1">
          <li>• この機能はテスト/デバッグ用です。本番環境での大量実行は避けてください。</li>
          <li>• 作成された提出は通常の提出と同様にジャッジキューに追加されます。</li>
          <li>• 最大100件まで同時に作成できます。</li>
          <li>• ワーカーの負荷状況はシステム状態画面で確認できます。</li>
        </ul>
      </Alert>
    </div>
  )
}
