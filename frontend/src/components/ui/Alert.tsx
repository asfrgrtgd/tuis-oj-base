import { AlertCircle, CheckCircle, Info, AlertTriangle } from 'lucide-react'

type AlertVariant = 'error' | 'success' | 'warning' | 'info'

interface AlertProps {
  variant?: AlertVariant
  children: React.ReactNode
  className?: string
}

const variantStyles: Record<AlertVariant, { bg: string; text: string; icon: typeof AlertCircle }> = {
  error: {
    bg: 'bg-red-50 border-red-200',
    text: 'text-red-700',
    icon: AlertCircle,
  },
  success: {
    bg: 'bg-green-50 border-green-200',
    text: 'text-green-700',
    icon: CheckCircle,
  },
  warning: {
    bg: 'bg-yellow-50 border-yellow-200',
    text: 'text-yellow-700',
    icon: AlertTriangle,
  },
  info: {
    bg: 'bg-blue-50 border-blue-200',
    text: 'text-blue-700',
    icon: Info,
  },
}

export function Alert({ variant = 'info', children, className = '' }: AlertProps) {
  const styles = variantStyles[variant]
  const Icon = styles.icon

  return (
    <div className={`px-4 py-3 rounded-lg border flex items-start gap-3 ${styles.bg} ${styles.text} ${className}`}>
      <Icon size={18} className="flex-shrink-0 mt-0.5" />
      <div className="text-sm">{children}</div>
    </div>
  )
}

