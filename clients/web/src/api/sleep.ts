import { apiRequest } from './client'
import type {
  SleepTimesResponse,
  SaveResponse,
  AppendSleepRequest,
  DeleteSleepRequest,
  UpdateSleepRequest,
  SleepSummaryResponse,
  ExportSleepResponse,
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

export async function updateSleepTime(data: UpdateSleepRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/sleep-times/update', {
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

export async function fetchSleepSummary(): Promise<SleepSummaryResponse> {
  return apiRequest<SleepSummaryResponse>('/api/sleep-times/summary')
}

export async function exportSleepMarkdown(): Promise<ExportSleepResponse> {
  return apiRequest<ExportSleepResponse>('/api/sleep-times/export-markdown', {
    method: 'POST',
    body: {},
  })
}
