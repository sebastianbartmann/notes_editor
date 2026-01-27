import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { PersonProvider, PersonContext } from './PersonContext'
import { useContext } from 'react'

function TestComponent() {
  const person = useContext(PersonContext)
  if (!person) return <div>No context</div>

  return (
    <div>
      <div data-testid="person">{person.person ?? 'null'}</div>
      <button onClick={() => person.setPerson('sebastian')}>Set Sebastian</button>
      <button onClick={() => person.setPerson('petra')}>Set Petra</button>
      <button onClick={() => person.setPerson(null)}>Clear</button>
    </div>
  )
}

describe('PersonContext', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to null when localStorage is empty', () => {
    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    expect(screen.getByTestId('person')).toHaveTextContent('null')
  })

  it('initializes from localStorage sebastian value', () => {
    localStorage.setItem('notes_person', 'sebastian')

    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    expect(screen.getByTestId('person')).toHaveTextContent('sebastian')
  })

  it('initializes from localStorage petra value', () => {
    localStorage.setItem('notes_person', 'petra')

    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    expect(screen.getByTestId('person')).toHaveTextContent('petra')
  })

  it('ignores invalid localStorage values and defaults to null', () => {
    localStorage.setItem('notes_person', 'invalid-person')

    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    expect(screen.getByTestId('person')).toHaveTextContent('null')
  })

  it('setPerson changes person and persists to localStorage', async () => {
    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Sebastian' }))
    })

    expect(screen.getByTestId('person')).toHaveTextContent('sebastian')
    expect(localStorage.getItem('notes_person')).toBe('sebastian')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Petra' }))
    })

    expect(screen.getByTestId('person')).toHaveTextContent('petra')
    expect(localStorage.getItem('notes_person')).toBe('petra')
  })

  it('setPerson(null) clears person and removes from localStorage', async () => {
    localStorage.setItem('notes_person', 'sebastian')

    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    expect(screen.getByTestId('person')).toHaveTextContent('sebastian')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
    })

    expect(screen.getByTestId('person')).toHaveTextContent('null')
    expect(localStorage.getItem('notes_person')).toBeNull()
  })

  it('persists when switching between valid persons', async () => {
    render(
      <PersonProvider>
        <TestComponent />
      </PersonProvider>
    )

    // Set to sebastian
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Sebastian' }))
    })
    expect(localStorage.getItem('notes_person')).toBe('sebastian')

    // Switch to petra
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Set Petra' }))
    })
    expect(localStorage.getItem('notes_person')).toBe('petra')

    // Clear
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
    })
    expect(localStorage.getItem('notes_person')).toBeNull()
  })
})
