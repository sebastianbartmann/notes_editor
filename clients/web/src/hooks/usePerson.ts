import { useContext } from 'react'
import { PersonContext } from '../context/PersonContext'

export function usePerson() {
  const context = useContext(PersonContext)
  if (!context) {
    throw new Error('usePerson must be used within a PersonProvider')
  }
  return context
}
