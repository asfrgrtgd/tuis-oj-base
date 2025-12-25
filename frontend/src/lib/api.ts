import axios, { type AxiosError, type AxiosResponse } from 'axios'
import {
  type User,
  type LoginRequest,
  type LoginResponse,
  type Problem,
  type Submission,
  type SubmissionsResponse,
  type SubmitCodeRequest,
  type SubmitCodeResponse,
  type LanguagesResponse,
  type ProblemStats,
  type ApiError,
  type Language,
  type AdminUser,
  type AdminUsersResponse,
  type QueueDepth,
  type UserProfile,
} from '@/types'
import { API_BASE } from '@/lib/constants'

const SAFE_METHODS = ['GET', 'HEAD', 'OPTIONS', 'TRACE']
let csrfToken = ''

export const apiClient = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
})

// CSRF トークンをレスポンスから拾う
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    const token = response.headers['x-csrf-token']
    if (token) csrfToken = token
    return response
  },
  (error: AxiosError<ApiError>) => {
    const token = error.response?.headers?.['x-csrf-token']
    if (token) csrfToken = token
    return Promise.reject(error)
  }
)

// CSRF トークンをリクエストに付与
apiClient.interceptors.request.use((config) => {
  const method = config.method?.toUpperCase()
  if (method && !SAFE_METHODS.includes(method) && csrfToken) {
    config.headers['X-CSRF-Token'] = csrfToken
  }
  return config
})

export async function initCsrf(): Promise<void> {
  if (csrfToken) return
  try {
    await apiClient.get('/problems', { params: { page: 1, per_page: 1 } })
  } catch {
    // トークン取得目的なのでエラーは握りつぶす
  }
}

// ---------- 正規化ユーティリティ ----------

function normalizeProblemList(data: any): Problem[] {
  const src = (() => {
    if (!data) return []
    if (Array.isArray(data)) return data
    if (Array.isArray(data.items)) return data.items
    if (Array.isArray(data.problems)) return data.problems
    return []
  })()
  return src.map((p: any) => normalizeProblem(p))
}

function normalizeProblem(data: any): Problem {
  const sampleSrc =
    data.samples || data.sample_cases || data.sampleCases || data.sample_cases || []
  const samples = Array.isArray(sampleSrc)
    ? sampleSrc.map((s: any) => ({
        input: s.input ?? s.stdin ?? '',
        output: s.output ?? '',
        description: s.description ?? '',
      }))
    : []

  return {
    id: Number(data.id ?? data.problem_id ?? 0),
    title: data.title ?? 'Problem',
    slug: data.slug ?? '',
    statement: data.statement ?? data.statement_md ?? data.statementMarkdown ?? '',
    samples,
  time_limit_ms: data.time_limit_ms ?? data.time_limit ?? data.limits?.time_ms ?? 0,
  memory_limit_kb:
    data.memory_limit_kb ?? data.memory_limit ?? data.limits?.memory_kb ?? 0,
  is_public: data.is_public ?? (data.visibility ? data.visibility === 'public' : true),
  created_at: data.created_at,
  solved_count: data.solved_count ?? data.accepted_count,
  submission_count: data.submission_count,
}
}

function normalizeSubmissions(data: any): SubmissionsResponse {
  if (Array.isArray(data?.items)) return data as SubmissionsResponse
  if (Array.isArray(data)) {
    return {
      items: data as Submission[],
      page: 1,
      per_page: data.length,
      total_items: data.length,
      total_pages: 1,
    }
  }
  if (Array.isArray(data?.submissions)) {
    return {
      items: data.submissions,
      page: data.page ?? 1,
      per_page: data.per_page ?? data.perPage ?? data.submissions.length,
      total_items: data.total_items ?? data.submissions.length,
      total_pages: data.total_pages ?? 1,
    }
  }
  return {
    items: [],
    page: 1,
    per_page: 20,
    total_items: 0,
    total_pages: 0,
  }
}

// ---------- 認証 ----------

const authApi = {
  login: async (payload: LoginRequest): Promise<LoginResponse> => {
    const res = await apiClient.post<LoginResponse>('/auth/login', payload)
    return res.data
  },
  logout: async (): Promise<void> => {
    await initCsrf()
    await apiClient.post('/auth/logout')
  },
  me: async (): Promise<User | null> => {
    await initCsrf()
    try {
      const res = await apiClient.get<User>('/users/me')
      return res.data
    } catch (err) {
      const axiosErr = err as AxiosError
      if (axiosErr.response?.status === 401) return null
      throw err
    }
  },
}

// ---------- 問題 ----------

const problemsApi = {
  list: async (): Promise<Problem[]> => {
    const res = await apiClient.get('/problems')
    return normalizeProblemList(res.data)
  },
  get: async (id: number): Promise<Problem> => {
    const res = await apiClient.get(`/problems/${id}`)
    return normalizeProblem(res.data)
  },
  submissions: async (
    id: number,
    page = 1,
    perPage = 20
  ): Promise<SubmissionsResponse> => {
    const res = await apiClient.get(`/problems/${id}/submissions`, {
      params: { page, per_page: perPage },
    })
    return normalizeSubmissions(res.data)
  },
}

// ---------- 提出 ----------

const submissionsApi = {
  languages: async (): Promise<Language[]> => {
    const res = await apiClient.get<LanguagesResponse>('/languages')
    return res.data.languages ?? []
  },
  submit: async (payload: SubmitCodeRequest): Promise<SubmitCodeResponse> => {
    await initCsrf()
    const res = await apiClient.post<SubmitCodeResponse>('/submissions', payload)
    return res.data
  },
  mine: async (
    page = 1,
    perPage = 20,
    problemId?: number
  ): Promise<SubmissionsResponse> => {
    const res = await apiClient.get('/submissions', {
      params: { page, per_page: perPage, ...(problemId ? { problem_id: problemId } : {}) },
    })
    return normalizeSubmissions(res.data)
  },
  detail: async (id: number): Promise<Submission> => {
    const res = await apiClient.get<Submission>(`/submissions/${id}`)
    return res.data
  },
}

// ---------- ユーザー ----------

const usersApi = {
  profile: async (userid: string): Promise<UserProfile> => {
    const res = await apiClient.get<UserProfile>(`/users/${userid}`)
    return res.data
  },
}

// ---------- お知らせ ----------

interface Notice {
  id: number
  title: string
  body: string
  created_at: string
  updated_at: string
}

interface NoticesResponse {
  items: Notice[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

const noticesApi = {
  list: async (page = 1, perPage = 50): Promise<NoticesResponse> => {
    const res = await apiClient.get<NoticesResponse>('/notices', {
      params: { page, per_page: perPage },
    })
    return res.data
  },
}

// ---------- キュー / 管理 ----------

const miscApi = {
  queue: async (): Promise<QueueDepth> => {
    const res = await apiClient.get<QueueDepth>('/queue')
    return res.data
  },
}


interface SystemStatus {
  queue: {
    pending: number
    processing: number
  }
  workers?: {
    active: number
    total: number
  }
  memory?: {
    used_bytes: number
    total_bytes: number
  }
  judge?: {
    status: string
    version?: string
  }
  uptime_seconds?: number
}

interface MetricsOverview {
  queues: {
    pending: number
    processing: number
    expired_candidate?: number
  }
  workers: {
    worker_id: string
    hostname: string
    pid: number
    concurrency: number
    uptime_seconds: number
    status: string
    running_count: number
    current_job?: string
    running_jobs?: string[]
    processed_total: number
    failed_total: number
    last_error?: string
    memory_rss_bytes?: number
    num_goroutine?: number
    started_at: string
    updated_at: string
  }[]
}

interface AdminProblem {
  id: number
  title: string
  slug: string
  visibility: 'public' | 'hidden'
  solved_count: number
  submission_count: number
}

interface AdminProblemsResponse {
  items: AdminProblem[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

interface BulkCreateUsersResult {
  created_count: number
  failed_count: number
  failed_rows: Array<{
    row_number: number
    userid: string
    reason: string
  }>
}

interface BulkSubmitRequest {
  problem_id: number
  language: string
  count: number
  source_code: string
}

interface BulkSubmitResponse {
  created: number[]
  count: number
  problem: number
  language: string
}

const adminApi = {
  users: async (page = 1, perPage = 20): Promise<AdminUsersResponse> => {
    const res = await apiClient.get<AdminUsersResponse>('/admin/users', {
      params: { page, per_page: perPage },
    })
    return res.data
  },
  createUser: async (payload: { userid: string; password: string; role: string }): Promise<AdminUser> => {
    await initCsrf()
    const res = await apiClient.post<AdminUser>('/admin/users', payload)
    return res.data
  },
  bulkCreateUsers: async (file: File): Promise<BulkCreateUsersResult> => {
    await initCsrf()
    const form = new FormData()
    form.append('file', file)
    const res = await apiClient.post<BulkCreateUsersResult>('/admin/users/bulk', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return res.data
  },
  // お知らせ管理
  notices: async (page = 1, perPage = 100): Promise<NoticesResponse> => {
    const res = await apiClient.get<NoticesResponse>('/admin/notices', {
      params: { page, per_page: perPage },
    })
    return res.data
  },
  createNotice: async (payload: { title: string; body: string }): Promise<Notice> => {
    await initCsrf()
    const res = await apiClient.post<Notice>('/admin/notices', payload)
    return res.data
  },
  updateNotice: async (id: number, payload: { title: string; body: string }): Promise<Notice> => {
    await initCsrf()
    const res = await apiClient.patch<Notice>(`/admin/notices/${id}`, payload)
    return res.data
  },
  deleteNotice: async (id: number): Promise<void> => {
    await initCsrf()
    await apiClient.delete(`/admin/notices/${id}`)
  },
  // 問題管理
  problems: async (page = 1, perPage = 100): Promise<AdminProblemsResponse> => {
    const res = await apiClient.get<AdminProblemsResponse>('/admin/problems', {
      params: { page, per_page: perPage },
    })
    return res.data
  },
  updateProblemVisibility: async (id: number, isPublic: boolean) => {
    await initCsrf()
    const res = await apiClient.patch(`/admin/problems/${id}`, { is_public: isPublic })
    return res.data
  },
  importProblem: async (file: File) => {
    await initCsrf()
    const form = new FormData()
    form.append('file', file)
    const res = await apiClient.post('/admin/problems/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return res.data
  },
  downloadTemplate: async (): Promise<Blob> => {
    await initCsrf()
    const res = await apiClient.get('/admin/problems/template', { responseType: 'blob' })
    return res.data
  },
  problemStats: async (id: number): Promise<ProblemStats> => {
    const res = await apiClient.get<ProblemStats>(`/admin/problems/${id}/stats`)
    return res.data
  },
  // 提出テスト
  bulkSubmit: async (payload: BulkSubmitRequest): Promise<BulkSubmitResponse> => {
    await initCsrf()
    const res = await apiClient.post<BulkSubmitResponse>('/admin/submissions/bulk_test', payload)
    return res.data
  },
  // システム状態
  systemStatus: async (): Promise<SystemStatus> => {
    const res = await apiClient.get<SystemStatus>('/admin/system/status')
    return res.data
  },
  metricsOverview: async (): Promise<MetricsOverview> => {
    const res = await apiClient.get<MetricsOverview>('/admin/metrics/overview')
    return res.data
  },
}

export const api = {
  auth: authApi,
  problems: problemsApi,
  submissions: submissionsApi,
  users: usersApi,
  notices: noticesApi,
  misc: miscApi,
  admin: adminApi,
  initCsrf,
}
