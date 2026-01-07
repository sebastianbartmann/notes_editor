package com.bartmann.noteseditor

fun parseTodos(content: String): List<TodoItem> {
    val todos = mutableListOf<TodoItem>()
    val lines = content.lines()
    val regex = Regex("^\\s*-\\s*\\[(.| )\\]\\s*(.*)$")
    lines.forEachIndexed { index, line ->
        val match = regex.find(line) ?: return@forEachIndexed
        val marker = match.groupValues[1]
        val text = match.groupValues[2]
        val done = marker.equals("x", ignoreCase = true)
        todos.add(TodoItem(index + 1, text.ifBlank { line.trim() }, done))
    }
    return todos
}

fun parsePinned(content: String): List<PinnedItem> {
    val pinned = mutableListOf<PinnedItem>()
    val regex = Regex("^###\\s+.*<pinned>.*$", RegexOption.IGNORE_CASE)
    content.lines().forEachIndexed { index, line ->
        if (regex.containsMatchIn(line)) {
            pinned.add(PinnedItem(index + 1, line.trim()))
        }
    }
    return pinned
}
