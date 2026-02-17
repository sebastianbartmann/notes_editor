import { useState, useRef, useEffect, useMemo } from 'react'
import { usePerson } from '../hooks/usePerson'
import { agentChatStream, clearAgentSession, clearAllAgentSessions, exportAgentSessionsMarkdown, getAgentSessionHistory, listAgentActions, listAgentSessions } from '../api/agent'
import { useAgentSession } from '../context/AgentSessionContext'
import type { AgentAction, AgentChatRequest, AgentConversationItem, AgentSessionSummary } from '../api/types'
import styles from './ClaudePage.module.css'

const VERBOSE_OUTPUT_KEY = 'notes_agent_verbose_output'
const LEGACY_SHOW_TOOL_CALLS_KEY = 'notes_agent_show_tool_calls'

export default function ClaudePage() {
  const { person } = usePerson()
  const { getSession, setSessionId, setMessages, appendMessage, clearSession } = useAgentSession()
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const [error, setError] = useState('')
  const [actions, setActions] = useState<AgentAction[]>([])
  const [actionsError, setActionsError] = useState('')
  const [sessionsOpen, setSessionsOpen] = useState(false)
  const [sessions, setSessions] = useState<AgentSessionSummary[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [sessionsError, setSessionsError] = useState('')
  const [sessionsStatus, setSessionsStatus] = useState('')
  const [sessionsBusy, setSessionsBusy] = useState(false)
  const [verboseOutput, setVerboseOutput] = useState<boolean>(() => {
    const stored = localStorage.getItem(VERBOSE_OUTPUT_KEY)
    if (stored !== null) return stored !== 'false'
    const legacy = localStorage.getItem(LEGACY_SHOW_TOOL_CALLS_KEY)
    return legacy !== 'false'
  })
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const session = person ? getSession(person) : { sessionId: null, messages: [] as AgentConversationItem[] }
  const sessionId = session.sessionId
  const messages = session.messages
  const visibleMessages = verboseOutput
    ? messages
    : messages.filter(item =>
      item.type !== 'tool_call' &&
      item.type !== 'tool_result' &&
      item.type !== 'status' &&
      item.type !== 'usage'
    )
  const latestUsage = useMemo(() => {
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      const item = messages[i]
      if (item.type === 'usage' && item.usage) {
        return item.usage
      }
    }
    return null
  }, [messages])

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== VERBOSE_OUTPUT_KEY && event.key !== LEGACY_SHOW_TOOL_CALLS_KEY) return
      const current = localStorage.getItem(VERBOSE_OUTPUT_KEY)
      if (current !== null) {
        setVerboseOutput(current !== 'false')
        return
      }
      setVerboseOutput(localStorage.getItem(LEGACY_SHOW_TOOL_CALLS_KEY) !== 'false')
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [visibleMessages, streamingText])

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
          if (resp.items && resp.items.length > 0) {
            setMessages(person, resp.items)
            return
          }
          const legacyItems = (resp.messages || []).map((msg) => ({
            type: 'message' as const,
            role: msg.role,
            content: msg.content,
          }))
          setMessages(person, legacyItems)
        }
      })
      .catch(() => {
        // Ignore history load errors (e.g. server restart or unknown session)
      })
    return () => {
      cancelled = true
    }
  }, [person, sessionId, setMessages])

  useEffect(() => {
    if (!person || !sessionId) return
    const refresh = () => {
      if (isStreaming) return
      getAgentSessionHistory(sessionId)
        .then((resp) => {
          if (resp.items && resp.items.length > 0) {
            setMessages(person, resp.items)
            return
          }
          const legacyItems = (resp.messages || []).map((msg) => ({
            type: 'message' as const,
            role: msg.role,
            content: msg.content,
          }))
          setMessages(person, legacyItems)
        })
        .catch(() => {})
    }
    const onFocus = () => refresh()
    const onVisibility = () => {
      if (document.visibilityState === 'visible') {
        refresh()
      }
    }
    window.addEventListener('focus', onFocus)
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.removeEventListener('focus', onFocus)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [person, sessionId, isStreaming, setMessages])

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
    setSessionsStatus('')
    await loadSessions()
  }

  const handleSelectSession = (selectedSessionId: string) => {
    if (!person || isStreaming) return
    setSessionId(person, selectedSessionId)
    setSessionsOpen(false)
    setSessionsError('')
    setSessionsStatus('')
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
      setSessionsStatus('')
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
    setSessionsStatus('')
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

  const handleExportSessions = async () => {
    if (!person || isStreaming || sessionsBusy) return
    setSessionsBusy(true)
    setSessionsError('')
    setSessionsStatus('')
    try {
      const resp = await exportAgentSessionsMarkdown()
      const count = Math.max(0, (resp.files?.length || 0) - 1)
      setSessionsStatus(`Exported ${count} session file(s) to ${resp.directory}`)
    } catch (err) {
      setSessionsError(err instanceof Error ? err.message : 'Failed to export sessions')
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
      appendMessage(person, { type: 'message', role: 'user', content: userBubble })
    }
    setIsStreaming(true)
    setStreamingText('')
    setError('')

    let bufferedText = ''
    const flushBufferedText = () => {
      if (!bufferedText) return
      appendMessage(person, { type: 'message', role: 'assistant', content: bufferedText })
      bufferedText = ''
      setStreamingText('')
    }

    try {
      for await (const event of agentChatStream(request)) {
        switch (event.type) {
          case 'start':
            if (event.session_id) {
              setSessionId(person, event.session_id)
            }
            break
          case 'text':
            if (event.delta) {
              bufferedText += event.delta
              setStreamingText(bufferedText)
            }
            break
          case 'tool_call':
            flushBufferedText()
            if (verboseOutput) {
              appendMessage(person, {
                type: 'tool_call',
                tool: event.tool,
                args: event.args,
                run_id: event.run_id,
                seq: event.seq,
                ts: event.ts,
              })
            }
            break
          case 'tool_result':
            flushBufferedText()
            if (verboseOutput) {
              appendMessage(person, {
                type: 'tool_result',
                tool: event.tool,
                ok: event.ok,
                summary: event.summary,
                run_id: event.run_id,
                seq: event.seq,
                ts: event.ts,
              })
            }
            break
          case 'status':
            flushBufferedText()
            if (verboseOutput) {
              appendMessage(person, {
                type: 'status',
                message: event.message,
                run_id: event.run_id,
                seq: event.seq,
                ts: event.ts,
              })
            }
            break
          case 'error':
            flushBufferedText()
            appendMessage(person, {
              type: 'error',
              message: event.message || 'Stream error',
              run_id: event.run_id,
              seq: event.seq,
              ts: event.ts,
            })
            setError(event.message || 'Stream error')
            break
          case 'usage':
            flushBufferedText()
            if (verboseOutput) {
              appendMessage(person, {
                type: 'usage',
                usage: event.usage,
                run_id: event.run_id,
                seq: event.seq,
                ts: event.ts,
              })
            }
            break
          case 'done':
            flushBufferedText()
            if (event.session_id) {
              setSessionId(person, event.session_id)
            }
            break
        }
      }
      flushBufferedText()
    } catch (err) {
      flushBufferedText()
      setError(err instanceof Error ? err.message : 'Failed to send message')
    } finally {
      setIsStreaming(false)
      setStreamingText('')
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

  const formatUsage = (item: AgentConversationItem) => {
    const usage = item.usage
    if (!usage) return 'Usage update'
    const total = usage.total_tokens ?? 0
    const left = usage.remaining_tokens
    const window = usage.context_window
    if (typeof left === 'number' && typeof window === 'number' && window > 0) {
      return `Usage: ${total} tokens, ${left} left of ${window}`
    }
    return `Usage: ${total} tokens`
  }

  const formatUsageSummary = () => {
    if (!verboseOutput) {
      return 'Verbose output disabled.'
    }
    if (!latestUsage) {
      return 'Context usage not available yet.'
    }
    const total = latestUsage.total_tokens ?? 0
    const left = latestUsage.remaining_tokens
    const window = latestUsage.context_window
    if (typeof left === 'number' && typeof window === 'number' && window > 0) {
      return `Context: ${total.toLocaleString()} used, ${left.toLocaleString()} left of ${window.toLocaleString()}.`
    }
    return `Context: ${total.toLocaleString()} tokens used.`
  }

  const formatItemMeta = (item: AgentConversationItem) => {
    const parts: string[] = []
    if (item.seq) {
      parts.push(`#${item.seq}`)
    }
    if (item.ts) {
      const parsed = new Date(item.ts)
      if (!Number.isNaN(parsed.getTime())) {
        parts.push(parsed.toLocaleTimeString())
      }
    }
    return parts.join(' • ')
  }

  const renderInlineItem = (item: AgentConversationItem, key: string) => {
    if (item.type === 'message') {
      const isUser = item.role === 'user'
      return (
        <div
          key={key}
          className={`${styles.bubble} ${isUser ? styles.userBubble : styles.assistantBubble}`}
        >
          {item.content}
        </div>
      )
    }

    const meta = formatItemMeta(item)
    if (item.type === 'tool_call') {
      return (
        <div key={key} className={`${styles.eventRow} ${styles.eventToolCall}`}>
          <div className={styles.eventTitle}>Tool call: {item.tool || 'unknown'}</div>
          {meta && <div className={styles.eventMeta}>{meta}</div>}
          {item.args && (
            <details className={styles.eventArgs}>
              <summary>Arguments</summary>
              <pre>{JSON.stringify(item.args, null, 2)}</pre>
            </details>
          )}
        </div>
      )
    }
    if (item.type === 'tool_result') {
      const status = item.ok === false ? 'failed' : 'finished'
      return (
        <div key={key} className={`${styles.eventRow} ${styles.eventToolResult}`}>
          <div className={styles.eventTitle}>Tool {item.tool || 'unknown'} {status}</div>
          {item.summary && <div>{item.summary}</div>}
          {meta && <div className={styles.eventMeta}>{meta}</div>}
        </div>
      )
    }
    if (item.type === 'status') {
      return (
        <div key={key} className={`${styles.eventRow} ${styles.eventStatus}`}>
          <div className={styles.eventTitle}>{item.message || 'Status update'}</div>
          {meta && <div className={styles.eventMeta}>{meta}</div>}
        </div>
      )
    }
    if (item.type === 'usage') {
      return (
        <div key={key} className={`${styles.eventRow} ${styles.eventUsage}`}>
          <div className={styles.eventTitle}>{formatUsage(item)}</div>
          {meta && <div className={styles.eventMeta}>{meta}</div>}
        </div>
      )
    }
    return (
      <div key={key} className={`${styles.eventRow} ${styles.eventError}`}>
        <div className={styles.eventTitle}>{item.message || 'Error'}</div>
        {meta && <div className={styles.eventMeta}>{meta}</div>}
      </div>
    )
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
      <div className={styles.sessionInfo}>
        <div><strong>Session:</strong> {sessionId || 'new'}</div>
        <div>{formatUsageSummary()}</div>
      </div>

      <div className={styles.chat}>
        <div className={styles.messages}>
          {visibleMessages.map((item, i) => renderInlineItem(item, `${i}`))}
          {streamingText && (
            <div className={`${styles.bubble} ${styles.assistantBubble}`}>
              {streamingText}
              <span className={styles.cursor}>|</span>
            </div>
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
            {sessionsStatus && <div className={styles.message}>{sessionsStatus}</div>}
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
                className="ghost"
                onClick={handleExportSessions}
                disabled={sessionsBusy || isStreaming}
              >
                {sessionsBusy ? 'Working...' : 'Export .md'}
              </button>
              <button
                className="danger"
                onClick={handleDeleteAllSessions}
                disabled={sessionsBusy || isStreaming}
              >
                Delete all sessions
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
