package com.bartmann.noteseditor

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object ClaudeSessionStore {
    var sessionId by mutableStateOf<String?>(null)
    val messages = mutableStateListOf<ChatMessage>()
    private val draftInputsByPerson = mutableStateMapOf<String, String>()

    fun loadSession(newSessionId: String?, history: List<ChatMessage>) {
        sessionId = newSessionId
        messages.clear()
        messages.addAll(history)
    }

    fun clear() {
        sessionId = null
        messages.clear()
    }

    fun draftInput(person: String?): String {
        if (person == null) return ""
        return draftInputsByPerson[person].orEmpty()
    }

    fun updateDraftInput(person: String?, value: String) {
        if (person == null) return
        if (value.isEmpty()) {
            draftInputsByPerson.remove(person)
        } else {
            draftInputsByPerson[person] = value
        }
    }
}
