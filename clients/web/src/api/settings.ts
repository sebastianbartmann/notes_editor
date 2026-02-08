import { apiRequest } from './client'
import type { EnvResponse, SaveResponse, UpdateEnvRequest } from './types'

export async function fetchEnv(): Promise<EnvResponse> {
  return apiRequest<EnvResponse>('/api/settings/env')
}

export async function saveEnv(data: UpdateEnvRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/settings/env', {
    method: 'POST',
    body: data,
  })
}
