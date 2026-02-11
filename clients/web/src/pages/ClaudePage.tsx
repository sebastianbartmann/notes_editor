import { useState, useRef, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { agentChatStream, clearAgentSession, getAgentSessionHistory, listAgentActions } from '../api/agent'
import { useAgentSession } from '../context/AgentSessionContext'
import type { AgentAction, AgentChatRequest, AgentStreamEvent, ChatMessage } from '../api/types'
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
  const [historyHydratedKey, setHistoryHydratedKey] = useState<string | null>(null)
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
    if (!person || !sessionId || messages.length > 0) return
    const hydrateKey = `${person}:${sessionId}`
    if (historyHydratedKey === hydrateKey) return
    setHistoryHydratedKey(hydrateKey)
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
  }, [person, sessionId, messages.length, historyHydratedKey, setMessages])

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
        // Ignore
        break
    }
  }

  const handleClear = async () => {
    if (!person) return
    if (sessionId) {
      try {
        await clearAgentSession(sessionId)
      } catch {
        // Ignore clear errors
      }
    }
    clearSession(person)
    setHistoryHydratedKey(null)
    setStreamingText('')
    setError('')
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
        <button onClick={handleClear} className="ghost" disabled={isStreaming}>
          Clear
        </button>
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
    </div>
  )
}
