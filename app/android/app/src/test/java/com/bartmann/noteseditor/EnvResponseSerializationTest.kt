package com.bartmann.noteseditor

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class EnvResponseSerializationTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun decode_withoutSuccessField_usesDefaultSuccessTrue() {
        val decoded = json.decodeFromString<EnvResponse>("""{"content":"KEY=value"}""")

        assertTrue(decoded.success)
        assertEquals("KEY=value", decoded.content)
    }
}
