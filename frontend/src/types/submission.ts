export interface Submission {
  id: number
  userid: string
  problem_id: number
  problem_title?: string
  language: string
  status: string
  verdict?: string
  time_ms?: number
  memory_kb?: number
  created_at: string
  updated_at: string
  exit_code?: number
  error_message?: string
  source_code?: string
  judge_details?: JudgeDetail[]
}

export interface JudgeDetail {
  testcase: string
  status: string
  time_ms?: number
  memory_kb?: number
}

export interface SubmissionsResponse {
  items: Submission[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

export interface SubmitCodeRequest {
  problem_id: number
  language: string
  source_code: string
}

export interface SubmitCodeResponse {
  id: number
  problem_id: number
  language: string
  status: string
  created_at: string
}

export interface Language {
  key: string
  label: string
  syntax?: string
  defaultSource?: string
  defaultStdin?: string
}

export interface LanguagesResponse {
  languages: Language[]
}

export type SubmissionStatus =
  | 'pending'
  | 'judging'
  | 'succeeded'
  | 'failed'
  | 'ac'
  | 'wa'
  | 'tle'
  | 'mle'
  | 're'
  | 'ce'
