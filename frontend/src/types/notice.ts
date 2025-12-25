export interface Notice {
  id: number
  title: string
  body: string
  created_at: string
  updated_at: string
}

export interface NoticeListResponse {
  items: Notice[]
  page: number
  per_page: number
  total_items: number
  total_pages: number
}

