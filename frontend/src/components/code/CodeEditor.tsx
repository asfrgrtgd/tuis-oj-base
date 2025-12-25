import { lazy, Suspense } from 'react'

const MonacoEditor = lazy(() => import('@monaco-editor/react'))

type Props = {
  value: string
  language?: string
  onChange?: (value: string) => void
  height?: number
  readOnly?: boolean
}

export function CodeEditor({
  value,
  language = 'plaintext',
  onChange,
  height = 360,
  readOnly = false,
}: Props) {
  // SSR や window 未定義環境では textarea にフォールバック
  if (typeof window === 'undefined') {
    return (
      <textarea
        className="input mono text-sm w-full"
        rows={height / 24}
        value={value}
        readOnly={readOnly}
        onChange={(e) => onChange?.(e.target.value)}
      />
    )
  }

  return (
    <Suspense fallback={<div className="skeleton h-48 w-full" />}>
      <MonacoEditor
        height={height}
        language={language || 'plaintext'}
        theme="vs"
        value={value}
        onChange={(val) => onChange?.(val ?? '')}
        options={{
          readOnly,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          wordWrap: 'on',
          automaticLayout: true,
        }}
      />
    </Suspense>
  )
}
