import { format, formatDistanceToNow } from 'date-fns'
import ja from 'date-fns/locale/ja'

/**
 * 日付を標準フォーマットで表示
 * 例: 2025年1月15日 14:30
 */
export function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy年M月d日 HH:mm', { locale: ja })
  } catch {
    return dateStr
  }
}

/**
 * 日付を短縮フォーマットで表示
 * 例: 2025/01/15 14:30
 */
export function formatDateShort(dateStr?: string): string {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy/MM/dd HH:mm', { locale: ja })
  } catch {
    return dateStr
  }
}

/**
 * 日付を秒まで含めたフォーマットで表示
 * 例: 2025/01/15 14:30:45
 */
export function formatDateWithSeconds(dateStr?: string): string {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy/MM/dd HH:mm:ss', { locale: ja })
  } catch {
    return dateStr
  }
}

/**
 * 日付のみを表示
 * 例: 2025年1月15日
 */
export function formatDateOnly(dateStr?: string): string {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy年M月d日', { locale: ja })
  } catch {
    return dateStr
  }
}

/**
 * 相対時間で表示
 * 例: 3分前
 */
export function formatRelativeTime(dateStr: string): string {
  try {
    return formatDistanceToNow(new Date(dateStr), {
      addSuffix: true,
      locale: ja as any,
    } as any)
  } catch {
    return dateStr
  }
}

/**
 * 時間制限を整形
 * 例: 2 sec, 500 ms
 */
export function formatTimeLimit(ms: number): string {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(ms % 1000 === 0 ? 0 : 1)} sec`
  }
  return `${ms} ms`
}

/**
 * メモリ制限を整形
 * 例: 256 MB, 512 KB
 */
export function formatMemoryLimit(kb: number): string {
  if (kb >= 1024) {
    return `${Math.round(kb / 1024)} MB`
  }
  return `${kb} KB`
}
