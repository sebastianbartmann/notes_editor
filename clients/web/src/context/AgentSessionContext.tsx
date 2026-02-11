import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ChatMessage } from '../api/types'

type Person = 'sebastian' | 'petra'

interface AgentSessionState {
  sessionId: string | null
  messages: ChatMessage[]
}

interface AgentSessionContextType {
  getSession: (person: Person) => AgentSessionState
  setSessionId: (person: Person, sessionId: string | null) => void
  setMessages: (person: Person, messages: ChatMessage[]) => void
  appendMessage: (person: Person, message: ChatMessage) => void
  clearSession: (person: Person) => void
}

const STORAGE_KEY = 'notes_agent_sessions'
const MAX_MESSAGES_PER_PERSON = 200

const EMPTY_SESSION: AgentSessionState = { sessionId: null, messages: [] }

type SessionMap = Record<Person, AgentSessionState>

function normalizeMessages(value: unknown): ChatMessage[] {
  if (!Array.isArray(value)) return []
  const out: ChatMessage[] = []
  for (const item of value) {
    if (!item || typeof item !== 'object') continue
    const role = (item as Record<string, unknown>).role
    const content = (item as Record<string, unknown>).content
    if ((role === 'user' || role === 'assistant') && typeof content === 'string') {
      out.push({ role, content })
    }
  }
  return out.slice(-MAX_MESSAGES_PER_PERSON)
}

function parseStoredSessions(): Partial<SessionMap> {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const out: Partial<SessionMap> = {}

    // Backward compatibility: older format stored only session IDs as strings.
    if (typeof parsed.sebastian === 'string' || typeof parsed.petra === 'string') {
      if (typeof parsed.sebastian === 'string' && parsed.sebastian.trim()) {
        out.sebastian = { sessionId: parsed.sebastian, messages: [] }
      }
      if (typeof parsed.petra === 'string' && parsed.petra.trim()) {
        out.petra = { sessionId: parsed.petra, messages: [] }
      }
      return out
    }

    const sebastian = parsed.sebastian
    if (sebastian && typeof sebastian === 'object') {
      const personState = sebastian as Record<string, unknown>
      const sessionId = typeof personState.sessionId === 'string' && personState.sessionId.trim()
        ? personState.sessionId
        : null
      const messages = normalizeMessages(personState.messages)
      out.sebastian = { sessionId, messages }
    }

    const petra = parsed.petra
    if (petra && typeof petra === 'object') {
      const personState = petra as Record<string, unknown>
      const sessionId = typeof personState.sessionId === 'string' && personState.sessionId.trim()
        ? personState.sessionId
        : null
      const messages = normalizeMessages(personState.messages)
      out.petra = { sessionId, messages }
    }
    return out
  } catch {
    return {}
  }
}

const AgentSessionContext = createContext<AgentSessionContextType | null>(null)

export function AgentSessionProvider({ children }: { children: ReactNode }) {
  const [sessions, setSessions] = useState<SessionMap>(() => {
    const stored = parseStoredSessions()
    return {
      sebastian: stored.sebastian || { sessionId: null, messages: [] },
      petra: stored.petra || { sessionId: null, messages: [] },
    }
  })

  useEffect(() => {
    const payload = {
      sebastian: {
        sessionId: sessions.sebastian.sessionId || null,
        messages: sessions.sebastian.messages.slice(-MAX_MESSAGES_PER_PERSON),
      },
      petra: {
        sessionId: sessions.petra.sessionId || null,
        messages: sessions.petra.messages.slice(-MAX_MESSAGES_PER_PERSON),
      },
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
  }, [sessions.sebastian.sessionId, sessions.sebastian.messages, sessions.petra.sessionId, sessions.petra.messages])

  const getSession = useCallback((person: Person) => sessions[person] || EMPTY_SESSION, [sessions])

  const setSessionId = useCallback((person: Person, sessionId: string | null) => {
    setSessions((prev) => ({
      ...prev,
      [person]: { ...prev[person], sessionId },
    }))
  }, [])

  const setMessages = useCallback((person: Person, messages: ChatMessage[]) => {
    setSessions((prev) => ({
      ...prev,
      [person]: { ...prev[person], messages },
    }))
  }, [])

  const appendMessage = useCallback((person: Person, message: ChatMessage) => {
    setSessions((prev) => ({
      ...prev,
      [person]: {
        ...prev[person],
        messages: [...prev[person].messages, message].slice(-MAX_MESSAGES_PER_PERSON),
      },
    }))
  }, [])

  const clearSession = useCallback((person: Person) => {
    setSessions((prev) => ({
      ...prev,
      [person]: { sessionId: null, messages: [] },
    }))
  }, [])

  const value: AgentSessionContextType = useMemo(() => ({
    getSession,
    setSessionId,
    setMessages,
    appendMessage,
    clearSession,
  }), [getSession, setSessionId, setMessages, appendMessage, clearSession])

  return (
    <AgentSessionContext.Provider value={value}>
      {children}
    </AgentSessionContext.Provider>
  )
}

export function useAgentSession() {
  const context = useContext(AgentSessionContext)
  if (!context) {
    throw new Error('useAgentSession must be used within an AgentSessionProvider')
  }
  return context
}
