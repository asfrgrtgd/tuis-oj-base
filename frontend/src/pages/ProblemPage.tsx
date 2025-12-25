import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Alert } from '@/components/ui/Alert'
import { BackLink, CopyButton } from '@/components/common'
import { formatTimeLimit, formatMemoryLimit } from '@/lib/utils'
import { CodeEditor } from '@/components/code/CodeEditor'
import type { Language, Problem, SubmitCodeRequest } from '@/types'
import { Clock, HardDrive, Send, Search } from 'lucide-react'

// シンプルなマークダウンパーサー
function SimpleMarkdown({ content }: { content: string }) {
  const parseMarkdown = (text: string): string => {
    let html = text
      // コードブロック
      .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
      // インラインコード
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      // 見出し
      .replace(/^### (.+)$/gm, '<h3>$1</h3>')
      .replace(/^## (.+)$/gm, '<h2>$1</h2>')
      .replace(/^# (.+)$/gm, '<h1>$1</h1>')
      // 太字
      .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
      // 斜体
      .replace(/\*([^*]+)\*/g, '<em>$1</em>')
      // リスト
      .replace(/^- (.+)$/gm, '<li>$1</li>')
      .replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>')
      // 改行
      .replace(/\n\n/g, '</p><p>')
      .replace(/\n/g, '<br/>')
    
    return `<p>${html}</p>`
      .replace(/<p><\/p>/g, '')
      .replace(/<p>(<h[123]>)/g, '$1')
      .replace(/(<\/h[123]>)<\/p>/g, '$1')
      .replace(/<p>(<pre>)/g, '$1')
      .replace(/(<\/pre>)<\/p>/g, '$1')
      .replace(/<p>(<ul>)/g, '$1')
      .replace(/(<\/ul>)<\/p>/g, '$1')
  }

  return (
    <div 
      className="markdown"
      dangerouslySetInnerHTML={{ __html: parseMarkdown(content) }}
    />
  )
}

const FALLBACK_LANGS: Language[] = [
  {
    key: 'c',
    label: 'C (GCC)',
    defaultSource: '#include <stdio.h>\nint main(){int a,b; if(scanf("%d %d",&a,&b)!=2) return 1; printf("%d\\n",a+b);}',
  },
  {
    key: 'cpp',
    label: 'C++17 (G++)',
    defaultSource: '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ios::sync_with_stdio(false);cin.tie(nullptr);long long a,b;if(!(cin>>a>>b)) return 0; cout<<a+b<<\"\\n\";}\n',
  },
  {
    key: 'python',
    label: 'Python 3',
    defaultSource: 'a,b = map(int, input().split())\nprint(a+b)',
  },
  {
    key: 'java',
    label: 'Java',
    defaultSource: 'import java.util.Scanner;\npublic class Main {\n    public static void main(String[] args) {\n        Scanner sc = new Scanner(System.in);\n        int a = sc.nextInt();\n        int b = sc.nextInt();\n        System.out.println(a + b);\n    }\n}',
  },
]

const LAST_LANGUAGE_STORAGE_KEY = 'preferred_language'

const getStoredLanguage = (): string | null => {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(LAST_LANGUAGE_STORAGE_KEY)
}

const getInitialLanguage = (): string => {
  const storedLanguage = getStoredLanguage()
  if (storedLanguage) {
    const fallbackLang = FALLBACK_LANGS.find((lang) => lang.key === storedLanguage)
    if (fallbackLang) return storedLanguage
  }
  return FALLBACK_LANGS[0].key
}

const getInitialSource = (): string => {
  const storedLanguage = getStoredLanguage()
  const initialLangMeta = FALLBACK_LANGS.find((lang) => lang.key === storedLanguage)
  if (initialLangMeta?.defaultSource) return initialLangMeta.defaultSource
  return FALLBACK_LANGS[0].defaultSource ?? ''
}

export function ProblemPage() {
  const params = useParams()
  const navigate = useNavigate()
  const problemId = Number(params.id)
  const [language, setLanguage] = useState<string>(getInitialLanguage)
  const [source, setSource] = useState<string>(getInitialSource)
  const initializedFromQuery = useRef(false)

  const problemQuery = useQuery<Problem>({
    queryKey: ['problem', problemId],
    queryFn: () => api.problems.get(problemId),
    enabled: Number.isFinite(problemId),
  })

  const languagesQuery = useQuery<Language[]>({
    queryKey: ['languages'],
    queryFn: () => api.submissions.languages(),
  })

  // 言語一覧取得後に初期値を反映
  useEffect(() => {
    const langs = languagesQuery.data
    if (initializedFromQuery.current) return
    if (!langs || langs.length === 0) return
    initializedFromQuery.current = true
    const storedLanguage = getStoredLanguage()
    const selectedLanguage =
      storedLanguage && langs.some((lang) => lang.key === storedLanguage)
        ? storedLanguage
        : langs[0].key
    localStorage.setItem(LAST_LANGUAGE_STORAGE_KEY, selectedLanguage)
    setLanguage(selectedLanguage)
    const meta = langs.find((lang) => lang.key === selectedLanguage)
    setSource(meta?.defaultSource ?? '')
  }, [languagesQuery.data])

  const submitMutation = useMutation({
    mutationFn: (payload: SubmitCodeRequest) => api.submissions.submit(payload),
    onSuccess: (res) => {
      // 提出成功後、提出詳細ページへ自動遷移
      navigate(`/submissions/${res.id}`)
    },
  })
  const submitError =
    (submitMutation.error &&
      typeof submitMutation.error === 'object' &&
      'response' in submitMutation.error &&
      (submitMutation.error as any)?.response?.data?.error?.message) ||
    ''

  const problem: Problem | undefined = problemQuery.data
  const languages: Language[] = languagesQuery.data ?? FALLBACK_LANGS

  const applyLanguageDefault = (key: string) => {
    setLanguage(key)
    localStorage.setItem(LAST_LANGUAGE_STORAGE_KEY, key)
    const meta = languages.find((l) => l.key === key)
    if (meta?.defaultSource) {
      setSource(meta.defaultSource)
    }
  }

  if (problemQuery.isLoading) {
    return (
      <div className="py-8">
        <div className="skeleton h-8 w-48 mb-4" />
        <div className="skeleton h-64 w-full" />
      </div>
    )
  }

  if (!problem) {
    return (
      <div className="py-8">
        <div className="card">
          <div className="empty-state">
            <Search size={48} className="text-muted opacity-50 mb-4" />
            <h2 className="empty-state-title">問題が見つかりません</h2>
            <p className="empty-state-description">
              指定された問題は存在しないか、公開されていません
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
    <div className="py-8 max-w-5xl mx-auto space-y-6">
      {/* ヘッダー */}
      <div className="mb-6">
        <div className="mb-4">
          <BackLink to="/problems">問題一覧に戻る</BackLink>
        </div>
        <h1 className="page-title mb-2">{problem.title}</h1>
        <div className="stats-row">
          <div className="stat-item">
            <span className="stat-label flex items-center gap-1">
              <Clock size={12} /> 実行時間制限
            </span>
            <span className="stat-value mono">{formatTimeLimit(problem.time_limit_ms)}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label flex items-center gap-1">
              <HardDrive size={12} /> メモリ制限
            </span>
            <span className="stat-value mono">{formatMemoryLimit(problem.memory_limit_kb)}</span>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        {/* 問題文 */}
        <div className="space-y-3">
          <div className="card">
            <div className="card-body">
              <SimpleMarkdown content={problem.statement} />
            </div>
          </div>

          {/* サンプルケース */}
          {problem.samples && problem.samples.length > 0 && (
            <div className="card">
              <div className="card-header">
                <h2 className="font-semibold">入出力例</h2>
              </div>
              <div className="card-body">
                {problem.samples.map((sample, idx) => (
                  <div
                    key={idx}
                    className={`grid md:grid-cols-2 gap-3 md:gap-6 ${idx > 0 ? 'mt-6' : ''}`}
                  >
                    <div className="sample-case">
                      <div className="sample-case-header">
                        <span>入力例 {idx + 1}</span>
                        <CopyButton text={sample.input} appendNewline={true} />
                      </div>
                      <div className="sample-case-content">{sample.input || '（なし）'}</div>
                    </div>
                    <div className="sample-case">
                      <div className="sample-case-header">
                        <span>出力例 {idx + 1}</span>
                        <CopyButton text={sample.output} />
                      </div>
                      <div className="sample-case-content">{sample.output}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* 提出フォーム */}
        <div className="space-y-3">
          <div className="card">
            <div className="card-header">
              <h2 className="font-semibold">提出</h2>
            </div>
            <div className="card-body">
              <div className="form-group">
                <label htmlFor="language" className="label">言語</label>
                <select
                  id="language"
                  className="input"
                  value={language}
                  onChange={(e) => applyLanguageDefault(e.target.value)}
                >
                  {languages.map((lang) => (
                    <option value={lang.key} key={lang.key}>
                      {lang.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label htmlFor="source" className="label">ソースコード</label>
                <div className="border border-border rounded-md overflow-hidden">
                  <CodeEditor
                    value={source}
                    language={language || 'plaintext'}
                    onChange={(val) => setSource(val)}
                    height={360}
                  />
                </div>
              </div>

              <button
                onClick={() =>
                  submitMutation.mutate({
                    problem_id: problemId,
                    language,
                    source_code: source,
                  })
                }
                disabled={submitMutation.isPending || !problemId}
                className="btn btn-primary w-full"
              >
                {submitMutation.isPending ? (
                  <>
                    <span className="loading-spinner" />
                    送信中...
                  </>
                ) : (
                  <>
                    <Send size={16} />
                    提出する
                  </>
                )}
              </button>

              {submitMutation.isError && (
                <Alert variant="error" className="mt-3">
                  提出に失敗しました。{submitError || 'もう一度お試しください。'}
                </Alert>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
