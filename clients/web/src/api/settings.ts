import { apiRequest } from './client'
import type { EnvResponse, SaveResponse, UpdateEnvRequest } from './types'

const API_BASE = import.meta.env.VITE_API_URL || ''

export async function fetchEnv(): Promise<EnvResponse> {
  return apiRequest<EnvResponse>('/api/settings/env')
}

export async function saveEnv(data: UpdateEnvRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/settings/env', {
    method: 'POST',
    body: data,
  })
}

export async function downloadVaultBackup(): Promise<void> {
  const token = localStorage.getItem('notes_token')
  const person = localStorage.getItem('notes_person')

  const headers: Record<string, string> = {
    'Accept': 'application/zip',
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  if (person) {
    headers['X-Notes-Person'] = person
  }

  const response = await fetch(`${API_BASE}/api/settings/vault-backup`, {
    method: 'GET',
    headers,
  })

  if (!response.ok) {
    let detail = `Request failed with status ${response.status}`
    try {
      const body = await response.json() as { detail?: string }
      if (body.detail) {
        detail = body.detail
      }
    } catch {
      // ignore invalid JSON body
    }
    throw new Error(detail)
  }

  const blob = await response.blob()
  const disposition = response.headers.get('Content-Disposition') || ''
  const filenameMatch = disposition.match(/filename="([^"]+)"/)
  const filename = filenameMatch?.[1] || `${person || 'notes'}-vault-backup.zip`

  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(objectUrl)
}

export async function downloadApk(): Promise<void> {
  const token = localStorage.getItem('notes_token')
  const person = localStorage.getItem('notes_person')

  const headers: Record<string, string> = {
    'Accept': 'application/vnd.android.package-archive',
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  if (person) {
    headers['X-Notes-Person'] = person
  }

  const response = await fetch(`${API_BASE}/api/apk/download`, {
    method: 'GET',
    headers,
  })

  if (!response.ok) {
    let detail = `Request failed with status ${response.status}`
    try {
      const body = await response.json() as { detail?: string }
      if (body.detail) {
        detail = body.detail
      }
    } catch {
      // ignore invalid JSON body
    }
    throw new Error(detail)
  }

  const blob = await response.blob()
  const disposition = response.headers.get('Content-Disposition') || ''
  const filenameMatch = disposition.match(/filename="([^"]+)"/)
  const filename = filenameMatch?.[1] || 'app-debug.apk'

  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(objectUrl)
}
