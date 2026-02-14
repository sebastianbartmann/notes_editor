package com.bartmann.noteseditor

import org.junit.Assert.assertEquals
import org.junit.Test

class UserSettingsTextScaleTest {
    @Test
    fun sanitizeTextScale_clampsAndFallsBackForInvalid() {
        assertEquals(DEFAULT_TEXT_SCALE, sanitizeTextScale(Float.NaN), 0.0001f)
        assertEquals(DEFAULT_TEXT_SCALE, sanitizeTextScale(Float.POSITIVE_INFINITY), 0.0001f)
        assertEquals(MIN_TEXT_SCALE, sanitizeTextScale(0.1f), 0.0001f)
        assertEquals(MAX_TEXT_SCALE, sanitizeTextScale(4.0f), 0.0001f)
        assertEquals(1.15f, sanitizeTextScale(1.15f), 0.0001f)
    }

    @Test
    fun nextTextScale_stepsByConfiguredIncrementAndClampsBounds() {
        assertEquals(1.05f, nextTextScale(1.0f, 1), 0.0001f)
        assertEquals(0.95f, nextTextScale(1.0f, -1), 0.0001f)
        assertEquals(1.1f, nextTextScale(1.03f, 1), 0.0001f)
        assertEquals(MAX_TEXT_SCALE, nextTextScale(MAX_TEXT_SCALE, 1), 0.0001f)
        assertEquals(MIN_TEXT_SCALE, nextTextScale(MIN_TEXT_SCALE, -1), 0.0001f)
    }
}
