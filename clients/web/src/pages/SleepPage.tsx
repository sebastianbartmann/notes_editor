import { useState, useEffect, FormEvent } from 'react'
import { usePerson } from '../hooks/usePerson'
import { fetchSleepTimes, appendSleepTime, deleteSleepTime } from '../api/sleep'
import type { SleepEntry } from '../api/types'
import styles from './SleepPage.module.css'

export default function SleepPage() {
  const { person } = usePerson()
  const [entries, setEntries] = useState<SleepEntry[]>([])
  const [child, setChild] = useState('Fabian')
  const [entry, setEntry] = useState('')
  const [asleep, setAsleep] = useState(false)
  const [woke, setWoke] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!person) return
    loadEntries()
  }, [person])

  const loadEntries = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchSleepTimes()
      setEntries(data.entries)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load entries')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!entry.trim()) return
    try {
      await appendSleepTime({ child, entry, asleep, woke })
      setEntry('')
      setAsleep(false)
      setWoke(false)
      await loadEntries()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add entry')
    }
  }

  const handleDelete = async (lineNo: number) => {
    try {
      await deleteSleepTime({ line: lineNo })
      await loadEntries()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete entry')
    }
  }

  const handleAsleepChange = (checked: boolean) => {
    setAsleep(checked)
    if (checked) setWoke(false)
  }

  const handleWokeChange = (checked: boolean) => {
    setWoke(checked)
    if (checked) setAsleep(false)
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
      <h2>Sleep Tracking</h2>

      <form onSubmit={handleSubmit} className={styles.form}>
        <div className={styles.childSelect}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="child"
              value="Thomas"
              checked={child === 'Thomas'}
              onChange={e => setChild(e.target.value)}
            />
            Thomas
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="child"
              value="Fabian"
              checked={child === 'Fabian'}
              onChange={e => setChild(e.target.value)}
            />
            Fabian
          </label>
        </div>

        <div className={styles.statusSelect}>
          <label className={styles.checkbox}>
            <input
              type="checkbox"
              checked={asleep}
              onChange={e => handleAsleepChange(e.target.checked)}
            />
            Eingeschlafen
          </label>
          <label className={styles.checkbox}>
            <input
              type="checkbox"
              checked={woke}
              onChange={e => handleWokeChange(e.target.checked)}
            />
            Aufgewacht
          </label>
        </div>

        <div className={styles.entryRow}>
          <input
            type="text"
            value={entry}
            onChange={e => setEntry(e.target.value)}
            placeholder="19:30"
            className={styles.timeInput}
          />
          <button type="submit" disabled={!entry.trim()}>
            Add
          </button>
        </div>
      </form>

      {error && <p className={styles.error}>{error}</p>}

      <div className={styles.history}>
        <h3>Recent Entries</h3>
        {loading ? (
          <div className={styles.message}>Loading...</div>
        ) : entries.length === 0 ? (
          <div className={styles.message}>No entries yet</div>
        ) : (
          <ul className={styles.list}>
            {entries.map(e => (
              <li key={e.line_no} className={styles.entry}>
                <span className={styles.entryText}>{e.text}</span>
                <button
                  onClick={() => handleDelete(e.line_no)}
                  className="danger ghost"
                >
                  Delete
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
