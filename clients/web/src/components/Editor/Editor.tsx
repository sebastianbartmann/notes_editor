import { useState } from 'react'
import styles from './Editor.module.css'

interface EditorProps {
  content: string
  onSave: (content: string) => void
  onCancel: () => void
  className?: string
  textareaClassName?: string
}

export default function Editor({
  content,
  onSave,
  onCancel,
  className,
  textareaClassName,
}: EditorProps) {
  const [value, setValue] = useState(content)

  const handleSave = () => {
    onSave(value)
  }

  return (
    <div className={`${styles.editor} ${className ?? ''}`.trim()}>
      <textarea
        value={value}
        onChange={e => setValue(e.target.value)}
        className={`${styles.textarea} ${textareaClassName ?? ''}`.trim()}
        autoFocus
      />
      <div className={styles.actions}>
        <button onClick={onCancel} className="ghost">
          Cancel
        </button>
        <button onClick={handleSave}>Save</button>
      </div>
    </div>
  )
}
