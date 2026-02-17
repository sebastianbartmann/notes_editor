import { createContext, useState, useEffect, ReactNode } from 'react'

type Theme = 'dark' | 'light'

interface ThemeContextType {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  textScale: number
  setTextScale: (scale: number) => void
  stepTextScale: (stepDelta: number) => void
  resetTextScale: () => void
}

export const ThemeContext = createContext<ThemeContextType | null>(null)

const STORAGE_KEY = 'notes_theme'
const TEXT_SCALE_STORAGE_KEY = 'notes_text_scale'
export const DEFAULT_TEXT_SCALE = 1
export const MIN_TEXT_SCALE = 0.85
export const MAX_TEXT_SCALE = 1.4
export const TEXT_SCALE_STEP = 0.05

function sanitizeTextScale(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_TEXT_SCALE
  return Math.min(MAX_TEXT_SCALE, Math.max(MIN_TEXT_SCALE, value))
}

function nextTextScale(current: number, stepDelta: number): number {
  const base = sanitizeTextScale(current)
  const stepped = base + (stepDelta * TEXT_SCALE_STEP)
  const snapped = Math.round(stepped / TEXT_SCALE_STEP) * TEXT_SCALE_STEP
  return sanitizeTextScale(snapped)
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'dark' || stored === 'light') {
      return stored
    }
    return 'dark'
  })
  const [textScale, setTextScaleState] = useState<number>(() => {
    const stored = localStorage.getItem(TEXT_SCALE_STORAGE_KEY)
    if (!stored) return DEFAULT_TEXT_SCALE
    const parsed = Number(stored)
    return sanitizeTextScale(parsed)
  })

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, theme)
    if (theme === 'light') {
      document.body.classList.add('theme-light')
    } else {
      document.body.classList.remove('theme-light')
    }
  }, [theme])

  useEffect(() => {
    const normalized = sanitizeTextScale(textScale)
    localStorage.setItem(TEXT_SCALE_STORAGE_KEY, normalized.toString())
    document.documentElement.style.setProperty('--text-scale', normalized.toString())
  }, [textScale])

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme)
  }

  const toggleTheme = () => {
    setThemeState(current => current === 'dark' ? 'light' : 'dark')
  }

  const setTextScale = (scale: number) => {
    setTextScaleState(sanitizeTextScale(scale))
  }

  const stepTextScale = (stepDelta: number) => {
    if (stepDelta === 0) return
    setTextScaleState(current => nextTextScale(current, stepDelta))
  }

  const resetTextScale = () => {
    setTextScaleState(DEFAULT_TEXT_SCALE)
  }

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggleTheme, textScale, setTextScale, stepTextScale, resetTextScale }}>
      {children}
    </ThemeContext.Provider>
  )
}
