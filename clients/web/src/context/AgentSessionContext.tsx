import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { AgentConversationItem } from '../api/types'

type Person = 'sebastian' | 'petra'

interface AgentSessionState {
  sessionId: string | null
  messages: AgentConversationItem[]
}

interface AgentSessionContextType {
  getSession: (person: Person) => AgentSessionState
  setSessionId: (person: Person, sessionId: string | null) => void
  setMessages: (person: Person, messages: AgentConversationItem[]) => void
  appendMessage: (person: Person, message: AgentConversationItem) => void
  clearSession: (person: Person) => void
}

const STORAGE_KEY = 'notes_agent_sessions'

const EMPTY_SESSION: AgentSessionState = { sessionId: null, messages: [] }

type SessionMap = Record<Person, AgentSessionState>

function parseStoredSessionIds(): Partial<Record<Person, string>> {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const out: Partial<Record<Person, string>> = {}
    if (typeof parsed.sebastian === 'string' && parsed.sebastian.trim()) {
      out.sebastian = parsed.sebastian
    }
    if (typeof parsed.petra === 'string' && parsed.petra.trim()) {
      out.petra = parsed.petra
    }

    // Backward compatibility with session object format.
    if (!out.sebastian && parsed.sebastian && typeof parsed.sebastian === 'object') {
      const personState = parsed.sebastian as Record<string, unknown>
      if (typeof personState.sessionId === 'string' && personState.sessionId.trim()) {
        out.sebastian = personState.sessionId
      }
    }
    if (!out.petra && parsed.petra && typeof parsed.petra === 'object') {
      const personState = parsed.petra as Record<string, unknown>
      if (typeof personState.sessionId === 'string' && personState.sessionId.trim()) {
        out.petra = personState.sessionId
      }
    }
    return out
  } catch {
    return {}
  }
}

const AgentSessionContext = createContext<AgentSessionContextType | null>(null)

export function AgentSessionProvider({ children }: { children: ReactNode }) {
  const [sessions, setSessions] = useState<SessionMap>(() => {
    const ids = parseStoredSessionIds()
    return {
      sebastian: { sessionId: ids.sebastian || null, messages: [] },
      petra: { sessionId: ids.petra || null, messages: [] },
    }
  })

  useEffect(() => {
    const payload = {
      sebastian: sessions.sebastian.sessionId || null,
      petra: sessions.petra.sessionId || null,
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
  }, [sessions.sebastian.sessionId, sessions.petra.sessionId])

  const getSession = useCallback((person: Person) => sessions[person] || EMPTY_SESSION, [sessions])

  const setSessionId = useCallback((person: Person, sessionId: string | null) => {
    setSessions((prev) => ({
      ...prev,
      [person]: { ...prev[person], sessionId },
    }))
  }, [])

  const setMessages = useCallback((person: Person, messages: AgentConversationItem[]) => {
    setSessions((prev) => ({
      ...prev,
      [person]: { ...prev[person], messages },
    }))
  }, [])

  const appendMessage = useCallback((person: Person, message: AgentConversationItem) => {
    setSessions((prev) => ({
      ...prev,
      [person]: {
        ...prev[person],
        messages: [...prev[person].messages, message],
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
