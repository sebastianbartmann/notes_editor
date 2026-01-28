package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.ui.Alignment
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.clickable
import androidx.compose.foundation.text.selection.SelectionContainer

private data class NoteLine(
    val lineNo: Int,
    val text: String,
    val type: LineType,
    val done: Boolean = false,
    val isPinned: Boolean = false
)

private enum class LineType {
    H1, H2, H3, H4, TASK, TEXT, EMPTY
}

@Composable
fun NoteView(
    content: String,
    onToggleTask: (Int) -> Unit,
    onUnpin: ((Int) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    val lines = parseNoteLines(content)
    val scrollState = rememberScrollState()
    NoteSurface(
        modifier = modifier
            .verticalScroll(scrollState)
    ) {
        SelectionContainer {
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                lines.forEach { line ->
                when (line.type) {
                    LineType.EMPTY -> AppText(
                        text = " ",
                        style = AppTheme.typography.bodySmall,
                        color = AppTheme.colors.text
                    )
                    LineType.H1 -> AppText(
                        text = line.text,
                        style = AppTheme.typography.title.copy(fontWeight = FontWeight.SemiBold),
                        color = AppTheme.colors.accent
                    )
                    LineType.H2 -> AppText(
                        text = line.text.uppercase(),
                        style = AppTheme.typography.section.copy(fontWeight = FontWeight.SemiBold),
                        color = AppTheme.colors.muted
                    )
                    LineType.H3 -> {
                        if (line.isPinned && onUnpin != null) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                AppText(
                                    text = line.text,
                                    style = AppTheme.typography.section.copy(fontWeight = FontWeight.SemiBold),
                                    color = AppTheme.colors.accent,
                                    modifier = Modifier.weight(1f)
                                )
                                CompactTextButton(
                                    text = "Unpin",
                                    onClick = { onUnpin(line.lineNo) }
                                )
                            }
                        } else {
                            AppText(
                                text = line.text,
                                style = AppTheme.typography.section.copy(fontWeight = FontWeight.SemiBold),
                                color = AppTheme.colors.accent
                            )
                        }
                    }
                    LineType.H4 -> AppText(
                        text = line.text,
                        style = AppTheme.typography.bodySmall.copy(fontWeight = FontWeight.SemiBold),
                        color = AppTheme.colors.text
                    )
                    LineType.TASK -> {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { onToggleTask(line.lineNo) },
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            AppCheckbox(
                                checked = line.done
                            )
                            AppText(
                                text = line.text,
                                style = AppTheme.typography.body,
                                color = if (line.done) AppTheme.colors.muted else AppTheme.colors.text
                            )
                        }
                    }
                    LineType.TEXT -> AppText(
                        text = line.text,
                        style = AppTheme.typography.body,
                        color = AppTheme.colors.text
                    )
                }
            }
            }
        }
    }
}

private val PINNED_REGEX = Regex("<pinned>", RegexOption.IGNORE_CASE)

private fun parseNoteLines(content: String): List<NoteLine> {
    val lines = content.lines()
    val items = mutableListOf<NoteLine>()
    val taskRegex = Regex("^\\s*-\\s*\\[( |x|X)\\]\\s*(.*)$")
    for ((idx, raw) in lines.withIndex()) {
        val lineNo = idx + 1
        val trimmed = raw.trimEnd()
        if (trimmed.isBlank()) {
            items.add(NoteLine(lineNo, "", LineType.EMPTY))
            continue
        }
        val taskMatch = taskRegex.find(trimmed)
        if (taskMatch != null) {
            val marker = taskMatch.groupValues[1]
            val text = taskMatch.groupValues[2].ifBlank { trimmed }
            items.add(
                NoteLine(
                    lineNo = lineNo,
                    text = text,
                    type = LineType.TASK,
                    done = marker.equals("x", ignoreCase = true)
                )
            )
            continue
        }
        when {
            trimmed.startsWith("#### ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("#### ").trim(), LineType.H4))
            trimmed.startsWith("### ") -> {
                val headingText = trimmed.removePrefix("### ").trim()
                val isPinned = PINNED_REGEX.containsMatchIn(headingText)
                val displayText = if (isPinned) headingText.replace(PINNED_REGEX, "").trim() else headingText
                items.add(NoteLine(lineNo, displayText, LineType.H3, isPinned = isPinned))
            }
            trimmed.startsWith("## ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("## ").trim(), LineType.H2))
            trimmed.startsWith("# ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("# ").trim(), LineType.H1))
            else -> items.add(NoteLine(lineNo, trimmed, LineType.TEXT))
        }
    }
    return items
}
