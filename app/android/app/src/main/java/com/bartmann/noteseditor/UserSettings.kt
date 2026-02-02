package com.bartmann.noteseditor

import android.content.Context
import android.content.SharedPreferences
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object UserSettings {
    private const val PREFS_NAME = "notes_settings"
    private const val KEY_PERSON = "person_root"
    private const val KEY_THEME = "theme"
    private const val KEY_BOTTOM_NAV = "bottom_nav"
    private lateinit var prefs: SharedPreferences

    var person by mutableStateOf<String?>(null)
        private set
    var theme by mutableStateOf("dark")
        private set
    var bottomNavIds by mutableStateOf(defaultBottomNavIds)
        private set

    fun init(context: Context) {
        prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        person = prefs.getString(KEY_PERSON, null)
        theme = prefs.getString(KEY_THEME, "dark") ?: "dark"
        val storedBottomNav = prefs.getString(KEY_BOTTOM_NAV, "") ?: ""
        bottomNavIds = sanitizeStoredBottomNavIds(
            storedBottomNav.split(",").map { it.trim() }.filter { it.isNotBlank() }
        )
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
}
