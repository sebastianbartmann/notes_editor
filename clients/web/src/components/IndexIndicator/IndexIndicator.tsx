import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { fetchIndexStatus, type IndexStatus } from '../../api/index'
import { usePerson } from '../../hooks/usePerson'
import styles from './IndexIndicator.module.css'

function formatAge(ms: number): string {
  if (ms < 5_000) return 'now'
  if (ms < 60_000) return `${Math.round(ms / 1_000)}s`
  if (ms < 60 * 60_000) return `${Math.round(ms / 60_000)}m`
  return `${Math.round(ms / (60 * 60_000))}h`
}

export default function IndexIndicator() {
  const { person } = usePerson()
  const navigate = useNavigate()
  const [status, setStatus] = useState<IndexStatus | null>(null)

  useEffect(() => {
    if (!person) return

    let cancelled = false
    let interval: number | undefined

    const tick = async () => {
      try {
        const s = await fetchIndexStatus()
        if (cancelled) return
        setStatus(s)
      } catch {
        // Best-effort indicator only.
      }
    }

    void tick()
    interval = window.setInterval(tick, 5_000)

    const onVisibility = () => {
      if (document.visibilityState === 'visible') {
        void tick()
      }
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      cancelled = true
      if (interval) window.clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [person])

  const view = useMemo(() => {
    if (!person) {
      return { cls: styles.idle, label: 'Index n/a', title: 'Select a person in Settings.' }
    }
    if (!status) {
      return { cls: styles.idle, label: 'Index ?', title: 'No index status yet.' }
    }
    if (status.in_progress || status.pending) {
      const reason = status.last_reason ? ` (${status.last_reason})` : ''
      return { cls: styles.active, label: 'Indexing', title: `Indexing${reason}` }
    }
    if (status.last_error) {
      return { cls: styles.error, label: 'Index err', title: status.last_error }
    }
    if (status.last_success_at) {
      const started = Date.parse(status.last_success_at)
      if (!Number.isNaN(started)) {
        const age = Date.now() - started
        return { cls: styles.ok, label: `Indexed ${formatAge(age)}`, title: `Last success: ${status.last_success_at}` }
      }
    }
    return { cls: styles.idle, label: 'Index idle', title: 'Indexer has not completed a run yet.' }
  }, [person, status])

  return (
    <button
      type="button"
      className={`${styles.pill} ${view.cls}`}
      title={`${view.title}\nOpen Sync view`}
      onClick={() => navigate('/sync')}
    >
      <span className={styles.dot} />
      {view.label}
    </button>
  )
}

