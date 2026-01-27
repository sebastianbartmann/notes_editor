import { useState } from 'react'
import styles from './Editor.module.css'

interface EditorProps {
  content: string
  onSave: (content: string) => void
  onCancel: () => void
}

export default function Editor({ content, onSave, onCancel }: EditorProps) {
  const [value, setValue] = useState(content)

  const handleSave = () => {
    onSave(value)
  }

  return (
    <div className={styles.editor}>
      <textarea
        value={value}
        onChange={e => setValue(e.target.value)}
        className={styles.textarea}
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
