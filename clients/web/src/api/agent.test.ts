import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  agentChatStream,
  clearAgentSession,
  getAgentConfig,
  getAgentGatewayHealth,
  listAgentActions,
} from './agent'
import type { AgentStreamEvent } from './types'

function createMockStream(events: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  let index = 0

  return new ReadableStream({
    pull(controller) {
      if (index < events.length) {
        controller.enqueue(encoder.encode(events[index]))
        index++
      } else {
        controller.close()
      }
    },
  })
}

function mockFetchResponse(
  stream: ReadableStream<Uint8Array>,
  status = 200
): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    body: stream,
    json: async () => ({}),
  } as Response
}

describe('agentChatStream', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('parses v2 start/tool_call/tool_result/done events', async () => {
    const events = [
      '{"type":"start","session_id":"s1","run_id":"r1"}\n',
      '{"type":"tool_call","tool":"read_file","args":{"path":"notes.md"},"run_id":"r1"}\n',
      '{"type":"tool_result","tool":"read_file","ok":true,"summary":"Tool read_file executed","run_id":"r1"}\n',
      '{"type":"done","session_id":"s1","run_id":"r1"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: AgentStreamEvent[] = []
    for await (const event of agentChatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(4)
    expect(collected[0]).toEqual({ type: 'start', session_id: 's1', run_id: 'r1' })
    expect(collected[1]).toEqual({
      type: 'tool_call',
      tool: 'read_file',
      args: { path: 'notes.md' },
      run_id: 'r1',
    })
    expect(collected[2]).toEqual({
      type: 'tool_result',
      tool: 'read_file',
      ok: true,
      summary: 'Tool read_file executed',
      run_id: 'r1',
    })
  })

  it('handles split chunks', async () => {
    const events = [
      '{"type":"text","delta":"Hel',
      'lo","run_id":"r1"}\n{"type":"done","session_id":"s1","run_id":"r1"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: AgentStreamEvent[] = []
    for await (const event of agentChatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toEqual([
      { type: 'text', delta: 'Hello', run_id: 'r1' },
      { type: 'done', session_id: 's1', run_id: 'r1' },
    ])
  })
})

describe('clearAgentSession', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('posts to /api/agent/session/clear', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true, message: 'Session cleared' }),
    } as Response)

    await clearAgentSession('session-123')

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/agent/session/clear',
      expect.objectContaining({
        method: 'POST',
      })
    )
  })
})

describe('agent config/actions APIs', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('loads agent config', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        runtime_mode: 'gateway_subscription',
        prompt_path: 'agents.md',
        actions_path: 'agent/actions',
        prompt: 'test prompt',
      }),
    } as Response)

    const cfg = await getAgentConfig()
    expect(cfg.prompt).toBe('test prompt')
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/agent/config',
      expect.objectContaining({ method: 'GET' })
    )
  })

  it('loads actions list', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        actions: [{ id: 'a1', label: 'A1', path: 'agent/actions/A1.md', metadata: { requires_confirmation: false } }],
      }),
    } as Response)

    const actions = await listAgentActions()
    expect(actions).toHaveLength(1)
    expect(actions[0].id).toBe('a1')
  })

  it('loads gateway health', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        url: 'http://127.0.0.1:4317',
        configured: true,
        reachable: true,
        healthy: true,
        mode: 'claude_cli',
      }),
    } as Response)

    const health = await getAgentGatewayHealth()
    expect(health.healthy).toBe(true)
    expect(health.mode).toBe('claude_cli')
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/agent/gateway/health',
      expect.objectContaining({ method: 'GET' })
    )
  })
})
