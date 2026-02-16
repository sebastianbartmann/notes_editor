package com.bartmann.noteseditor

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ClaudeSessionStoreTest {
    @Test
    fun draftInput_isScopedByPersonAndClearedWhenEmpty() {
        ClaudeSessionStore.updateDraftInput("sebastian", "hello")
        ClaudeSessionStore.updateDraftInput("petra", "world")

        assertEquals("hello", ClaudeSessionStore.draftInput("sebastian"))
        assertEquals("world", ClaudeSessionStore.draftInput("petra"))

        ClaudeSessionStore.updateDraftInput("sebastian", "")

        assertEquals("", ClaudeSessionStore.draftInput("sebastian"))
        assertEquals("world", ClaudeSessionStore.draftInput("petra"))
    }

    @Test
    fun clear_resetsSessionAndMessagesButKeepsDraftInput() {
        ClaudeSessionStore.sessionId = "session-1"
        ClaudeSessionStore.messages.add(ChatMessage(role = "user", content = "hi"))
        ClaudeSessionStore.updateDraftInput("sebastian", "draft")

        ClaudeSessionStore.clear()

        assertNull(ClaudeSessionStore.sessionId)
        assertEquals(0, ClaudeSessionStore.messages.size)
        assertEquals("draft", ClaudeSessionStore.draftInput("sebastian"))
    }
}
