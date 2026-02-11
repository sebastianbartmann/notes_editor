import { useEffect, useMemo, useState } from 'react'
import { fetchSyncStatus, type SyncStatus } from '../../api/sync'
import { usePerson } from '../../hooks/usePerson'
import { useNavigate } from 'react-router-dom'
import styles from './SyncIndicator.module.css'

function formatAge(ms: number): string {
  if (ms < 5_000) return 'just now'
  if (ms < 60_000) return `${Math.round(ms / 1_000)}s`
  if (ms < 60 * 60_000) return `${Math.round(ms / 60_000)}m`
  return `${Math.round(ms / (60 * 60_000))}h`
}

export default function SyncIndicator() {
  const { person } = usePerson()
  const navigate = useNavigate()
  const [status, setStatus] = useState<SyncStatus | null>(null)

  useEffect(() => {
    if (!person) return

    let cancelled = false
    let interval: number | undefined

    const tick = async () => {
      try {
        const s = await fetchSyncStatus()
        if (cancelled) return
        setStatus(s)
      } catch {
        // Best-effort indicator: ignore transient failures so we don't flicker.
      }
    }

    tick()
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
      return { cls: styles.notSynced, label: 'Not synced (no person)', title: 'Select a person in Settings.' }
    }

    // Two-state indicator:
    // - Synced: recent successful pull + nothing pending/in-progress + no recent errors
    // - Not synced: otherwise, with a reason hint
    if (!status) {
      return { cls: styles.notSynced, label: 'Not synced (no status)', title: 'No sync status yet.' }
    }

    const now = Date.now()
    const lastPullMs = status.last_pull_at ? Date.parse(status.last_pull_at) : NaN
    const lastPullAge = Number.isNaN(lastPullMs) ? Infinity : now - lastPullMs
    const lastErrAtMs = status.last_error_at ? Date.parse(status.last_error_at) : NaN
    const lastErrAge = Number.isNaN(lastErrAtMs) ? Infinity : now - lastErrAtMs

    const pending = status.in_progress || status.pending_pull || status.pending_push
    const recentError = Boolean(status.last_error) && lastErrAge <= 10 * 60_000
    const stale = lastPullAge > 2 * 60_000

    const reasons: string[] = []
    if (!status.last_pull_at) reasons.push('never pulled')
    else if (stale) reasons.push(`stale ${formatAge(lastPullAge)}`)
    if (pending) reasons.push('syncing')
    if (recentError) reasons.push('recent error')

    const isSynced = !pending && !recentError && !stale && Boolean(status.last_pull_at)
    if (isSynced) {
      return {
        cls: styles.synced,
        label: `Synced ${formatAge(lastPullAge)}`,
        title: `Last pull: ${status.last_pull_at}`,
      }
    }

    const hint = reasons.length ? reasons.join(', ') : 'unknown'
    const title = status.last_error ? `${hint}\n${status.last_error}` : hint
    return { cls: styles.notSynced, label: `Not synced (${hint})`, title }
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
