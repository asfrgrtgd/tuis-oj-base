import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

interface BackLinkProps {
  to: string
  children: React.ReactNode
}

export function BackLink({ to, children }: BackLinkProps) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-1 text-sm text-muted hover:text-foreground"
    >
      <ArrowLeft size={16} />
      {children}
    </Link>
  )
}

