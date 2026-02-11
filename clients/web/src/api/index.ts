import { apiRequest } from './client'

export interface IndexStatus {
  in_progress: boolean
  pending: boolean
  last_reason?: string
  last_started_at?: string
  last_success_at?: string
  last_error?: string
  last_error_at?: string
}

export async function fetchIndexStatus(): Promise<IndexStatus> {
  return apiRequest<IndexStatus>('/api/sync/index-status')
}

