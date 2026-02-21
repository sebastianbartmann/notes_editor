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
  id: string
  date: string
  child: string
  time: string
  status: string
  notes?: string
  occurred_at?: string
}

export interface SleepTimesResponse {
  entries: SleepEntry[]
}

export interface AppendSleepRequest {
  child: string
  status: 'eingeschlafen' | 'aufgewacht'
  occurred_at: string
  notes?: string
}

export interface UpdateSleepRequest {
  id: string
  child: string
  status: 'eingeschlafen' | 'aufgewacht'
  occurred_at: string
  notes?: string
}

export interface DeleteSleepRequest {
  id: string
}

export interface ExportSleepResponse {
  success: boolean
  message: string
  path: string
}

export interface SleepNightSummary {
  night_date: string
  child: string
  duration_minutes: number
  bedtime: string
  wake_time: string
}

export interface SleepAverageSummary {
  days: number
  child: string
  average_bedtime: string
  average_wake_time: string
}

export interface SleepSummaryResponse {
  nights: SleepNightSummary[]
  averages: SleepAverageSummary[]
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

// Settings API types
export interface EnvResponse {
  content: string
}

export interface UpdateEnvRequest {
  content: string
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

// Agent API types
export interface AgentChatRequest {
  message: string
  session_id?: string
  action_id?: string
  confirm?: boolean
}

export interface AgentChatResponse {
  response: string
  session_id: string
  run_id: string
}

export interface AgentStreamEvent {
  type: 'start' | 'text' | 'tool_call' | 'tool_result' | 'status' | 'error' | 'usage' | 'done'
  session_id?: string
  run_id?: string
  seq?: number
  ts?: string
  delta?: string
  tool?: string
  args?: Record<string, unknown>
  ok?: boolean
  summary?: string
  message?: string
  usage?: AgentUsage
}

export interface AgentUsage {
  input_tokens?: number
  output_tokens?: number
  cache_read_tokens?: number
  cache_write_tokens?: number
  total_tokens?: number
  context_window?: number
  remaining_tokens?: number
}

export interface AgentConversationItem {
  type: 'message' | 'tool_call' | 'tool_result' | 'status' | 'error' | 'usage'
  role?: 'user' | 'assistant'
  content?: string
  session_id?: string
  run_id?: string
  seq?: number
  ts?: string
  tool?: string
  args?: Record<string, unknown>
  ok?: boolean
  summary?: string
  message?: string
  usage?: AgentUsage
}

export interface AgentConfig {
  runtime_mode: string
  prompt_path: string
  actions_path: string
  prompt: string
}

export interface AgentConfigUpdate {
  runtime_mode?: string
  prompt?: string
}

export interface AgentActionMetadata {
  requires_confirmation: boolean
  max_steps?: number
}

export interface AgentAction {
  id: string
  label: string
  path: string
  metadata: AgentActionMetadata
}

export interface AgentActionsResponse {
  actions: AgentAction[]
}

export interface AgentGatewayHealth {
  url: string
  configured: boolean
  reachable: boolean
  healthy: boolean
  mode?: string
  last_checked?: string
  last_error?: string
}

export interface AgentHistoryResponse {
  items?: AgentConversationItem[]
  messages: ChatMessage[]
  active_run?: {
    run_id: string
    session_id?: string
    started_at: string
    updated_at: string
  } | null
}

export interface AgentSessionSummary {
  session_id: string
  name: string
  created_at: string
  last_used_at: string
  message_count: number
  last_preview?: string
}

export interface AgentSessionsResponse {
  sessions: AgentSessionSummary[]
}

export interface AgentSessionsExportResponse {
  success: boolean
  message: string
  directory: string
  files: string[]
}
