import { apiRequest, apiStreamRequest } from './client'
import type {
  ChatResponse,
  ChatRequest,
  ClearSessionRequest,
  HistoryResponse,
  SaveResponse,
  StreamEvent,
} from './types'

export async function chat(data: ChatRequest): Promise<ChatResponse> {
  return apiRequest<ChatResponse>('/api/claude/chat', {
    method: 'POST',
    body: data,
  })
}

export async function* chatStream(
  data: ChatRequest
): AsyncGenerator<StreamEvent, void, unknown> {
  const reader = await apiStreamRequest('/api/claude/chat-stream', data)
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.trim()) {
          try {
            const event = JSON.parse(line) as StreamEvent
            yield event
          } catch {
            console.error('Failed to parse stream event:', line)
          }
        }
      }
    }

    // Process any remaining buffer
    if (buffer.trim()) {
      try {
        const event = JSON.parse(buffer) as StreamEvent
        yield event
      } catch {
        console.error('Failed to parse final stream event:', buffer)
      }
    }
  } finally {
    reader.releaseLock()
  }
}

export async function clearSession(data: ClearSessionRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/claude/clear', {
    method: 'POST',
    body: data,
  })
}

export async function getHistory(sessionId: string): Promise<HistoryResponse> {
  return apiRequest<HistoryResponse>(`/api/claude/history?session_id=${sessionId}`)
}
