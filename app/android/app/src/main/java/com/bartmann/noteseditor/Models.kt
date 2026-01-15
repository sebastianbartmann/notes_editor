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
    @SerialName("line_no")
    val lineNo: Int,
    val text: String
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
