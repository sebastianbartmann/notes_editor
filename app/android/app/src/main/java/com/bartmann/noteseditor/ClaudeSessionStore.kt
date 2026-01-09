package com.bartmann.noteseditor

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object ClaudeSessionStore {
    var sessionId by mutableStateOf<String?>(null)
    val messages = mutableStateListOf<ChatMessage>()

    fun clear() {
        sessionId = null
        messages.clear()
    }
}
