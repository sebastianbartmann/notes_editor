import { apiRequest } from './client'

export interface SyncStatus {
  in_progress: boolean
  pending_pull: boolean
  pending_push: boolean
  last_pull_at?: string
  last_push_at?: string
  last_error?: string
  last_error_at?: string
}

export async function fetchSyncStatus(): Promise<SyncStatus> {
  return apiRequest<SyncStatus>('/api/sync/status')
}

export async function syncNow(params: {
  wait: boolean
  timeoutMs?: number
}): Promise<SyncStatus> {
  return apiRequest<SyncStatus>('/api/sync', {
    method: 'POST',
    body: {
      wait: params.wait,
      timeout_ms: params.timeoutMs ?? 0,
    },
  })
}

export async function syncIfStale(params: {
  maxAgeMs: number
  timeoutMs: number
}): Promise<void> {
  try {
    const status = await fetchSyncStatus()
    if (status.last_pull_at) {
      const t = Date.parse(status.last_pull_at)
      if (!Number.isNaN(t) && Date.now() - t < params.maxAgeMs) {
        return
      }
    }
  } catch {
    // Ignore status failures and attempt a sync anyway.
  }

  try {
    await syncNow({ wait: true, timeoutMs: params.timeoutMs })
  } catch {
    // Best-effort: don't block the UI if sync fails/timeouts.
  }
}

