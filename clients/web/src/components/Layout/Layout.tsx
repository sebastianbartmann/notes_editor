import { ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import Header from './Header'
import Navigation from './Navigation'
import styles from './Layout.module.css'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const isFilesRoute = location.pathname === '/files' || location.pathname.startsWith('/files/')

  return (
    <div className={styles.layout}>
      <Header />
      <Navigation />
      <main className={styles.main}>
        <div className={`${styles.container} ${isFilesRoute ? styles.containerWide : ''}`}>
          {children}
        </div>
      </main>
    </div>
  )
}
