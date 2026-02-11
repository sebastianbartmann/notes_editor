import { apiRequest } from './client'

export interface GitStatusResponse {
  output: string
}

export interface GitActionResponse {
  success: boolean
  message: string
  output?: string
}

export async function fetchGitStatus(): Promise<GitStatusResponse> {
  return apiRequest<GitStatusResponse>('/api/git/status')
}

export async function gitCommit(): Promise<GitActionResponse> {
  return apiRequest<GitActionResponse>('/api/git/commit', { method: 'POST', body: {} })
}

export async function gitPush(): Promise<GitActionResponse> {
  return apiRequest<GitActionResponse>('/api/git/push', { method: 'POST', body: {} })
}

export async function gitPull(): Promise<GitActionResponse> {
  return apiRequest<GitActionResponse>('/api/git/pull', { method: 'POST', body: {} })
}

export async function gitCommitPush(): Promise<GitActionResponse> {
  return apiRequest<GitActionResponse>('/api/git/commit-push', { method: 'POST', body: {} })
}

