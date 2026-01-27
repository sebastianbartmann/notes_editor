import styles from './NoteView.module.css'

interface NoteViewProps {
  content: string
  path: string
  onTaskToggle?: (line: number) => void
  onUnpin?: (line: number) => void
}

export type LineType = 'H1' | 'H2' | 'H3' | 'H4' | 'H5' | 'H6' | 'TASK' | 'TEXT' | 'EMPTY'

export interface ParsedLine {
  type: LineType
  content: string
  checked?: boolean
  isPinned?: boolean
  lineNumber: number
}

const TASK_REGEX = /^\s*-\s*\[([ xX])\]\s*(.*)$/
const HEADING_REGEX = /^(#{1,6})\s+(.*)$/
const PINNED_REGEX = /<pinned>/i

export function parseLine(line: string, lineNumber: number): ParsedLine {
  // Check for task
  const taskMatch = line.match(TASK_REGEX)
  if (taskMatch) {
    return {
      type: 'TASK',
      content: taskMatch[2],
      checked: taskMatch[1].toLowerCase() === 'x',
      lineNumber,
    }
  }

  // Check for heading
  const headingMatch = line.match(HEADING_REGEX)
  if (headingMatch) {
    const level = headingMatch[1].length as 1 | 2 | 3 | 4 | 5 | 6
    const headingType = `H${level}` as LineType
    const content = headingMatch[2]
    const isPinned = level === 3 && PINNED_REGEX.test(content)
    // Only strip <pinned> marker from H3 headings (where pinned is valid)
    const displayContent = isPinned ? content.replace(PINNED_REGEX, '').trim() : content
    return {
      type: headingType,
      content: displayContent,
      isPinned,
      lineNumber,
    }
  }

  // Check for empty
  if (!line.trim()) {
    return {
      type: 'EMPTY',
      content: '',
      lineNumber,
    }
  }

  // Default to text
  return {
    type: 'TEXT',
    content: line,
    lineNumber,
  }
}

export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

export default function NoteView({
  content,
  path: _path,
  onTaskToggle,
  onUnpin,
}: NoteViewProps) {
  const lines = content.split('\n')
  const parsed = lines.map((line, i) => parseLine(line, i + 1))

  return (
    <div className={styles.noteView}>
      {parsed.map((line, i) => (
        <div key={i} className={styles.line}>
          {line.type === 'EMPTY' ? (
            <span>&nbsp;</span>
          ) : line.type === 'TASK' ? (
            <label className={`${styles.task} ${line.checked ? styles.taskDone : ''}`}>
              <input
                type="checkbox"
                checked={line.checked}
                onChange={() => onTaskToggle?.(line.lineNumber)}
                className={styles.checkbox}
              />
              <span>{escapeHtml(line.content)}</span>
            </label>
          ) : line.type.startsWith('H') ? (
            <div
              className={`${styles.heading} ${styles[line.type.toLowerCase()]} ${
                line.isPinned ? styles.pinned : ''
              }`}
            >
              <span>{escapeHtml(line.content)}</span>
              {line.isPinned && onUnpin && (
                <button
                  onClick={() => onUnpin(line.lineNumber)}
                  className={styles.unpinBtn}
                >
                  Unpin
                </button>
              )}
            </div>
          ) : (
            <span>{escapeHtml(line.content)}</span>
          )}
        </div>
      ))}
    </div>
  )
}
