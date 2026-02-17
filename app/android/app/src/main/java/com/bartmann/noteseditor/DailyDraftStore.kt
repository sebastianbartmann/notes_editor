package com.bartmann.noteseditor

import androidx.compose.runtime.mutableStateMapOf

private data class DailyDraftState(
    val appendText: String = "",
    val pinned: Boolean = false,
    val taskInputMode: String? = null,
    val taskInputText: String = ""
)

object DailyDraftStore {
    private val draftsByPerson = mutableStateMapOf<String, DailyDraftState>()

    fun appendText(person: String?): String {
        if (person == null) return ""
        return draftsByPerson[person]?.appendText.orEmpty()
    }

    fun pinned(person: String?): Boolean {
        if (person == null) return false
        return draftsByPerson[person]?.pinned ?: false
    }

    fun taskInputMode(person: String?): String? {
        if (person == null) return null
        return draftsByPerson[person]?.taskInputMode
    }

    fun taskInputText(person: String?): String {
        if (person == null) return ""
        return draftsByPerson[person]?.taskInputText.orEmpty()
    }

    fun updateAppendText(person: String?, value: String) {
        update(person) { it.copy(appendText = value) }
    }

    fun updatePinned(person: String?, value: Boolean) {
        update(person) { it.copy(pinned = value) }
    }

    fun updateTaskInputMode(person: String?, value: String?) {
        update(person) { it.copy(taskInputMode = value) }
    }

    fun updateTaskInputText(person: String?, value: String) {
        update(person) { it.copy(taskInputText = value) }
    }

    private fun update(person: String?, transform: (DailyDraftState) -> DailyDraftState) {
        if (person == null) return
        val current = draftsByPerson[person] ?: DailyDraftState()
        val next = transform(current)
        if (next.appendText.isBlank() && !next.pinned && next.taskInputMode == null && next.taskInputText.isBlank()) {
            draftsByPerson.remove(person)
            return
        }
        draftsByPerson[person] = next
    }
}
