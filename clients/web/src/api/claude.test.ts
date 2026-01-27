import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { chatStream, chat, clearSession, getHistory } from './claude'
import type { StreamEvent } from './types'

// Helper to create a mock ReadableStream from NDJSON events
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

// Helper to create a mock fetch response
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

describe('chatStream', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('parses text events with delta field', async () => {
    const events = [
      '{"type":"text","delta":"Hello"}\n',
      '{"type":"text","delta":" world"}\n',
      '{"type":"done","session_id":"abc123"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'Hi' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(3)
    expect(collected[0]).toEqual({ type: 'text', delta: 'Hello' })
    expect(collected[1]).toEqual({ type: 'text', delta: ' world' })
    expect(collected[2]).toEqual({ type: 'done', session_id: 'abc123' })
  })

  it('parses tool_use events', async () => {
    const events = [
      '{"type":"tool_use","name":"Read","input":{"path":"test.md"}}\n',
      '{"type":"text","delta":"File contents..."}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'Read file' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(2)
    expect(collected[0]).toEqual({
      type: 'tool_use',
      name: 'Read',
      input: { path: 'test.md' },
    })
  })

  it('parses session events', async () => {
    const events = ['{"type":"session","session_id":"new-session-123"}\n']

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected[0]).toEqual({
      type: 'session',
      session_id: 'new-session-123',
    })
  })

  it('parses ping events (keepalive)', async () => {
    const events = [
      '{"type":"text","delta":"Starting..."}\n',
      '{"type":"ping"}\n',
      '{"type":"text","delta":"...done"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(3)
    expect(collected[1]).toEqual({ type: 'ping' })
  })

  it('parses error events', async () => {
    const events = [
      '{"type":"error","message":"Rate limit exceeded"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected[0]).toEqual({
      type: 'error',
      message: 'Rate limit exceeded',
    })
  })

  it('handles chunked data spanning multiple reads', async () => {
    // Simulate data split across chunks (partial JSON lines)
    const events = [
      '{"type":"text","delta":"Hel',  // Partial first event
      'lo"}\n{"type":"text"',          // End of first + partial second
      ',"delta":" world"}\n',          // End of second
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'Hi' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(2)
    expect(collected[0]).toEqual({ type: 'text', delta: 'Hello' })
    expect(collected[1]).toEqual({ type: 'text', delta: ' world' })
  })

  it('handles final buffer content without trailing newline', async () => {
    const events = [
      '{"type":"text","delta":"Hello"}\n',
      '{"type":"done","session_id":"final"}',  // No trailing newline
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(2)
    expect(collected[1]).toEqual({ type: 'done', session_id: 'final' })
  })

  it('skips empty lines in stream', async () => {
    const events = [
      '{"type":"text","delta":"A"}\n',
      '\n',                              // Empty line
      '   \n',                           // Whitespace line
      '{"type":"text","delta":"B"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(2)
    expect(collected[0]).toEqual({ type: 'text', delta: 'A' })
    expect(collected[1]).toEqual({ type: 'text', delta: 'B' })
  })

  it('handles JSON parse errors gracefully', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const events = [
      '{"type":"text","delta":"Good"}\n',
      'invalid json line\n',              // Invalid JSON
      '{"type":"text","delta":"OK"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    // Should have parsed 2 valid events, skipped the invalid one
    expect(collected).toHaveLength(2)
    expect(consoleSpy).toHaveBeenCalledWith(
      'Failed to parse stream event:',
      'invalid json line'
    )
  })

  it('handles multiple events in single chunk', async () => {
    const events = [
      '{"type":"text","delta":"A"}\n{"type":"text","delta":"B"}\n{"type":"text","delta":"C"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(3)
    expect(collected.map((e) => e.delta)).toEqual(['A', 'B', 'C'])
  })

  it('sends correct headers', async () => {
    const events = ['{"type":"done"}\n']

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    // Consume the stream
    for await (const _ of chatStream({ message: 'test' })) {
      // drain
    }

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Accept': 'application/x-ndjson',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
          'X-Notes-Person': 'sebastian',
        }),
        body: JSON.stringify({ message: 'test' }),
      })
    )
  })

  it('includes session_id in request when provided', async () => {
    const events = ['{"type":"done"}\n']

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    for await (const _ of chatStream({
      message: 'continue',
      session_id: 'existing-session',
    })) {
      // drain
    }

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        body: JSON.stringify({
          message: 'continue',
          session_id: 'existing-session',
        }),
      })
    )
  })

  it('throws ApiError on non-200 response', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ detail: 'Invalid token' }),
    } as Response)

    await expect(async () => {
      for await (const _ of chatStream({ message: 'test' })) {
        // drain
      }
    }).rejects.toThrow('Request failed with status 401')
  })

  it('throws ApiError when response body is null', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: null,
    } as Response)

    await expect(async () => {
      for await (const _ of chatStream({ message: 'test' })) {
        // drain
      }
    }).rejects.toThrow('No response body')
  })

  it('handles stream with only whitespace', async () => {
    const events = ['  \n\n  \n']

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    const collected: StreamEvent[] = []
    for await (const event of chatStream({ message: 'test' })) {
      collected.push(event)
    }

    expect(collected).toHaveLength(0)
  })

  it('accumulates text deltas correctly for UI display', async () => {
    const events = [
      '{"type":"text","delta":"The "}\n',
      '{"type":"text","delta":"quick "}\n',
      '{"type":"text","delta":"brown "}\n',
      '{"type":"text","delta":"fox"}\n',
      '{"type":"done","session_id":"test"}\n',
    ]

    global.fetch = vi.fn().mockResolvedValue(
      mockFetchResponse(createMockStream(events))
    )

    let accumulated = ''
    for await (const event of chatStream({ message: 'test' })) {
      if (event.type === 'text' && event.delta) {
        accumulated += event.delta
      }
    }

    expect(accumulated).toBe('The quick brown fox')
  })
})

describe('chat (non-streaming)', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('sends POST request and returns response', async () => {
    const mockResponse = {
      success: true,
      session_id: 'session-123',
      response: 'Hello!',
      history: [
        { role: 'user', content: 'Hi' },
        { role: 'assistant', content: 'Hello!' },
      ],
    }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockResponse,
    } as Response)

    const result = await chat({ message: 'Hi' })

    expect(result).toEqual(mockResponse)
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ message: 'Hi' }),
      })
    )
  })
})

describe('clearSession', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('sends POST request to clear session', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ success: true, message: 'Session cleared' }),
    } as Response)

    const result = await clearSession({ session_id: 'session-123' })

    expect(result).toEqual({ success: true, message: 'Session cleared' })
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/clear',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ session_id: 'session-123' }),
      })
    )
  })
})

describe('getHistory', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('sends GET request with session_id query param', async () => {
    const mockHistory = {
      success: true,
      history: [
        { role: 'user', content: 'Hello' },
        { role: 'assistant', content: 'Hi there!' },
      ],
    }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockHistory,
    } as Response)

    const result = await getHistory('session-456')

    expect(result).toEqual(mockHistory)
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/history?session_id=session-456',
      expect.objectContaining({
        method: 'GET',
      })
    )
  })
})
