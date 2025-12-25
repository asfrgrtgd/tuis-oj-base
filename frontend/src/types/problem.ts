export interface Problem {
  id: number
  title: string
  slug: string
  statement: string
  samples: SampleCase[]
  time_limit_ms: number
  memory_limit_kb: number
  is_public: boolean
  created_at?: string
  updated_at?: string
  solved_count?: number
  submission_count?: number
  visibility?: 'public' | 'hidden'
}

export interface SampleCase {
  input: string
  output: string
}

export interface ProblemsResponse {
  problems: Problem[]
}

export interface ProblemStats {
  problem_id: number
  title: string
  submission_count: number
  accepted_count: number
  unique_users: number
  unique_accepted_users: number
  acceptance_rate: number
  last_submission_at?: string
  status_breakdown: Record<string, number>
}

export interface AdminProblemsResponse {
  items: Problem[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}
