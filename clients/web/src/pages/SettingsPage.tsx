import { useState, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { useTheme } from '../hooks/useTheme'
import { useAuth } from '../hooks/useAuth'
import { fetchEnv, saveEnv } from '../api/settings'
import styles from './SettingsPage.module.css'

export default function SettingsPage() {
  const { person, setPerson } = usePerson()
  const { theme, setTheme } = useTheme()
  const { logout } = useAuth()
  const [envContent, setEnvContent] = useState('')
  const [envStatus, setEnvStatus] = useState('')
  const [isSavingEnv, setIsSavingEnv] = useState(false)

  useEffect(() => {
    fetchEnv()
      .then(data => setEnvContent(data.content))
      .catch(err => setEnvStatus(`Failed to load .env: ${err instanceof Error ? err.message : err}`))
  }, [])

  const handleSaveEnv = async () => {
    if (isSavingEnv) return
    setIsSavingEnv(true)
    setEnvStatus('')
    try {
      await saveEnv({ content: envContent })
      setEnvStatus('Saved .env')
    } catch (err) {
      setEnvStatus(`Save failed: ${err instanceof Error ? err.message : err}`)
    } finally {
      setIsSavingEnv(false)
    }
  }

  return (
    <div className={styles.page}>
      <h2>Settings</h2>

      <section className={styles.section}>
        <h3>Person</h3>
        <div className={styles.options}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="person"
              value="sebastian"
              checked={person === 'sebastian'}
              onChange={() => setPerson('sebastian')}
            />
            Sebastian
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="person"
              value="petra"
              checked={person === 'petra'}
              onChange={() => setPerson('petra')}
            />
            Petra
          </label>
        </div>
        {!person && (
          <p className={styles.hint}>
            Please select a person to access your notes.
          </p>
        )}
      </section>

      <section className={styles.section}>
        <h3>Theme</h3>
        <div className={styles.options}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="theme"
              value="dark"
              checked={theme === 'dark'}
              onChange={() => setTheme('dark')}
            />
            Dark
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="theme"
              value="light"
              checked={theme === 'light'}
              onChange={() => setTheme('light')}
            />
            Light
          </label>
        </div>
      </section>

      <section className={styles.section}>
        <h3>Server .env</h3>
        <p className={styles.hint}>Edit environment variables stored on the server.</p>
        <textarea
          value={envContent}
          onChange={e => setEnvContent(e.target.value)}
          className={styles.envEditor}
          rows={8}
          spellCheck={false}
        />
        <div className={styles.envActions}>
          <button onClick={handleSaveEnv} disabled={isSavingEnv}>
            {isSavingEnv ? 'Saving...' : 'Save'}
          </button>
          {envStatus && <span className={styles.envStatus}>{envStatus}</span>}
        </div>
      </section>

      <section className={styles.section}>
        <h3>Account</h3>
        <button onClick={logout} className="danger">
          Logout
        </button>
      </section>
    </div>
  )
}
