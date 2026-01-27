import { useState, useRef, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { chatStream, clearSession } from '../api/claude'
import type { ChatMessage, StreamEvent } from '../api/types'
import styles from './ClaudePage.module.css'

export default function ClaudePage() {
  const { person } = usePerson()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const [toolStatus, setToolStatus] = useState<string | null>(null)
  const [error, setError] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingText])

  const handleSend = async () => {
    if (!input.trim() || isStreaming) return

    const userMessage = input.trim()
    setInput('')
    setMessages(prev => [...prev, { role: 'user', content: userMessage }])
    setIsStreaming(true)
    setStreamingText('')
    setToolStatus(null)
    setError('')

    let fullResponse = ''

    try {
      for await (const event of chatStream({
        message: userMessage,
        session_id: sessionId || undefined,
      })) {
        handleStreamEvent(event, (text) => {
          fullResponse += text
          setStreamingText(fullResponse)
        })
      }

      if (fullResponse) {
        setMessages(prev => [...prev, { role: 'assistant', content: fullResponse }])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message')
    } finally {
      setIsStreaming(false)
      setStreamingText('')
      setToolStatus(null)
    }
  }

  const handleStreamEvent = (
    event: StreamEvent,
    onText: (text: string) => void
  ) => {
    switch (event.type) {
      case 'text':
        if (event.delta) {
          onText(event.delta)
        }
        break
      case 'session':
        if (event.session_id) {
          setSessionId(event.session_id)
        }
        break
      case 'tool_use':
        if (event.name) {
          setToolStatus(`Using ${event.name}...`)
        }
        break
      case 'error':
        setError(event.message || 'Stream error')
        break
      case 'ping':
      case 'done':
        // Ignore
        break
    }
  }

  const handleClear = async () => {
    if (sessionId) {
      try {
        await clearSession({ session_id: sessionId })
      } catch {
        // Ignore clear errors
      }
    }
    setMessages([])
    setSessionId(null)
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
