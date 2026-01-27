import { apiRequest } from './client'
import type {
  SleepTimesResponse,
  SaveResponse,
  AppendSleepRequest,
  DeleteSleepRequest,
} from './types'

export async function fetchSleepTimes(): Promise<SleepTimesResponse> {
  return apiRequest<SleepTimesResponse>('/api/sleep-times')
}

export async function appendSleepTime(data: AppendSleepRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/sleep-times/append', {
    method: 'POST',
    body: data,
  })
}

export async function deleteSleepTime(data: DeleteSleepRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/sleep-times/delete', {
    method: 'POST',
    body: data,
  })
}
