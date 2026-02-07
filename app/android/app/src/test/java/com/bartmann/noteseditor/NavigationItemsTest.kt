package com.bartmann.noteseditor

import org.junit.Assert.assertEquals
import org.junit.Test

class NavigationItemsTest {
    @Test
    fun sanitizeBottomNavIds_filtersInvalidDeduplicatesAndCapsAtMax() {
        val ids = listOf("daily", "files", "daily", "invalid", "sleep", "claude", "noise")

        val sanitized = sanitizeBottomNavIds(ids)

        assertEquals(listOf("daily", "files", "sleep", "claude"), sanitized)
    }

    @Test
    fun sanitizeStoredBottomNavIds_fallsBackToDefaultWhenEmptyAfterSanitize() {
        val stored = listOf("invalid", "still-invalid")

        val sanitized = sanitizeStoredBottomNavIds(stored)

        assertEquals(defaultBottomNavIds, sanitized)
    }
}
