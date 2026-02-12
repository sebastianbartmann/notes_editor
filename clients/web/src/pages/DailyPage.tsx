import { KeyboardEvent, useEffect, useState } from 'react'
import { usePerson } from '../hooks/usePerson'
import { fetchDaily, saveDaily, appendDaily } from '../api/daily'
import { syncIfStale } from '../api/sync'
import { addTodo, toggleTodo } from '../api/todos'
import { listFiles, readFile, unpinEntry } from '../api/files'
import NoteView from '../components/NoteView/NoteView'
import Editor from '../components/Editor/Editor'
import styles from './DailyPage.module.css'

function dateFromDailyPath(path: string): string {
  const name = path.split('/').pop() || ''
  const m = name.match(/^(\d{4}-\d{2}-\d{2})\.md$/)
  return m?.[1] ?? ''
}

function buildDailyPaths(entries: { name: string; path: string; is_dir: boolean }[], todayDate: string): string[] {
  const out: string[] = []
  for (const e of entries) {
    if (e.is_dir) continue
    const m = e.name.match(/^(\d{4}-\d{2}-\d{2})\.md$/)
    if (!m) continue
    const d = m[1]
    if (d > todayDate) continue
    out.push(e.path)
  }
  return Array.from(new Set(out)).sort()
}

type TaskCategory = 'work' | 'priv'

export default function DailyPage() {
  const { person } = usePerson()
  const [content, setContent] = useState('')
  const [path, setPath] = useState('')
  const [date, setDate] = useState('')
  const [todayPath, setTodayPath] = useState('')
  const [availableDailyPaths, setAvailableDailyPaths] = useState<string[]>([])
  const [isEditing, setIsEditing] = useState(false)
  const [taskInputMode, setTaskInputMode] = useState<TaskCategory | null>(null)
  const [taskInputText, setTaskInputText] = useState('')
  const [appendText, setAppendText] = useState('')
  const [isPinned, setIsPinned] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!person) return
    loadToday()
  }, [person])

  const loadToday = async () => {
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
      setTodayPath(data.path)

      try {
        const listing = await listFiles('daily')
        const fromFiles = buildDailyPaths(listing.entries, data.date)
        setAvailableDailyPaths(Array.from(new Set([...fromFiles, data.path])).sort())
      } catch {
        setAvailableDailyPaths([data.path])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load daily note')
    } finally {
      setLoading(false)
    }
  }

  const loadPath = async (targetPath: string) => {
    if (!targetPath) return
    if (isEditing) {
      const ok = window.confirm('Discard unsaved changes?')
      if (!ok) return
      setIsEditing(false)
    }
    setLoading(true)
    setError('')
    try {
      await syncIfStale({ maxAgeMs: 30_000, timeoutMs: 2_000 })
      const data = await readFile(targetPath)
      setContent(data.content)
      setPath(data.path)
      setDate(dateFromDailyPath(data.path))
      setAppendText('')
      setIsPinned(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load note')
    } finally {
      setLoading(false)
    }
  }

  const reloadCurrentNote = async () => {
    if (path === todayPath) {
      await loadToday()
    } else {
      await loadPath(path)
    }
  }

  const currentIndex = availableDailyPaths.indexOf(path)
  const prevPath = currentIndex > 0 ? availableDailyPaths[currentIndex - 1] : ''
  const nextPath =
    currentIndex !== -1 && currentIndex < availableDailyPaths.length - 1
      ? availableDailyPaths[currentIndex + 1]
      : ''

  const handleSave = async (newContent: string) => {
    try {
      await saveDaily({ path, content: newContent })
      setContent(newContent)
      setIsEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const handleSubmitTask = async () => {
    if (!taskInputMode) return
    try {
      await addTodo({ category: taskInputMode, text: taskInputText })
      setTaskInputMode(null)
      setTaskInputText('')
      await reloadCurrentNote()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add task')
    }
  }

  const handleTaskInputKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      void handleSubmitTask()
    }
  }

  const handleAppend = async () => {
    if (!appendText.trim()) return
    try {
      await appendDaily({ path, text: appendText, pinned: isPinned })
      setAppendText('')
      setIsPinned(false)
      await reloadCurrentNote()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to append')
    }
  }

  const handleTaskToggle = async (line: number) => {
    try {
      await toggleTodo({ path, line })
      await reloadCurrentNote()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle task')
    }
  }

  const handleUnpin = async (line: number) => {
    try {
      await unpinEntry({ path, line })
      await reloadCurrentNote()
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
        <div className={styles.navRow}>
          <button
            onClick={() => loadPath(prevPath)}
            className="ghost"
            disabled={!prevPath}
          >
            prev
          </button>
          <button
            onClick={() => loadPath(nextPath)}
            className="ghost"
            disabled={!nextPath}
          >
            next
          </button>
          {todayPath && path !== todayPath ? (
            <button onClick={loadToday} className={styles.todayLink}>
              {date}
            </button>
          ) : (
            <h2 className={styles.date}>{date}</h2>
          )}
        </div>
        <div className={styles.actions}>
          {!isEditing && taskInputMode === null && (
            <>
              <button onClick={() => setTaskInputMode('work')} className="ghost">Work task</button>
              <button onClick={() => setTaskInputMode('priv')} className="ghost">Priv task</button>
              <button onClick={() => setIsEditing(true)}>Edit</button>
            </>
          )}
          <button onClick={loadToday} className="ghost">
            Refresh
          </button>
        </div>
      </div>

      {taskInputMode && !isEditing && (
        <div className={styles.taskInlineForm}>
          <input
            type="text"
            value={taskInputText}
            onChange={e => setTaskInputText(e.target.value)}
            onKeyDown={handleTaskInputKeyDown}
            placeholder="Task description"
            className={styles.taskInput}
            autoFocus
          />
          <button onClick={() => void handleSubmitTask()}>
            Save
          </button>
          <button
            onClick={() => {
              setTaskInputMode(null)
              setTaskInputText('')
            }}
            className="ghost"
          >
            Cancel
          </button>
        </div>
      )}

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
