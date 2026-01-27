import { createContext, useState, useEffect, ReactNode } from 'react'

type Person = 'sebastian' | 'petra' | null

interface PersonContextType {
  person: Person
  setPerson: (person: Person) => void
}

export const PersonContext = createContext<PersonContextType | null>(null)

const STORAGE_KEY = 'notes_person'

export function PersonProvider({ children }: { children: ReactNode }) {
  const [person, setPersonState] = useState<Person>(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'sebastian' || stored === 'petra') {
      return stored
    }
    return null
  })

  useEffect(() => {
    if (person) {
      localStorage.setItem(STORAGE_KEY, person)
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [person])

  const setPerson = (newPerson: Person) => {
    setPersonState(newPerson)
  }

  return (
    <PersonContext.Provider value={{ person, setPerson }}>
      {children}
    </PersonContext.Provider>
  )
}
