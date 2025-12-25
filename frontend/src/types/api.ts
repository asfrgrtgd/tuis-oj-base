export interface ApiError {
  error: {
    code: string
    message: string
  }
}

export interface PaginationParams {
  page?: number
  per_page?: number
}

export interface PaginatedResponse<T> {
  items: T[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}
