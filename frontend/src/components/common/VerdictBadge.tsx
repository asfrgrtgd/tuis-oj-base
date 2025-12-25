interface VerdictBadgeProps {
  verdict?: string
  status?: string
  size?: 'normal' | 'large'
}

const verdictClasses: Record<string, string> = {
  AC: 'verdict verdict-ac',
  WA: 'verdict verdict-wa',
  TLE: 'verdict verdict-tle',
  MLE: 'verdict verdict-mle',
  RE: 'verdict verdict-re',
  CE: 'verdict verdict-ce',
  SE: 'verdict verdict-se',
  pending: 'verdict verdict-pending',
  running: 'verdict verdict-pending',
}

export function VerdictBadge({ verdict, status, size = 'normal' }: VerdictBadgeProps) {
  const v = verdict ?? status ?? 'pending'
  const sizeClass = size === 'large' ? 'text-lg px-4 py-2' : ''

  return (
    <span className={`${verdictClasses[v] || 'verdict verdict-pending'} ${sizeClass}`}>
      {v}
    </span>
  )
}

