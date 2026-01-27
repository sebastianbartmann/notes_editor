import { useState, useEffect } from 'react'
import { listFiles } from '../../api/files'
import type { FileEntry } from '../../api/types'
import styles from './FileTree.module.css'

interface FileTreeProps {
  entries: FileEntry[]
  selectedPath: string | null
  onSelect: (path: string) => void
}

interface TreeItemProps {
  entry: FileEntry
  selectedPath: string | null
  onSelect: (path: string) => void
  depth: number
}

function TreeItem({ entry, selectedPath, onSelect, depth }: TreeItemProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [children, setChildren] = useState<FileEntry[]>([])
  const [loading, setLoading] = useState(false)

  const isSelected = entry.path === selectedPath

  useEffect(() => {
    if (isExpanded && entry.is_dir && children.length === 0) {
      loadChildren()
    }
  }, [isExpanded])

  const loadChildren = async () => {
    setLoading(true)
    try {
      const data = await listFiles(entry.path)
      setChildren(data.entries)
    } catch (err) {
      console.error('Failed to load directory:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleClick = () => {
    if (entry.is_dir) {
      setIsExpanded(!isExpanded)
    } else {
      onSelect(entry.path)
    }
  }

  return (
    <div className={styles.item}>
      <div
        className={`${styles.row} ${isSelected ? styles.selected : ''}`}
        style={{ paddingLeft: depth * 14 + 6 }}
        onClick={handleClick}
      >
        {entry.is_dir && (
          <span className={`${styles.toggle} ${isExpanded ? styles.expanded : ''}`}>
            {isExpanded ? '▼' : '▶'}
          </span>
        )}
        <span className={entry.is_dir ? styles.folder : styles.file}>
          {entry.name}
        </span>
      </div>
      {isExpanded && entry.is_dir && (
        <div className={styles.children}>
          {loading ? (
            <div className={styles.loading} style={{ paddingLeft: (depth + 1) * 14 + 6 }}>
              Loading...
            </div>
          ) : (
            children.map(child => (
              <TreeItem
                key={child.path}
                entry={child}
                selectedPath={selectedPath}
                onSelect={onSelect}
                depth={depth + 1}
              />
            ))
          )}
        </div>
      )}
    </div>
  )
}

export default function FileTree({ entries, selectedPath, onSelect }: FileTreeProps) {
  return (
    <div className={styles.tree}>
      {entries.map(entry => (
        <TreeItem
          key={entry.path}
          entry={entry}
          selectedPath={selectedPath}
          onSelect={onSelect}
          depth={0}
        />
      ))}
    </div>
  )
}
