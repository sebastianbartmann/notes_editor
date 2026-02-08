import { apiRequest, apiStreamRequest } from './client'
import type {
  AgentAction,
  AgentActionsResponse,
  AgentChatRequest,
  AgentChatResponse,
  AgentConfig,
  AgentConfigUpdate,
  AgentGatewayHealth,
  AgentStreamEvent,
  SaveResponse,
} from './types'

export async function agentChat(data: AgentChatRequest): Promise<AgentChatResponse> {
  return apiRequest<AgentChatResponse>('/api/agent/chat', {
    method: 'POST',
    body: data,
  })
}

export async function* agentChatStream(
  data: AgentChatRequest
): AsyncGenerator<AgentStreamEvent, void, unknown> {
  const reader = await apiStreamRequest('/api/agent/chat-stream', data)
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
            const event = JSON.parse(line) as AgentStreamEvent
            yield event
          } catch {
            console.error('Failed to parse agent stream event:', line)
          }
        }
      }
    }

    if (buffer.trim()) {
      try {
        const event = JSON.parse(buffer) as AgentStreamEvent
        yield event
      } catch {
        console.error('Failed to parse final agent stream event:', buffer)
      }
    }
  } finally {
    reader.releaseLock()
  }
}

export async function clearAgentSession(sessionId: string): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/agent/session/clear', {
    method: 'POST',
    body: { session_id: sessionId },
  })
}

export async function getAgentConfig(): Promise<AgentConfig> {
  return apiRequest<AgentConfig>('/api/agent/config')
}

export async function saveAgentConfig(data: AgentConfigUpdate): Promise<AgentConfig> {
  return apiRequest<AgentConfig>('/api/agent/config', {
    method: 'POST',
    body: data,
  })
}

export async function listAgentActions(): Promise<AgentAction[]> {
  const resp = await apiRequest<AgentActionsResponse>('/api/agent/actions')
  return resp.actions
}

export async function runAgentAction(
  actionId: string,
  payload: { session_id?: string; message?: string; confirm?: boolean } = {}
): Promise<AgentChatResponse> {
  return apiRequest<AgentChatResponse>(`/api/agent/actions/${actionId}/run`, {
    method: 'POST',
    body: payload,
  })
}

export async function getAgentGatewayHealth(): Promise<AgentGatewayHealth> {
  return apiRequest<AgentGatewayHealth>('/api/agent/gateway/health')
}
