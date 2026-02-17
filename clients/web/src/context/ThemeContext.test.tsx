import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { ThemeProvider, ThemeContext } from './ThemeContext'
import { useContext } from 'react'

function TestComponent() {
  const theme = useContext(ThemeContext)
  if (!theme) return <div>No context</div>

  return (
    <div>
      <div data-testid="theme">{theme.theme}</div>
      <div data-testid="text-scale">{theme.textScale}</div>
      <button onClick={() => theme.setTheme('dark')}>Set Dark</button>
      <button onClick={() => theme.setTheme('light')}>Set Light</button>
      <button onClick={theme.toggleTheme}>Toggle</button>
      <button onClick={() => theme.stepTextScale(-1)}>Scale Down</button>
      <button onClick={() => theme.stepTextScale(1)}>Scale Up</button>
      <button onClick={theme.resetTextScale}>Scale Reset</button>
    </div>
  )
}

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear()
    document.body.className = ''
    document.documentElement.style.removeProperty('--text-scale')
  })

  it('defaults to dark theme when localStorage is empty', () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
    expect(document.body.classList.contains('theme-light')).toBe(false)
  })

  it('initializes from localStorage dark value', () => {
    localStorage.setItem('notes_theme', 'dark')

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
    expect(document.body.classList.contains('theme-light')).toBe(false)
  })

  it('initializes from localStorage light value', () => {
    localStorage.setItem('notes_theme', 'light')

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme')).toHaveTextContent('light')
    expect(document.body.classList.contains('theme-light')).toBe(true)
  })

  it('ignores invalid localStorage values and defaults to dark', () => {
    localStorage.setItem('notes_theme', 'invalid')

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
  })

  it('setTheme changes theme and persists to localStorage', async () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Light' }))
    })

    expect(screen.getByTestId('theme')).toHaveTextContent('light')
    expect(localStorage.getItem('notes_theme')).toBe('light')
    expect(document.body.classList.contains('theme-light')).toBe(true)

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Dark' }))
    })

    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
    expect(localStorage.getItem('notes_theme')).toBe('dark')
    expect(document.body.classList.contains('theme-light')).toBe(false)
  })

  it('toggleTheme switches between dark and light', async () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    // Initially dark
    expect(screen.getByTestId('theme')).toHaveTextContent('dark')

    // Toggle to light
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Toggle' }))
    })
    expect(screen.getByTestId('theme')).toHaveTextContent('light')
    expect(document.body.classList.contains('theme-light')).toBe(true)

    // Toggle back to dark
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Toggle' }))
    })
    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
    expect(document.body.classList.contains('theme-light')).toBe(false)
  })

  it('updates body class when theme changes', async () => {
    localStorage.setItem('notes_theme', 'dark')

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    // Initially no theme-light class
    expect(document.body.classList.contains('theme-light')).toBe(false)

    // Switch to light
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Light' }))
    })

    // Now has theme-light class
    expect(document.body.classList.contains('theme-light')).toBe(true)

    // Switch back to dark
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Dark' }))
    })

    // Class removed
    expect(document.body.classList.contains('theme-light')).toBe(false)
  })

  it('defaults text scale to 1 and persists it', () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('text-scale')).toHaveTextContent('1')
    expect(localStorage.getItem('notes_text_scale')).toBe('1')
    expect(document.documentElement.style.getPropertyValue('--text-scale')).toBe('1')
  })

  it('steps and resets text scale', async () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Scale Up' }))
    })

    expect(screen.getByTestId('text-scale')).toHaveTextContent('1.05')
    expect(localStorage.getItem('notes_text_scale')).toBe('1.05')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Scale Reset' }))
    })

    expect(screen.getByTestId('text-scale')).toHaveTextContent('1')
    expect(localStorage.getItem('notes_text_scale')).toBe('1')
  })
})
