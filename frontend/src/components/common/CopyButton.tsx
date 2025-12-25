import { useState } from 'react'
import { Copy, Check } from 'lucide-react'

interface CopyButtonProps {
  text: string
  showLabel?: boolean
  className?: string
  appendNewline?: boolean
}

export function CopyButton({ text, showLabel = false, className = '', appendNewline = false }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    const textToCopy = appendNewline ? text + '\n' : text
    await navigator.clipboard.writeText(textToCopy)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <button
      onClick={handleCopy}
      className={`btn btn-ghost btn-sm ${className}`}
      title="クリップボードにコピー"
    >
      {copied ? <Check size={14} /> : <Copy size={14} />}
      {showLabel && (copied ? 'コピーしました' : 'コピー')}
    </button>
  )
}

