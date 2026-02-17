import { useState, useEffect, FormEvent } from 'react'
import { usePerson } from '../hooks/usePerson'
import {
  fetchSleepTimes,
  appendSleepTime,
  deleteSleepTime,
  updateSleepTime,
  fetchSleepSummary,
  exportSleepMarkdown,
} from '../api/sleep'
import type { SleepEntry, SleepSummaryResponse } from '../api/types'
import styles from './SleepPage.module.css'

type SleepTab = 'log' | 'history' | 'summary'
type SleepStatus = 'eingeschlafen' | 'aufgewacht'

function currentLocalDateTimeValue(): string {
  const now = new Date()
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function localDateTimeToIso(value: string): string | undefined {
  if (!value) return undefined
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return undefined
  return parsed.toISOString()
}

export default function SleepPage() {
  const { person } = usePerson()
  const [tab, setTab] = useState<SleepTab>('log')

  const [entries, setEntries] = useState<SleepEntry[]>([])
  const [summary, setSummary] = useState<SleepSummaryResponse>({ nights: [], averages: [] })

  const [child, setChild] = useState('Fabian')
  const [status, setStatus] = useState<SleepStatus>('eingeschlafen')
  const [timeText, setTimeText] = useState('')
  const [occurredAtLocal, setOccurredAtLocal] = useState(currentLocalDateTimeValue())

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingChild, setEditingChild] = useState('Fabian')
  const [editingStatus, setEditingStatus] = useState<SleepStatus>('eingeschlafen')
  const [editingTimeText, setEditingTimeText] = useState('')
  const [editingOccurredAtLocal, setEditingOccurredAtLocal] = useState('')

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [exportStatus, setExportStatus] = useState('')

  useEffect(() => {
    if (!person) return
    loadAll()
  }, [person])

  const loadAll = async () => {
    setLoading(true)
    setError('')
    try {
      const [times, summaryResp] = await Promise.all([fetchSleepTimes(), fetchSleepSummary()])
      setEntries(times.entries)
      setSummary(summaryResp)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sleep data')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!timeText.trim() && !occurredAtLocal.trim()) return

    try {
      await appendSleepTime({
        child,
        time: timeText,
        status,
        occurred_at: localDateTimeToIso(occurredAtLocal),
      })
      setTimeText('')
      setOccurredAtLocal(currentLocalDateTimeValue())
      await loadAll()
      setTab('history')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add entry')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteSleepTime({ id })
      await loadAll()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete entry')
    }
  }

  const startEdit = (entry: SleepEntry) => {
    setEditingId(entry.id)
    setEditingChild(entry.child)
    setEditingStatus((entry.status === 'aufgewacht' ? 'aufgewacht' : 'eingeschlafen'))
    setEditingTimeText(entry.time || '')

    if (entry.occurred_at) {
      const parsed = new Date(entry.occurred_at)
      if (!Number.isNaN(parsed.getTime())) {
        const local = new Date(parsed.getTime() - parsed.getTimezoneOffset() * 60_000)
        setEditingOccurredAtLocal(local.toISOString().slice(0, 16))
      } else {
        setEditingOccurredAtLocal('')
      }
    } else {
      setEditingOccurredAtLocal('')
    }
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditingTimeText('')
    setEditingOccurredAtLocal('')
  }

  const saveEdit = async () => {
    if (!editingId) return
    if (!editingTimeText.trim() && !editingOccurredAtLocal.trim()) {
      setError('Time is required')
      return
    }

    try {
      await updateSleepTime({
        id: editingId,
        child: editingChild,
        time: editingTimeText,
        status: editingStatus,
        occurred_at: localDateTimeToIso(editingOccurredAtLocal),
      })
      cancelEdit()
      await loadAll()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update entry')
    }
  }

  const handleExportMarkdown = async () => {
    setExportStatus('')
    try {
      const resp = await exportSleepMarkdown()
      setExportStatus(resp.message)
    } catch (err) {
      setExportStatus(err instanceof Error ? err.message : 'Export failed')
    }
  }

  if (!person) {
    return <div className={styles.message}>Please select a person in Settings first.</div>
  }

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <h2>Sleep Tracking</h2>
        <button onClick={handleExportMarkdown}>Export sleep data to markdown</button>
      </div>

      <div className={styles.tabs}>
        <button className={tab === 'log' ? styles.tabActive : ''} onClick={() => setTab('log')}>Log</button>
        <button className={tab === 'history' ? styles.tabActive : ''} onClick={() => setTab('history')}>History</button>
        <button className={tab === 'summary' ? styles.tabActive : ''} onClick={() => setTab('summary')}>Summary</button>
      </div>

      {error && <p className={styles.error}>{error}</p>}
      {exportStatus && <p className={styles.status}>{exportStatus}</p>}

      {tab === 'log' && (
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
            <label className={styles.radio}>
              <input
                type="radio"
                name="status"
                checked={status === 'eingeschlafen'}
                onChange={() => setStatus('eingeschlafen')}
              />
              Eingeschlafen
            </label>
            <label className={styles.radio}>
              <input
                type="radio"
                name="status"
                checked={status === 'aufgewacht'}
                onChange={() => setStatus('aufgewacht')}
              />
              Aufgewacht
            </label>
          </div>

          <div className={styles.entryRow}>
            <input
              type="datetime-local"
              value={occurredAtLocal}
              onChange={e => setOccurredAtLocal(e.target.value)}
              className={styles.timeInput}
            />
            <input
              type="text"
              value={timeText}
              onChange={e => setTimeText(e.target.value)}
              placeholder="Legacy time/raw note"
              className={styles.timeInput}
            />
          </div>

          <div className={styles.actions}>
            <button type="submit" disabled={!timeText.trim() && !occurredAtLocal.trim()}>
              Add
            </button>
          </div>
        </form>
      )}

      {tab === 'history' && (
        <div className={styles.history}>
          <h3>Entries</h3>
          {loading ? (
            <div className={styles.message}>Loading...</div>
          ) : entries.length === 0 ? (
            <div className={styles.message}>No entries yet</div>
          ) : (
            <ul className={styles.list}>
              {entries.map(e => (
                <li key={e.id} className={styles.entry}>
                  {editingId === e.id ? (
                    <div className={styles.editGrid}>
                      <select value={editingChild} onChange={ev => setEditingChild(ev.target.value)}>
                        <option value="Thomas">Thomas</option>
                        <option value="Fabian">Fabian</option>
                      </select>
                      <select value={editingStatus} onChange={ev => setEditingStatus(ev.target.value as SleepStatus)}>
                        <option value="eingeschlafen">eingeschlafen</option>
                        <option value="aufgewacht">aufgewacht</option>
                      </select>
                      <input
                        type="datetime-local"
                        value={editingOccurredAtLocal}
                        onChange={ev => setEditingOccurredAtLocal(ev.target.value)}
                      />
                      <input
                        type="text"
                        value={editingTimeText}
                        onChange={ev => setEditingTimeText(ev.target.value)}
                      />
                      <div className={styles.actions}>
                        <button onClick={saveEdit}>Save</button>
                        <button className="ghost" onClick={cancelEdit}>Cancel</button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <span className={styles.entryText}>
                        {e.date} | {e.child} | {e.time || '-'} | {e.status}
                        {e.notes ? ` | ${e.notes}` : ''}
                      </span>
                      <div className={styles.actions}>
                        <button className="ghost" onClick={() => startEdit(e)}>Edit</button>
                        <button className="danger ghost" onClick={() => handleDelete(e.id)}>Delete</button>
                      </div>
                    </>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {tab === 'summary' && (
        <div className={styles.history}>
          <h3>Summary</h3>
          {loading ? (
            <div className={styles.message}>Loading...</div>
          ) : (
            <>
              <h4>Average Bed/Wake (7d/30d)</h4>
              {summary.averages.length === 0 ? (
                <p className={styles.message}>Not enough paired asleep/awake data yet.</p>
              ) : (
                <ul className={styles.list}>
                  {summary.averages.map(avg => (
                    <li key={`${avg.child}-${avg.days}`} className={styles.entry}>
                      <span className={styles.entryText}>
                        {avg.child} ({avg.days}d): Bed {avg.average_bedtime} | Wake {avg.average_wake_time}
                      </span>
                    </li>
                  ))}
                </ul>
              )}

              <h4>Nightly Total Duration</h4>
              {summary.nights.length === 0 ? (
                <p className={styles.message}>No completed nights yet.</p>
              ) : (
                <ul className={styles.list}>
                  {summary.nights.map((night, idx) => (
                    <li key={`${night.child}-${night.night_date}-${idx}`} className={styles.entry}>
                      <span className={styles.entryText}>
                        {night.night_date} | {night.child} | {night.duration_minutes} min | {night.bedtime} - {night.wake_time}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}
