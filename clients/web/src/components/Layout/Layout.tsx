import { ReactNode } from 'react'
import Header from './Header'
import Navigation from './Navigation'
import styles from './Layout.module.css'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div className={styles.layout}>
      <Header />
      <Navigation />
      <main className={styles.main}>
        <div className={styles.container}>
          {children}
        </div>
      </main>
    </div>
  )
}
