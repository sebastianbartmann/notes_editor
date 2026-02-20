package com.bartmann.noteseditor

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object ClaudeSessionStore {
    var sessionId by mutableStateOf<String?>(null)
    val messages = mutableStateListOf<AgentConversationItem>()
    private val sessionCache = mutableMapOf<String, List<AgentConversationItem>>()
    private val draftInputsByPerson = mutableStateMapOf<String, String>()

    fun loadSession(newSessionId: String?, history: List<AgentConversationItem>) {
        sessionId = newSessionId
        messages.clear()
        messages.addAll(history)
    }

    fun clear() {
        sessionId?.let { sessionCache.remove(it) }
        sessionId = null
        messages.clear()
    }

    fun saveCurrentToCache() {
        val currentSessionId = sessionId ?: return
        sessionCache[currentSessionId] = messages.toList()
    }

    fun switchTo(targetId: String, fallbackHistory: List<AgentConversationItem>? = null) {
        saveCurrentToCache()
        sessionId = targetId
        messages.clear()
        val cached = sessionCache[targetId]
        when {
            cached != null -> messages.addAll(cached)
            fallbackHistory != null -> messages.addAll(fallbackHistory)
        }
    }

    fun startNew() {
        saveCurrentToCache()
        sessionId = null
        messages.clear()
    }

    fun clearCache() {
        sessionCache.clear()
    }

    fun isInCache(targetId: String): Boolean {
        return sessionCache.containsKey(targetId)
    }

    fun removeFromCache(id: String) {
        sessionCache.remove(id)
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
