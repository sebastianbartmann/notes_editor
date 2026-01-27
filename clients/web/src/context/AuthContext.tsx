import { createContext, useState, useEffect, ReactNode } from 'react'

interface AuthState {
  token: string | null
  isAuthenticated: boolean
}

interface AuthContextType extends AuthState {
  login: (token: string) => void
  logout: () => void
}

export const AuthContext = createContext<AuthContextType | null>(null)

const STORAGE_KEY = 'notes_token'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem(STORAGE_KEY)
  })

  const isAuthenticated = token !== null

  useEffect(() => {
    if (token) {
      localStorage.setItem(STORAGE_KEY, token)
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [token])

  const login = (newToken: string) => {
    setToken(newToken)
  }

  const logout = () => {
    setToken(null)
  }

  return (
    <AuthContext.Provider value={{ token, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
