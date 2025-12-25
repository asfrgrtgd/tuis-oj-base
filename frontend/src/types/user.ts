export interface User {
  userid: string
  role: 'user' | 'admin'
  problem_solved_count?: number
  submission_count?: number
  created_at?: string
}

export interface LoginRequest {
  userid: string
  password: string
}

export interface LoginResponse {
  user: User
}

export interface AdminUser {
  id: number
  userid: string
  role: 'user' | 'admin'
  created_at: string
}

export interface AdminUsersResponse {
  items: AdminUser[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

export interface UserProfile {
  userid: string
  solved_count: number
  submission_count: number
  created_at: string
}
