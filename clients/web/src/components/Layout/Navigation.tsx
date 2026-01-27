import { NavLink } from 'react-router-dom'
import styles from './Layout.module.css'

const navItems = [
  { to: '/daily', label: 'Daily' },
  { to: '/files', label: 'Files' },
  { to: '/sleep', label: 'Sleep' },
  { to: '/claude', label: 'Claude' },
  { to: '/noise', label: 'Noise' },
  { to: '/settings', label: 'Settings' },
]

export default function Navigation() {
  return (
    <nav className={styles.nav}>
      {navItems.map(item => (
        <NavLink
          key={item.to}
          to={item.to}
          className={({ isActive }) =>
            `${styles.navLink} ${isActive ? styles.navLinkActive : ''}`
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
