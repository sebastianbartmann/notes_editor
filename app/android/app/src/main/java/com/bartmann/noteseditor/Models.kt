package com.bartmann.noteseditor

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class DailyNote(
    val date: String,
    val path: String,
    val content: String
)


@Serializable
data class SleepTimesResponse(
    val entries: List<SleepEntry>
)

@Serializable
data class SleepEntry(
    val line: Int,
    val date: String,
    val child: String,
    val time: String,
    val status: String
)

@Serializable
data class FilesResponse(
    val entries: List<FileEntry>
)

@Serializable
data class FilesResponsePayload(
    val entries: List<FileEntry>? = emptyList()
)

@Serializable
data class FileEntry(
    val name: String,
    val path: String,
    @SerialName("is_dir")
    val isDir: Boolean
)

@Serializable
data class FileReadResponse(
    val path: String,
    val content: String
)

@Serializable
data class ApiMessage(
    val success: Boolean = true,
    val message: String = ""
)

@Serializable
data class SyncStatus(
    @SerialName("in_progress")
    val inProgress: Boolean = false,
    @SerialName("pending_pull")
    val pendingPull: Boolean = false,
    @SerialName("pending_push")
    val pendingPush: Boolean = false,
    @SerialName("last_pull_at")
    val lastPullAt: String? = null,
    @SerialName("last_push_at")
    val lastPushAt: String? = null,
    @SerialName("last_error")
    val lastError: String? = null,
    @SerialName("last_error_at")
    val lastErrorAt: String? = null
)

@Serializable
data class IndexStatus(
    @SerialName("in_progress")
    val inProgress: Boolean = false,
    val pending: Boolean = false,
    @SerialName("last_reason")
    val lastReason: String? = null,
    @SerialName("last_started_at")
    val lastStartedAt: String? = null,
    @SerialName("last_success_at")
    val lastSuccessAt: String? = null,
    @SerialName("last_error")
    val lastError: String? = null,
    @SerialName("last_error_at")
    val lastErrorAt: String? = null
)

@Serializable
data class GitStatusResponse(
    val output: String = ""
)

@Serializable
data class GitActionResponse(
    val success: Boolean = false,
    val message: String = "",
    val output: String? = null
)

@Serializable
data class EnvResponse(
    val success: Boolean = true,
    val content: String = "",
    val message: String = ""
)

@Serializable
data class ChatMessage(
    val role: String,
    val content: String
)

@Serializable
data class ClaudeChatResponse(
    val success: Boolean,
    val message: String = "",
    val response: String = "",
    @SerialName("session_id")
    val sessionId: String = "",
    val history: List<ChatMessage> = emptyList()
)

@Serializable
data class ClaudeStreamEvent(
    val type: String,
    val delta: String? = null,
    val name: String? = null,
    val input: JsonElement? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    val message: String? = null
)

@Serializable
data class AgentStreamEvent(
    val type: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("run_id")
    val runId: String? = null,
    val seq: Int? = null,
    val ts: String? = null,
    val delta: String? = null,
    val tool: String? = null,
    val args: JsonElement? = null,
    val ok: Boolean? = null,
    val summary: String? = null,
    val message: String? = null,
    val usage: AgentUsage? = null
)

@Serializable
data class AgentUsage(
    @SerialName("input_tokens")
    val inputTokens: Int? = null,
    @SerialName("output_tokens")
    val outputTokens: Int? = null,
    @SerialName("cache_read_tokens")
    val cacheReadTokens: Int? = null,
    @SerialName("cache_write_tokens")
    val cacheWriteTokens: Int? = null,
    @SerialName("total_tokens")
    val totalTokens: Int? = null,
    @SerialName("context_window")
    val contextWindow: Int? = null,
    @SerialName("remaining_tokens")
    val remainingTokens: Int? = null
)

@Serializable
data class AgentConversationItem(
    val type: String,
    val role: String? = null,
    val content: String? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("run_id")
    val runId: String? = null,
    val seq: Int? = null,
    val ts: String? = null,
    val tool: String? = null,
    val args: JsonElement? = null,
    val ok: Boolean? = null,
    val summary: String? = null,
    val message: String? = null,
    val usage: AgentUsage? = null
)

@Serializable
data class AgentConfig(
    @SerialName("runtime_mode")
    val runtimeMode: String = "gateway_subscription",
    @SerialName("prompt_path")
    val promptPath: String = "agents.md",
    @SerialName("actions_path")
    val actionsPath: String = "agent/actions",
    val prompt: String = ""
)

@Serializable
data class AgentActionMetadata(
    @SerialName("requires_confirmation")
    val requiresConfirmation: Boolean = false,
    @SerialName("max_steps")
    val maxSteps: Int? = null
)

@Serializable
data class AgentAction(
    val id: String,
    val label: String,
    val path: String,
    val metadata: AgentActionMetadata
)

@Serializable
data class AgentActionsResponse(
    val actions: List<AgentAction> = emptyList()
)

@Serializable
data class AgentSessionSummary(
    @SerialName("session_id")
    val sessionId: String,
    val name: String,
    @SerialName("created_at")
    val createdAt: String,
    @SerialName("last_used_at")
    val lastUsedAt: String,
    @SerialName("message_count")
    val messageCount: Int = 0,
    @SerialName("last_preview")
    val lastPreview: String? = null
)

@Serializable
data class AgentSessionsResponse(
    val sessions: List<AgentSessionSummary> = emptyList()
)

@Serializable
data class AgentSessionHistoryResponse(
    val items: List<AgentConversationItem> = emptyList(),
    val messages: List<ChatMessage> = emptyList()
)

// Request models for JSON serialization

@Serializable
data class SaveDailyRequest(
    val path: String,
    val content: String
)

@Serializable
data class AppendDailyRequest(
    val path: String,
    val text: String,
    val pinned: Boolean = false
)

@Serializable
data class ClearPinnedRequest(
    val path: String
)

@Serializable
data class AddTodoRequest(
    val category: String,
    val text: String = ""
)

@Serializable
data class ToggleTodoRequest(
    val path: String,
    val line: Int
)

@Serializable
data class AppendSleepRequest(
    val child: String,
    val time: String,
    val status: String
)

@Serializable
data class DeleteSleepRequest(
    val line: Int
)

@Serializable
data class CreateFileRequest(
    val path: String
)

@Serializable
data class SaveFileRequest(
    val path: String,
    val content: String
)

@Serializable
data class DeleteFileRequest(
    val path: String
)

@Serializable
data class UnpinEntryRequest(
    val path: String,
    val line: Int
)

@Serializable
data class SyncRequest(
    val wait: Boolean = false,
    @SerialName("timeout_ms")
    val timeoutMs: Int = 0
)

@Serializable
data class ClaudeChatRequest(
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null
)

@Serializable
data class AgentChatRequest(
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("action_id")
    val actionId: String? = null,
    val confirm: Boolean? = null
)

@Serializable
data class AgentConfigUpdateRequest(
    @SerialName("runtime_mode")
    val runtimeMode: String? = null,
    val prompt: String? = null
)

@Serializable
data class ClaudeClearRequest(
    @SerialName("session_id")
    val sessionId: String
)

@Serializable
data class SaveEnvRequest(
    val content: String
)
