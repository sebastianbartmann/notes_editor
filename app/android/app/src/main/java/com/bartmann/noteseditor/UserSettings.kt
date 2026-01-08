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
    private lateinit var prefs: SharedPreferences

    var person by mutableStateOf<String?>(null)
        private set
    var theme by mutableStateOf("dark")
        private set

    fun init(context: Context) {
        prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        person = prefs.getString(KEY_PERSON, null)
        theme = prefs.getString(KEY_THEME, "dark") ?: "dark"
    }

    fun updatePerson(value: String) {
        person = value
        prefs.edit().putString(KEY_PERSON, value).apply()
    }

    fun updateTheme(value: String) {
        theme = value
        prefs.edit().putString(KEY_THEME, value).apply()
    }
}
