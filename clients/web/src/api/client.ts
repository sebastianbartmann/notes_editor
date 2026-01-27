const API_BASE = import.meta.env.VITE_API_URL || ''

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  headers?: Record<string, string>
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public detail?: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

function getToken(): string | null {
  return localStorage.getItem('notes_token')
}

function getPerson(): string | null {
  return localStorage.getItem('notes_person')
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const token = getToken()
  const person = getPerson()

  const headers: Record<string, string> = {
    'Accept': 'application/json',
    ...options.headers,
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  if (person) {
    headers['X-Notes-Person'] = person
  }

  if (options.body) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  })

  if (!response.ok) {
    let detail: string | undefined
    try {
      const errorData = await response.json()
      detail = errorData.detail
    } catch {
      // Ignore JSON parse errors
    }
    throw new ApiError(
      `Request failed with status ${response.status}`,
      response.status,
      detail
    )
  }

  return response.json()
}

export async function apiStreamRequest(
  endpoint: string,
  body: unknown
): Promise<ReadableStreamDefaultReader<Uint8Array>> {
  const token = getToken()
  const person = getPerson()

  const headers: Record<string, string> = {
    'Accept': 'application/x-ndjson',
    'Content-Type': 'application/json',
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  if (person) {
    headers['X-Notes-Person'] = person
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })

  if (!response.ok) {
    let detail: string | undefined
    try {
      const errorData = await response.json()
      detail = errorData.detail
    } catch {
      // Ignore JSON parse errors
    }
    throw new ApiError(
      `Request failed with status ${response.status}`,
      response.status,
      detail
    )
  }

  if (!response.body) {
    throw new ApiError('No response body', 0)
  }

  return response.body.getReader()
}
