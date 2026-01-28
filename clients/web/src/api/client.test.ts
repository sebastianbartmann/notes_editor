import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { apiRequest, apiStreamRequest, ApiError } from './client'

describe('apiRequest', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'sebastian')
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('includes Authorization header when token exists', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: 'test' }),
    } as Response)

    await apiRequest('/api/test')

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/test',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer test-token',
        }),
      })
    )
  })

  it('includes X-Notes-Person header when person exists', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: 'test' }),
    } as Response)

    await apiRequest('/api/test')

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/test',
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Notes-Person': 'sebastian',
        }),
      })
    )
  })

  it('omits Authorization header when no token', async () => {
    localStorage.removeItem('notes_token')

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: 'test' }),
    } as Response)

    await apiRequest('/api/test')

    const callHeaders = (global.fetch as any).mock.calls[0][1].headers
    expect(callHeaders.Authorization).toBeUndefined()
  })

  it('omits X-Notes-Person header when no person', async () => {
    localStorage.removeItem('notes_person')

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: 'test' }),
    } as Response)

    await apiRequest('/api/test')

    const callHeaders = (global.fetch as any).mock.calls[0][1].headers
    expect(callHeaders['X-Notes-Person']).toBeUndefined()
  })

  it('sets Content-Type for POST with body', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ success: true }),
    } as Response)

    await apiRequest('/api/test', {
      method: 'POST',
      body: { data: 'test' },
    })

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/test',
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
        body: JSON.stringify({ data: 'test' }),
      })
    )
  })

  it('throws ApiError with detail on non-200 response', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ detail: 'Invalid input' }),
    } as Response)

    await expect(apiRequest('/api/test')).rejects.toMatchObject({
      message: 'Request failed with status 400',
      status: 400,
      detail: 'Invalid input',
    })
  })

  it('throws ApiError without detail when JSON parse fails', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => {
        throw new Error('not json')
      },
    } as unknown as Response)

    await expect(apiRequest('/api/test')).rejects.toMatchObject({
      message: 'Request failed with status 500',
      status: 500,
      detail: undefined,
    })
  })
})

describe('apiStreamRequest', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.setItem('notes_token', 'test-token')
    localStorage.setItem('notes_person', 'petra')
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('sets Accept header for NDJSON', async () => {
    const mockReader = { read: vi.fn(), releaseLock: vi.fn() }
    const mockStream = { getReader: () => mockReader }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: mockStream,
    } as unknown as Response)

    await apiStreamRequest('/api/claude/chat-stream', { message: 'test' })

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        headers: expect.objectContaining({
          Accept: 'application/x-ndjson',
        }),
      })
    )
  })

  it('includes auth headers', async () => {
    const mockReader = { read: vi.fn(), releaseLock: vi.fn() }
    const mockStream = { getReader: () => mockReader }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: mockStream,
    } as unknown as Response)

    await apiStreamRequest('/api/claude/chat-stream', { message: 'test' })

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer test-token',
          'X-Notes-Person': 'petra',
        }),
      })
    )
  })

  it('uses POST method', async () => {
    const mockReader = { read: vi.fn(), releaseLock: vi.fn() }
    const mockStream = { getReader: () => mockReader }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: mockStream,
    } as unknown as Response)

    await apiStreamRequest('/api/claude/chat-stream', { message: 'test' })

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        method: 'POST',
      })
    )
  })

  it('JSON stringifies body', async () => {
    const mockReader = { read: vi.fn(), releaseLock: vi.fn() }
    const mockStream = { getReader: () => mockReader }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: mockStream,
    } as unknown as Response)

    const body = { message: 'hello', session_id: 'abc123' }
    await apiStreamRequest('/api/claude/chat-stream', body)

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/claude/chat-stream',
      expect.objectContaining({
        body: JSON.stringify(body),
      })
    )
  })

  it('returns ReadableStreamDefaultReader', async () => {
    const mockReader = {
      read: vi.fn().mockResolvedValue({ done: true, value: undefined }),
      releaseLock: vi.fn(),
    }
    const mockStream = { getReader: () => mockReader }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: mockStream,
    } as unknown as Response)

    const reader = await apiStreamRequest('/api/claude/chat-stream', { message: 'test' })

    expect(reader).toBe(mockReader)
  })

  it('throws ApiError on non-200 response', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ detail: 'Unauthorized' }),
    } as Response)

    await expect(
      apiStreamRequest('/api/claude/chat-stream', { message: 'test' })
    ).rejects.toMatchObject({
      message: 'Request failed with status 401',
      status: 401,
      detail: 'Unauthorized',
    })
  })

  it('throws ApiError when response body is null', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: null,
    } as Response)

    await expect(
      apiStreamRequest('/api/claude/chat-stream', { message: 'test' })
    ).rejects.toMatchObject({
      message: 'No response body',
      status: 0,
    })
  })
})

describe('ApiError', () => {
  it('has correct properties', () => {
    const error = new ApiError('Test error', 404, 'Not found')

    expect(error.message).toBe('Test error')
    expect(error.status).toBe(404)
    expect(error.detail).toBe('Not found')
    expect(error.name).toBe('ApiError')
    expect(error).toBeInstanceOf(Error)
  })

  it('works without detail', () => {
    const error = new ApiError('Server error', 500)

    expect(error.detail).toBeUndefined()
  })
})
