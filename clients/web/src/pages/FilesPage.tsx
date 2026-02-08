import { useState, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { usePerson } from '../hooks/usePerson'
import { listFiles, readFile, saveFile, createFile, deleteFile } from '../api/files'
import { toggleTodo } from '../api/todos'
import { unpinEntry } from '../api/files'
import type { FileEntry } from '../api/types'
import FileTree from '../components/FileTree/FileTree'
import NoteView from '../components/NoteView/NoteView'
import Editor from '../components/Editor/Editor'
import styles from './FilesPage.module.css'

export default function FilesPage() {
  const { person } = usePerson()
  const location = useLocation()
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [content, setContent] = useState('')
  const [isEditing, setIsEditing] = useState(false)
  const [newFileName, setNewFileName] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!person) return
    loadFiles()
  }, [person])

  useEffect(() => {
    const pathParam = location.pathname.replace('/files/', '').replace('/files', '')
    if (pathParam && pathParam !== selectedPath) {
      handleSelectFile(decodeURIComponent(pathParam))
    }
  }, [location.pathname])

  const loadFiles = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await listFiles('.')
      setEntries(data.entries)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load files')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectFile = async (path: string) => {
    setSelectedPath(path)
    setIsEditing(false)
    try {
      const data = await readFile(path)
      setContent(data.content)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to read file')
    }
  }

  const handleSave = async (newContent: string) => {
    if (!selectedPath) return
    try {
      await saveFile({ path: selectedPath, content: newContent })
      setContent(newContent)
      setIsEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const handleCreate = async () => {
    if (!newFileName.trim()) return
    try {
      await createFile({ path: newFileName })
      setNewFileName('')
      await loadFiles()
      await handleSelectFile(newFileName)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create file')
    }
  }

  const handleDelete = async () => {
    if (!selectedPath) return
    if (!confirm(`Delete ${selectedPath}?`)) return
    try {
      await deleteFile({ path: selectedPath })
      setSelectedPath(null)
      setContent('')
      await loadFiles()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete file')
    }
  }

  const handleTaskToggle = async (line: number) => {
    if (!selectedPath) return
    try {
      await toggleTodo({ path: selectedPath, line })
      const data = await readFile(selectedPath)
      setContent(data.content)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle task')
    }
  }

  const handleUnpin = async (line: number) => {
    if (!selectedPath) return
    try {
      await unpinEntry({ path: selectedPath, line })
      const data = await readFile(selectedPath)
      setContent(data.content)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpin')
    }
  }

  if (!person) {
    return (
      <div className={styles.message}>
        Please select a person in Settings first.
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.sidebar}>
        <div className={styles.createForm}>
          <input
            type="text"
            value={newFileName}
            onChange={e => setNewFileName(e.target.value)}
            placeholder="new-file.md"
            className={styles.createInput}
          />
          <button onClick={handleCreate} disabled={!newFileName.trim()}>
            Create
          </button>
        </div>
        {loading ? (
          <div className={styles.message}>Loading...</div>
        ) : (
          <FileTree
            entries={entries}
            selectedPath={selectedPath}
            onSelect={handleSelectFile}
          />
        )}
      </div>

      <div className={styles.content}>
        {error && <p className={styles.error}>{error}</p>}

        {selectedPath ? (
          <>
            <div className={styles.fileHeader}>
              <span className={styles.filePath}>{selectedPath}</span>
              <div className={styles.fileActions}>
                {!isEditing && (
                  <button onClick={() => setIsEditing(true)}>Edit</button>
                )}
                <button onClick={handleDelete} className="danger">
                  Delete
                </button>
              </div>
            </div>
            {isEditing ? (
              <Editor
                content={content}
                onSave={handleSave}
                onCancel={() => setIsEditing(false)}
              />
            ) : (
              <NoteView
                content={content}
                onTaskToggle={handleTaskToggle}
                onUnpin={handleUnpin}
              />
            )}
          </>
        ) : (
          <div className={styles.placeholder}>
            Select a file to view its contents
          </div>
        )}
      </div>
    </div>
  )
}
