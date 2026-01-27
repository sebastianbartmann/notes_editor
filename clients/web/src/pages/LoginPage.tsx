import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import styles from './LoginPage.module.css'

export default function LoginPage() {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!token.trim()) {
      setError('Token is required')
      return
    }
    login(token.trim())
    navigate('/')
  }

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <h1 className={styles.title}>Notes Editor</h1>
        <form onSubmit={handleSubmit} className={styles.form}>
          <div className={styles.field}>
            <label htmlFor="token" className={styles.label}>
              Access Token
            </label>
            <input
              id="token"
              type="password"
              value={token}
              onChange={e => setToken(e.target.value)}
              placeholder="Enter your token"
              className={styles.input}
              autoFocus
            />
          </div>
          {error && <p className={styles.error}>{error}</p>}
          <button type="submit" className={styles.button}>
            Login
          </button>
        </form>
      </div>
    </div>
  )
}
