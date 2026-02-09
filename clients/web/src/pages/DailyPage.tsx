import { useState, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { fetchDaily, saveDaily, appendDaily } from '../api/daily'
import { syncIfStale } from '../api/sync'
import { toggleTodo } from '../api/todos'
import { unpinEntry } from '../api/files'
import NoteView from '../components/NoteView/NoteView'
import Editor from '../components/Editor/Editor'
import styles from './DailyPage.module.css'

export default function DailyPage() {
  const { person } = usePerson()
  const [content, setContent] = useState('')
  const [path, setPath] = useState('')
  const [date, setDate] = useState('')
  const [isEditing, setIsEditing] = useState(false)
  const [appendText, setAppendText] = useState('')
  const [isPinned, setIsPinned] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!person) return
    loadDaily()
  }, [person])

  const loadDaily = async () => {
    setLoading(true)
    setError('')
    try {
      // Pull at most once per interval to keep the daily view fresh without
      // paying the git/network cost on every navigation.
      await syncIfStale({ maxAgeMs: 30_000, timeoutMs: 2_000 })
      const data = await fetchDaily()
      setContent(data.content)
      setPath(data.path)
      setDate(data.date)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load daily note')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async (newContent: string) => {
    try {
      await saveDaily({ path, content: newContent })
      setContent(newContent)
      setIsEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const handleAppend = async () => {
    if (!appendText.trim()) return
    try {
      await appendDaily({ path, text: appendText, pinned: isPinned })
      setAppendText('')
      setIsPinned(false)
      await loadDaily()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to append')
    }
  }

  const handleTaskToggle = async (line: number) => {
    try {
      await toggleTodo({ path, line })
      await loadDaily()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle task')
    }
  }

  const handleUnpin = async (line: number) => {
    try {
      await unpinEntry({ path, line })
      await loadDaily()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpin')
    }
  }

  if (!person) {
    return (
      <div className={styles.message}>
        Please select a person in Settings first.
      </div>
    )
  }

  if (loading) {
    return <div className={styles.message}>Loading...</div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h2 className={styles.date}>{date}</h2>
        <div className={styles.actions}>
          {!isEditing && (
            <button onClick={() => setIsEditing(true)}>Edit</button>
          )}
          <button onClick={loadDaily} className="ghost">
            Refresh
          </button>
        </div>
      </div>

      {error && <p className={styles.error}>{error}</p>}

      {isEditing ? (
        <Editor
          content={content}
          onSave={handleSave}
          onCancel={() => setIsEditing(false)}
        />
      ) : (
        <NoteView
          content={content}
          onTaskToggle={handleTaskToggle}
          onUnpin={handleUnpin}
        />
      )}

      <div className={styles.appendForm}>
        <textarea
          value={appendText}
          onChange={e => setAppendText(e.target.value)}
          placeholder="Add a note..."
          rows={3}
          className={styles.appendInput}
        />
        <div className={styles.appendActions}>
          <label className={styles.pinnedLabel}>
            <input
              type="checkbox"
              checked={isPinned}
              onChange={e => setIsPinned(e.target.checked)}
            />
            Pinned
          </label>
          <button onClick={handleAppend} disabled={!appendText.trim()}>
            Append
          </button>
        </div>
      </div>
    </div>
  )
}
