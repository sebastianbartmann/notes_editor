import { usePerson } from '../../hooks/usePerson'
import { useTheme } from '../../hooks/useTheme'
import SyncIndicator from '../SyncIndicator/SyncIndicator'
import styles from './Layout.module.css'

export default function Header() {
  const { person } = usePerson()
  const { theme, toggleTheme } = useTheme()

  return (
    <header className={styles.header}>
      <h1 className={styles.title}>Notes Editor</h1>
      <div className={styles.headerActions}>
        <SyncIndicator />
        {person && <span className={styles.person}>{person}</span>}
        <button onClick={toggleTheme} className="ghost">
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
      </div>
    </header>
  )
}
