import { useEffect, useMemo, useState } from 'react'
import { fetchSyncStatus, type SyncStatus, syncIfStale } from '../../api/sync'
import { usePerson } from '../../hooks/usePerson'
import styles from './SyncIndicator.module.css'

function formatAge(ms: number): string {
  if (ms < 5_000) return 'just now'
  if (ms < 60_000) return `${Math.round(ms / 1_000)}s`
  if (ms < 60 * 60_000) return `${Math.round(ms / 60_000)}m`
  return `${Math.round(ms / (60 * 60_000))}h`
}

export default function SyncIndicator() {
  const { person } = usePerson()
  const [status, setStatus] = useState<SyncStatus | null>(null)
  const [offline, setOffline] = useState(false)

  useEffect(() => {
    if (!person) return

    let cancelled = false
    let interval: number | undefined

    const tick = async () => {
      try {
        const s = await fetchSyncStatus()
        if (cancelled) return
        setStatus(s)
        setOffline(false)
      } catch {
        if (cancelled) return
        setOffline(true)
      }
    }

    // App/page open: attempt a bounded sync if stale, then start polling.
    ;(async () => {
      await syncIfStale({ maxAgeMs: 30_000, timeoutMs: 2_000 })
      await tick()
      interval = window.setInterval(tick, 5_000)
    })()

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
      return { cls: styles.offline, label: 'No person', title: 'Select a person in Settings.' }
    }
    if (offline) {
      return { cls: styles.offline, label: 'Offline', title: 'Cannot reach server.' }
    }
    if (!status) {
      return { cls: styles.busy, label: 'Sync...', title: 'Loading sync status...' }
    }

    if (status.last_error) {
      return { cls: styles.err, label: 'Sync error', title: status.last_error }
    }

    if (status.in_progress || status.pending_pull || status.pending_push) {
      return { cls: styles.busy, label: 'Syncing', title: 'Git sync in progress.' }
    }

    const last = status.last_pull_at || status.last_push_at
    if (last) {
      const t = Date.parse(last)
      if (!Number.isNaN(t)) {
        const age = Date.now() - t
        return {
          cls: styles.ok,
          label: `Synced ${formatAge(age)}`,
          title: `Last sync activity: ${new Date(t).toLocaleString()}`,
        }
      }
    }

    return { cls: styles.ok, label: 'Synced', title: 'No recent sync timestamp yet.' }
  }, [offline, person, status])

  return (
    <span className={`${styles.pill} ${view.cls}`} title={view.title}>
      <span className={styles.dot} />
      {view.label}
    </span>
  )
}

