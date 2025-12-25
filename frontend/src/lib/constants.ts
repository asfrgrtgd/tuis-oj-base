/**
 * APIベースURL
 */
export const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'

/**
 * クエリのデフォルト設定
 */
export const QUERY_STALE_TIME = 30000 // 30秒
export const QUERY_CACHE_TIME = 300000 // 5分
