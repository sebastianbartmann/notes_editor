import { useEffect, useState } from 'react'
import { fetchGitStatus, gitCommit, gitCommitPush, gitPull, gitPush, gitResetClean } from '../api/git'
import { fetchIndexStatus, type IndexStatus } from '../api/index'
import styles from './SyncPage.module.css'

export default function SyncPage() {
  const [statusOutput, setStatusOutput] = useState('')
  const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const refresh = async () => {
    setError('')
    try {
      const [data, idx] = await Promise.all([fetchGitStatus(), fetchIndexStatus()])
      setStatusOutput(data.output || '(clean)')
      setIndexStatus(idx)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load git status')
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  const runAction = async (
    label: string,
    action: () => Promise<{ message: string; output?: string }>,
    fallbackMessage?: string
  ) => {
    setBusy(true)
    setError('')
    try {
      const res = await action()
      setMessage(`${label}: ${res.message || fallbackMessage || 'Done'}`)
      if (res.output !== undefined) {
        setStatusOutput(res.output || '(clean)')
      } else {
        await refresh()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `${label} failed`)
      await refresh()
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className={styles.page}>
      <h2>Sync</h2>

      <div className={styles.actions}>
        <button onClick={() => runAction('Commit + Push', gitCommitPush)} disabled={busy}>
          Commit + Push
        </button>
        <button onClick={() => runAction('Commit', gitCommit)} disabled={busy}>
          Commit
        </button>
        <button onClick={() => runAction('Push', gitPush)} disabled={busy}>
          Push
        </button>
        <button onClick={() => runAction('Pull', gitPull)} disabled={busy}>
          Pull
        </button>
        <button
          onClick={() => {
            const ok = window.confirm('Discard all local changes and untracked files? This cannot be undone.')
            if (!ok) return
            void runAction('Reset/Clean', gitResetClean)
          }}
          disabled={busy}
        >
          Reset/Clean
        </button>
        <button
          onClick={() => runAction('Refresh', async () => {
            await refresh()
            return { message: 'Refreshed' }
          })}
          className="ghost"
          disabled={busy}
        >
          Refresh
        </button>
      </div>

      {message && <p className={styles.message}>{message}</p>}
      {error && <p className={styles.error}>{error}</p>}

      <div className={styles.indexStatus}>
        <h3>Index</h3>
        {!indexStatus && <p className={styles.message}>(no status)</p>}
        {indexStatus && (
          <ul className={styles.indexList}>
            <li>in_progress: {String(indexStatus.in_progress)}</li>
            <li>pending: {String(indexStatus.pending)}</li>
            <li>last_reason: {indexStatus.last_reason || '(none)'}</li>
            <li>last_started_at: {indexStatus.last_started_at || '(never)'}</li>
            <li>last_success_at: {indexStatus.last_success_at || '(never)'}</li>
            <li>last_error: {indexStatus.last_error || '(none)'}</li>
          </ul>
        )}
      </div>

      <div className={styles.statusBox}>
        <pre className={styles.statusText}>{statusOutput || '(empty)'}</pre>
      </div>
    </div>
  )
}
