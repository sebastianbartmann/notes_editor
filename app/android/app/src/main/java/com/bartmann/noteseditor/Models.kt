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
data class EnvResponse(
    val success: Boolean,
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
data class ClaudeChatRequest(
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null
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
