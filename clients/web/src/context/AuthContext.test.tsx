import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { AuthProvider, AuthContext } from './AuthContext'
import { useContext } from 'react'

function TestComponent() {
  const auth = useContext(AuthContext)
  if (!auth) return <div>No context</div>

  return (
    <div>
      <div data-testid="token">{auth.token ?? 'null'}</div>
      <div data-testid="authenticated">{auth.isAuthenticated ? 'yes' : 'no'}</div>
      <button onClick={() => auth.login('test-token')}>Login</button>
      <button onClick={() => auth.logout()}>Logout</button>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('provides null token initially when localStorage is empty', () => {
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    expect(screen.getByTestId('token')).toHaveTextContent('null')
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no')
  })

  it('initializes from localStorage', () => {
    localStorage.setItem('notes_token', 'stored-token')

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    expect(screen.getByTestId('token')).toHaveTextContent('stored-token')
    expect(screen.getByTestId('authenticated')).toHaveTextContent('yes')
  })

  it('login updates token and persists to localStorage', async () => {
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    expect(screen.getByTestId('token')).toHaveTextContent('null')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Login' }))
    })

    expect(screen.getByTestId('token')).toHaveTextContent('test-token')
    expect(screen.getByTestId('authenticated')).toHaveTextContent('yes')
    expect(localStorage.getItem('notes_token')).toBe('test-token')
  })

  it('logout clears token and removes from localStorage', async () => {
    localStorage.setItem('notes_token', 'existing-token')

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    expect(screen.getByTestId('authenticated')).toHaveTextContent('yes')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Logout' }))
    })

    expect(screen.getByTestId('token')).toHaveTextContent('null')
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no')
    expect(localStorage.getItem('notes_token')).toBeNull()
  })

  it('isAuthenticated is derived from token presence', async () => {
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    // Initially not authenticated
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no')

    // After login, authenticated
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Login' }))
    })
    expect(screen.getByTestId('authenticated')).toHaveTextContent('yes')

    // After logout, not authenticated
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Logout' }))
    })
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no')
  })
})
