import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import NoteView, { parseLine } from './NoteView'

describe('parseLine', () => {
  describe('headings', () => {
    it('parses H1 heading', () => {
      const result = parseLine('# Hello World', 1)
      expect(result).toEqual({
        type: 'H1',
        content: 'Hello World',
        isPinned: false,
        lineNumber: 1,
      })
    })

    it('parses H2 heading', () => {
      const result = parseLine('## Section Title', 5)
      expect(result).toEqual({
        type: 'H2',
        content: 'Section Title',
        isPinned: false,
        lineNumber: 5,
      })
    })

    it('parses H3 heading', () => {
      const result = parseLine('### Subsection', 10)
      expect(result).toEqual({
        type: 'H3',
        content: 'Subsection',
        isPinned: false,
        lineNumber: 10,
      })
    })

    it('parses H4 heading', () => {
      const result = parseLine('#### Deep Heading', 15)
      expect(result).toEqual({
        type: 'H4',
        content: 'Deep Heading',
        isPinned: false,
        lineNumber: 15,
      })
    })

    it('parses H5 heading', () => {
      const result = parseLine('##### Very Deep', 20)
      expect(result).toEqual({
        type: 'H5',
        content: 'Very Deep',
        isPinned: false,
        lineNumber: 20,
      })
    })

    it('parses H6 heading', () => {
      const result = parseLine('###### Deepest', 25)
      expect(result).toEqual({
        type: 'H6',
        content: 'Deepest',
        isPinned: false,
        lineNumber: 25,
      })
    })

    it('detects pinned marker on H3', () => {
      const result = parseLine('### 14:30 <pinned>', 8)
      expect(result).toEqual({
        type: 'H3',
        content: '14:30',
        isPinned: true,
        lineNumber: 8,
      })
    })

    it('detects pinned marker case-insensitive', () => {
      const result = parseLine('### Note <PINNED>', 9)
      expect(result).toEqual({
        type: 'H3',
        content: 'Note',
        isPinned: true,
        lineNumber: 9,
      })
    })

    it('does not detect pinned on non-H3 headings', () => {
      const result = parseLine('## Section <pinned>', 3)
      expect(result.isPinned).toBe(false)
      expect(result.content).toBe('Section <pinned>')
    })
  })

  describe('tasks', () => {
    it('parses unchecked task', () => {
      const result = parseLine('- [ ] Buy groceries', 1)
      expect(result).toEqual({
        type: 'TASK',
        content: 'Buy groceries',
        checked: false,
        lineNumber: 1,
      })
    })

    it('parses checked task with lowercase x', () => {
      const result = parseLine('- [x] Completed task', 2)
      expect(result).toEqual({
        type: 'TASK',
        content: 'Completed task',
        checked: true,
        lineNumber: 2,
      })
    })

    it('parses checked task with uppercase X', () => {
      const result = parseLine('- [X] Also completed', 3)
      expect(result).toEqual({
        type: 'TASK',
        content: 'Also completed',
        checked: true,
        lineNumber: 3,
      })
    })

    it('parses empty task', () => {
      const result = parseLine('- [ ] ', 4)
      expect(result).toEqual({
        type: 'TASK',
        content: '',
        checked: false,
        lineNumber: 4,
      })
    })

    it('parses task with leading whitespace', () => {
      const result = parseLine('  - [ ] Indented task', 5)
      expect(result).toEqual({
        type: 'TASK',
        content: 'Indented task',
        checked: false,
        lineNumber: 5,
      })
    })

    it('parses task with special characters', () => {
      const result = parseLine('- [ ] Task with <html> & "quotes"', 6)
      expect(result).toEqual({
        type: 'TASK',
        content: 'Task with <html> & "quotes"',
        checked: false,
        lineNumber: 6,
      })
    })
  })

  describe('empty lines', () => {
    it('parses empty string', () => {
      const result = parseLine('', 1)
      expect(result).toEqual({
        type: 'EMPTY',
        content: '',
        lineNumber: 1,
      })
    })

    it('parses whitespace-only line', () => {
      const result = parseLine('   ', 2)
      expect(result).toEqual({
        type: 'EMPTY',
        content: '',
        lineNumber: 2,
      })
    })

    it('parses tabs as empty', () => {
      const result = parseLine('\t\t', 3)
      expect(result).toEqual({
        type: 'EMPTY',
        content: '',
        lineNumber: 3,
      })
    })
  })

  describe('plain text', () => {
    it('parses regular text', () => {
      const result = parseLine('This is plain text', 1)
      expect(result).toEqual({
        type: 'TEXT',
        content: 'This is plain text',
        lineNumber: 1,
      })
    })

    it('parses text that looks almost like heading', () => {
      const result = parseLine('#NoSpace', 2)
      expect(result).toEqual({
        type: 'TEXT',
        content: '#NoSpace',
        lineNumber: 2,
      })
    })

    it('parses text that looks almost like task', () => {
      const result = parseLine('- Not a task', 3)
      expect(result).toEqual({
        type: 'TEXT',
        content: '- Not a task',
        lineNumber: 3,
      })
    })

    it('preserves text with special characters', () => {
      const result = parseLine('Text with <angle> & "special" chars', 4)
      expect(result).toEqual({
        type: 'TEXT',
        content: 'Text with <angle> & "special" chars',
        lineNumber: 4,
      })
    })
  })

  describe('line number handling', () => {
    it('preserves line numbers correctly', () => {
      expect(parseLine('# Test', 1).lineNumber).toBe(1)
      expect(parseLine('# Test', 100).lineNumber).toBe(100)
      expect(parseLine('# Test', 999).lineNumber).toBe(999)
    })
  })
})

describe('NoteView component', () => {
  it('renders headings correctly', () => {
    const content = `# Main Title
## Section
### Subsection`

    render(<NoteView content={content} />)

    expect(screen.getByText('Main Title')).toBeInTheDocument()
    expect(screen.getByText('Section')).toBeInTheDocument()
    expect(screen.getByText('Subsection')).toBeInTheDocument()
  })

  it('renders tasks with checkboxes', () => {
    const content = `- [ ] Unchecked
- [x] Checked`

    render(<NoteView content={content} />)

    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(2)
    expect(checkboxes[0]).not.toBeChecked()
    expect(checkboxes[1]).toBeChecked()
  })

  it('calls onTaskToggle with correct line number', () => {
    const onTaskToggle = vi.fn()
    const content = `# Title
- [ ] Task on line 2`

    render(<NoteView content={content} onTaskToggle={onTaskToggle} />)

    const checkbox = screen.getByRole('checkbox')
    fireEvent.click(checkbox)

    expect(onTaskToggle).toHaveBeenCalledWith(2)
  })

  it('renders empty lines with non-breaking space', () => {
    const content = `Line 1

Line 3`

    const { container } = render(<NoteView content={content} />)

    const lines = container.querySelectorAll('[class*="line"]')
    expect(lines).toHaveLength(3)
    // The middle line should have &nbsp; (rendered as \u00A0)
    expect(lines[1].textContent).toBe('\u00A0')
  })

  it('renders plain text', () => {
    const content = 'Just some plain text'

    render(<NoteView content={content} />)

    expect(screen.getByText('Just some plain text')).toBeInTheDocument()
  })

  it('renders unpin button for pinned H3 headings', () => {
    const onUnpin = vi.fn()
    const content = '### 14:30 <pinned>'

    render(<NoteView content={content} onUnpin={onUnpin} />)

    const unpinButton = screen.getByRole('button', { name: 'Unpin' })
    expect(unpinButton).toBeInTheDocument()
  })

  it('calls onUnpin with correct line number', () => {
    const onUnpin = vi.fn()
    const content = `# Title
### Note <pinned>`

    render(<NoteView content={content} onUnpin={onUnpin} />)

    const unpinButton = screen.getByRole('button', { name: 'Unpin' })
    fireEvent.click(unpinButton)

    expect(onUnpin).toHaveBeenCalledWith(2)
  })

  it('does not render unpin button without onUnpin callback', () => {
    const content = '### Note <pinned>'

    render(<NoteView content={content} />)

    expect(screen.queryByRole('button', { name: 'Unpin' })).not.toBeInTheDocument()
  })

  it('escapes HTML in rendered content', () => {
    const content = '- [ ] Task with <script>alert(1)</script>'

    const { container } = render(<NoteView content={content} />)

    // Should not have actual script tag
    expect(container.querySelector('script')).toBeNull()
    // Should have escaped text
    expect(screen.getByText(/Task with/)).toBeInTheDocument()
  })

  it('handles complex note structure', () => {
    const content = `# daily 2024-01-15
## todos
### work
- [ ] Code review
- [x] Write tests
### priv
- [ ] Buy groceries
## custom notes
### 14:30 <pinned>
Important meeting reminder
### 16:00
Regular note`

    const onTaskToggle = vi.fn()
    const onUnpin = vi.fn()

    render(
      <NoteView
        content={content}
               onTaskToggle={onTaskToggle}
        onUnpin={onUnpin}
      />
    )

    // Verify headings
    expect(screen.getByText('daily 2024-01-15')).toBeInTheDocument()
    expect(screen.getByText('todos')).toBeInTheDocument()
    expect(screen.getByText('work')).toBeInTheDocument()
    expect(screen.getByText('priv')).toBeInTheDocument()
    expect(screen.getByText('custom notes')).toBeInTheDocument()

    // Verify tasks
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(3)

    // Verify pinned entry
    expect(screen.getByRole('button', { name: 'Unpin' })).toBeInTheDocument()
    expect(screen.getByText('14:30')).toBeInTheDocument()
  })
})
