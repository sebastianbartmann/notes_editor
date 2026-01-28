import { apiRequest } from './client'
import type { DailyResponse, SaveResponse, SaveDailyRequest, AppendRequest } from './types'

export async function fetchDaily(): Promise<DailyResponse> {
  return apiRequest<DailyResponse>('/api/daily')
}

export async function saveDaily(data: SaveDailyRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/save', {
    method: 'POST',
    body: data,
  })
}

export async function appendDaily(data: AppendRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/append', {
    method: 'POST',
    body: data,
  })
}

export async function clearPinned(): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/clear-pinned', {
    method: 'POST',
    body: {},
  })
}
