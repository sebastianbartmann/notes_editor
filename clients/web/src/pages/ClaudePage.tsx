import { useState, useRef, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { agentChatStream, clearAgentSession, clearAllAgentSessions, getAgentSessionHistory, listAgentActions, listAgentSessions } from '../api/agent'
import { useAgentSession } from '../context/AgentSessionContext'
import type { AgentAction, AgentChatRequest, AgentSessionSummary, AgentStreamEvent, ChatMessage } from '../api/types'
import styles from './ClaudePage.module.css'

export default function ClaudePage() {
  const { person } = usePerson()
  const { getSession, setSessionId, setMessages, appendMessage, clearSession } = useAgentSession()
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const [toolStatus, setToolStatus] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [actions, setActions] = useState<AgentAction[]>([])
  const [actionsError, setActionsError] = useState('')
  const [sessionsOpen, setSessionsOpen] = useState(false)
  const [sessions, setSessions] = useState<AgentSessionSummary[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [sessionsError, setSessionsError] = useState('')
  const [sessionsBusy, setSessionsBusy] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const session = person ? getSession(person) : { sessionId: null, messages: [] as ChatMessage[] }
  const sessionId = session.sessionId
  const messages = session.messages

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingText])

  useEffect(() => {
    if (!person) return
    listAgentActions()
      .then(setActions)
      .catch(err => setActionsError(err instanceof Error ? err.message : 'Failed to load actions'))
  }, [person])

  useEffect(() => {
    if (!person || !sessionId) return
    let cancelled = false
    getAgentSessionHistory(sessionId)
      .then((resp) => {
        if (!cancelled) {
          setMessages(person, resp.messages || [])
        }
      })
      .catch(() => {
        // Ignore history load errors (e.g. server restart or unknown session)
      })
    return () => {
      cancelled = true
    }
  }, [person, sessionId, setMessages])

  const loadSessions = async () => {
    if (!person) return
    setSessionsLoading(true)
    setSessionsError('')
    try {
      const items = await listAgentSessions()
      setSessions(items)
    } catch (err) {
      setSessionsError(err instanceof Error ? err.message : 'Failed to load sessions')
    } finally {
      setSessionsLoading(false)
    }
  }

  const handleOpenSessions = async () => {
    if (isStreaming || !person) return
    setSessionsOpen(true)
    await loadSessions()
  }

  const handleSelectSession = (selectedSessionId: string) => {
    if (!person || isStreaming) return
    setSessionId(person, selectedSessionId)
    setSessionsOpen(false)
    setSessionsError('')
    setError('')
    setStreamingText('')
  }

  const handleDeleteAllSessions = async () => {
    if (!person || isStreaming || sessionsBusy) return
    const ok = window.confirm('Delete all sessions for this person? This cannot be undone.')
    if (!ok) return
    setSessionsBusy(true)
    try {
      await clearAllAgentSessions()
      clearSession(person)
      setSessions([])
      setSessionsOpen(false)
      setStreamingText('')
      setError('')
    } catch (err) {
      setSessionsError(err instanceof Error ? err.message : 'Failed to delete sessions')
    } finally {
      setSessionsBusy(false)
    }
  }

  const handleDeleteSession = async (targetSessionId: string) => {
    if (!person || isStreaming || sessionsBusy) return
    const ok = window.confirm('Delete this session?')
    if (!ok) return
    setSessionsBusy(true)
    setSessionsError('')
    try {
      await clearAgentSession(targetSessionId)
      setSessions(prev => prev.filter(item => item.session_id !== targetSessionId))
      if (sessionId === targetSessionId) {
        clearSession(person)
        setStreamingText('')
        setError('')
      }
    } catch (err) {
      setSessionsError(err instanceof Error ? err.message : 'Failed to delete session')
    } finally {
      setSessionsBusy(false)
    }
  }

  const handleSend = async () => {
    if (!input.trim() || isStreaming) return
    const userMessage = input.trim()
    setInput('')
    await runStream(
      { message: userMessage, session_id: sessionId || undefined },
      userMessage
    )
  }

  const runStream = async (request: AgentChatRequest, userBubble?: string) => {
    if (isStreaming || !person) return

    if (userBubble) {
      appendMessage(person, { role: 'user', content: userBubble })
    }
    setIsStreaming(true)
    setStreamingText('')
    setToolStatus(null)
    setError('')

    let fullResponse = ''
    try {
      for await (const event of agentChatStream(request)) {
        handleStreamEvent(event, (text) => {
          fullResponse += text
          setStreamingText(fullResponse)
        })
      }

      if (fullResponse) {
        appendMessage(person, { role: 'assistant', content: fullResponse })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message')
    } finally {
      setIsStreaming(false)
      setStreamingText('')
      setToolStatus(null)
    }
  }

  const handleActionRun = async (action: AgentAction) => {
    if (isStreaming) return
    if (action.metadata.requires_confirmation) {
      const ok = window.confirm(`Run action "${action.label}"?`)
      if (!ok) return
    }
    await runStream(
      {
        message: '',
        action_id: action.id,
        session_id: sessionId || undefined,
        confirm: action.metadata.requires_confirmation || undefined,
      },
      `Run action: ${action.label}`
    )
  }

  const handleStreamEvent = (
    event: AgentStreamEvent,
    onText: (text: string) => void
  ) => {
    if (!person) return
    switch (event.type) {
      case 'start':
        if (event.session_id) {
          setSessionId(person, event.session_id)
        }
        break
      case 'text':
        if (event.delta) {
          onText(event.delta)
        }
        break
      case 'tool_call':
        if (event.tool) {
          setToolStatus(`Using ${event.tool}...`)
        }
        break
      case 'tool_result':
        if (event.summary) {
          setToolStatus(event.summary)
        } else if (event.tool) {
          setToolStatus(`Tool ${event.tool} finished`)
        }
        break
      case 'status':
        setToolStatus(event.message || null)
        break
      case 'error':
        setError(event.message || 'Stream error')
        break
      case 'done':
        if (event.session_id) {
          setSessionId(person, event.session_id)
        }
        break
    }
  }

  const handleNewSession = () => {
    if (!person) return
    if (isStreaming) return
    clearSession(person)
    setStreamingText('')
    setError('')
  }

  const formatSessionMeta = (session: AgentSessionSummary) => {
    const when = new Date(session.last_used_at).toLocaleString()
    return `${session.message_count} msgs • ${when}`
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  if (!person) {
    return (
      <div className={styles.message}>
        Please select a person in Settings first.
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h2>Claude</h2>
        <div className={styles.headerActions}>
          <button onClick={handleOpenSessions} className="ghost" disabled={isStreaming}>
            Sessions
          </button>
          <button onClick={handleNewSession} className="ghost" disabled={isStreaming}>
            New
          </button>
        </div>
      </div>

      <div className={styles.actionsRow}>
        {actions.map(action => (
          <button
            key={action.id}
            className={styles.actionButton}
            onClick={() => handleActionRun(action)}
            disabled={isStreaming}
            title={action.metadata.requires_confirmation ? 'Requires confirmation' : action.path}
          >
            {action.label}
          </button>
        ))}
      </div>
      {actionsError && <div className={styles.error}>{actionsError}</div>}

      <div className={styles.chat}>
        <div className={styles.messages}>
          {messages.map((msg, i) => (
            <div
              key={i}
              className={`${styles.bubble} ${
                msg.role === 'user' ? styles.userBubble : styles.assistantBubble
              }`}
            >
              {msg.content}
            </div>
          ))}
          {streamingText && (
            <div className={`${styles.bubble} ${styles.assistantBubble}`}>
              {streamingText}
              <span className={styles.cursor}>|</span>
            </div>
          )}
          {toolStatus && (
            <div className={styles.toolStatus}>{toolStatus}</div>
          )}
          {error && <div className={styles.error}>{error}</div>}
          <div ref={messagesEndRef} />
        </div>

        <div className={styles.inputArea}>
          <textarea
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            rows={2}
            disabled={isStreaming}
            className={styles.input}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isStreaming}
          >
            {isStreaming ? 'Sending...' : 'Send'}
          </button>
        </div>
      </div>

      {sessionsOpen && (
        <div className={styles.modalBackdrop} onClick={() => setSessionsOpen(false)}>
          <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
            <div className={styles.modalHeader}>
              <h3>Sessions</h3>
              <button className="ghost" onClick={() => setSessionsOpen(false)} disabled={sessionsBusy}>
                Close
              </button>
            </div>
            {sessionsLoading && <div className={styles.message}>Loading sessions...</div>}
            {!sessionsLoading && sessions.length === 0 && !sessionsError && (
              <div className={styles.message}>No sessions yet.</div>
            )}
            {sessionsError && <div className={styles.error}>{sessionsError}</div>}
            {!sessionsLoading && sessions.length > 0 && (
              <div className={styles.sessionList}>
                {sessions.map((item) => (
                  <div key={item.session_id} className={styles.sessionRowWrap}>
                    <button
                      className={`${styles.sessionRow} ${item.session_id === sessionId ? styles.sessionRowActive : ''}`}
                      onClick={() => handleSelectSession(item.session_id)}
                      disabled={sessionsBusy || isStreaming}
                    >
                      <span className={styles.sessionName}>{item.name}</span>
                      <span className={styles.sessionMeta}>{formatSessionMeta(item)}</span>
                      {item.last_preview && <span className={styles.sessionPreview}>{item.last_preview}</span>}
                    </button>
                    <button
                      className={styles.sessionDelete}
                      onClick={() => handleDeleteSession(item.session_id)}
                      disabled={sessionsBusy || isStreaming}
                      title="Delete session"
                      aria-label="Delete session"
                    >
                      🗑
                    </button>
                  </div>
                ))}
              </div>
            )}
            <div className={styles.modalFooter}>
              <button
                className="danger"
                onClick={handleDeleteAllSessions}
                disabled={sessionsBusy || isStreaming}
              >
                {sessionsBusy ? 'Deleting...' : 'Delete all sessions'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
