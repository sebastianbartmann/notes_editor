package com.bartmann.noteseditor

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

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
data class ClaudeResponse(
    val success: Boolean,
    val message: String,
    val response: String
)
