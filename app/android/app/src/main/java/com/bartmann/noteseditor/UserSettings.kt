package com.bartmann.noteseditor

import android.content.Context
import android.content.SharedPreferences
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlin.math.round

object UserSettings {
    private const val PREFS_NAME = "notes_settings"
    private const val KEY_PERSON = "person_root"
    private const val KEY_THEME = "theme"
    private const val KEY_BOTTOM_NAV = "bottom_nav"
    private const val KEY_TEXT_SCALE = "text_scale"
    private const val KEY_SHOW_AGENT_TOOL_CALLS = "show_agent_tool_calls"
    private lateinit var prefs: SharedPreferences

    var person by mutableStateOf<String?>(null)
        private set
    var theme by mutableStateOf("dark")
        private set
    var bottomNavIds by mutableStateOf(defaultBottomNavIds)
        private set
    var textScale by mutableStateOf(DEFAULT_TEXT_SCALE)
        private set
    var showAgentToolCalls by mutableStateOf(true)
        private set

    fun init(context: Context) {
        prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        person = prefs.getString(KEY_PERSON, null)
        theme = prefs.getString(KEY_THEME, "dark") ?: "dark"
        val storedBottomNav = prefs.getString(KEY_BOTTOM_NAV, "") ?: ""
        bottomNavIds = sanitizeStoredBottomNavIds(
            storedBottomNav.split(",").map { it.trim() }.filter { it.isNotBlank() }
        )
        textScale = sanitizeTextScale(prefs.getFloat(KEY_TEXT_SCALE, DEFAULT_TEXT_SCALE))
        showAgentToolCalls = prefs.getBoolean(KEY_SHOW_AGENT_TOOL_CALLS, true)
    }

    fun updatePerson(value: String) {
        person = value
        prefs.edit().putString(KEY_PERSON, value).apply()
    }

    fun updateTheme(value: String) {
        theme = value
        prefs.edit().putString(KEY_THEME, value).apply()
    }

    fun updateBottomNav(ids: List<String>) {
        val normalized = sanitizeBottomNavIds(ids)
        bottomNavIds = normalized
        prefs.edit().putString(KEY_BOTTOM_NAV, normalized.joinToString(",")).apply()
    }

    fun updateTextScale(value: Float) {
        val normalized = sanitizeTextScale(value)
        textScale = normalized
        prefs.edit().putFloat(KEY_TEXT_SCALE, normalized).apply()
    }

    fun stepTextScale(stepDelta: Int) {
        if (stepDelta == 0) return
        updateTextScale(nextTextScale(textScale, stepDelta))
    }

    fun resetTextScale() {
        updateTextScale(DEFAULT_TEXT_SCALE)
    }

    fun updateShowAgentToolCalls(value: Boolean) {
        showAgentToolCalls = value
        prefs.edit().putBoolean(KEY_SHOW_AGENT_TOOL_CALLS, value).apply()
    }
}

const val DEFAULT_TEXT_SCALE = 1.0f
const val MIN_TEXT_SCALE = 0.85f
const val MAX_TEXT_SCALE = 1.4f
const val TEXT_SCALE_STEP = 0.05f

fun sanitizeTextScale(value: Float): Float {
    if (!value.isFinite()) {
        return DEFAULT_TEXT_SCALE
    }
    return value.coerceIn(MIN_TEXT_SCALE, MAX_TEXT_SCALE)
}

fun nextTextScale(current: Float, stepDelta: Int): Float {
    val base = sanitizeTextScale(current)
    val stepped = base + (stepDelta * TEXT_SCALE_STEP)
    val snapped = round(stepped / TEXT_SCALE_STEP) * TEXT_SCALE_STEP
    return sanitizeTextScale(snapped)
}
