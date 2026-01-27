import { usePerson } from '../hooks/usePerson'
import { useTheme } from '../hooks/useTheme'
import { useAuth } from '../hooks/useAuth'
import styles from './SettingsPage.module.css'

export default function SettingsPage() {
  const { person, setPerson } = usePerson()
  const { theme, setTheme } = useTheme()
  const { logout } = useAuth()

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
        <h3>Account</h3>
        <button onClick={logout} className="danger">
          Logout
        </button>
      </section>
    </div>
  )
}
