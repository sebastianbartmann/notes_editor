// Daily API types
export interface DailyResponse {
  date: string
  path: string
  content: string
}

export interface SaveResponse {
  success: boolean
  message: string
}

export interface SaveDailyRequest {
  path: string
  content: string
}

export interface AppendRequest {
  path: string
  text: string
  pinned?: boolean
}

// Todo API types
export interface AddTodoRequest {
  category: 'work' | 'priv'
  text?: string
}

export interface ToggleTodoRequest {
  path: string
  line: number
}

// Files API types
export interface FileEntry {
  name: string
  path: string
  is_dir: boolean
}

export interface ListFilesResponse {
  entries: FileEntry[]
}

export interface ReadFileResponse {
  path: string
  content: string
}

export interface CreateFileRequest {
  path: string
}

export interface SaveFileRequest {
  path: string
  content: string
}

export interface DeleteFileRequest {
  path: string
}

export interface UnpinRequest {
  path: string
  line: number
}

// Sleep API types
export interface SleepEntry {
  line: number
  date: string
  child: string
  time: string
  status: string
}

export interface SleepTimesResponse {
  entries: SleepEntry[]
}

export interface AppendSleepRequest {
  child: string
  time: string
  status: 'eingeschlafen' | 'aufgewacht'
}

export interface DeleteSleepRequest {
  line: number
}

// Claude API types
export interface ChatRequest {
  message: string
  session_id?: string
}

export interface ChatResponse {
  success: boolean
  session_id: string
  response: string
  history: ChatMessage[]
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface ClearSessionRequest {
  session_id: string
}

export interface HistoryResponse {
  success: boolean
  history: ChatMessage[]
}

// Stream event types
export interface StreamEvent {
  type: 'text' | 'tool_use' | 'session' | 'ping' | 'error' | 'done'
  delta?: string
  name?: string
  input?: Record<string, unknown>
  session_id?: string
  message?: string
}

// Settings API types
export interface EnvResponse {
  success: boolean
  content: string
}

export interface UpdateEnvRequest {
  content: string
}
