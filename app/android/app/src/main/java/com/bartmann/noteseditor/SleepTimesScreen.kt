package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun SleepTimesScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var entries by remember { mutableStateOf(listOf<SleepEntry>()) }
    var child by remember { mutableStateOf("") }
    var entryText by remember { mutableStateOf("") }
    var asleep by remember { mutableStateOf(false) }
    var woke by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun refresh() {
        scope.launch {
            try {
                val response = ApiClient.fetchSleepTimes()
                entries = response.entries
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Load failed: ${exc.message}"
            }
        }
    }

    LaunchedEffect(Unit) {
        refresh()
    }

    ScreenLayout(
        modifier = modifier,
        padding = padding
    ) {
        ScreenTitle(text = "Sleep Times")
        Panel {
            Row(horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs)) {
                CompactButton(text = "Refresh") { refresh() }
            }
            CompactTextField(
                value = child,
                onValueChange = { child = it },
                placeholder = "Child",
                modifier = Modifier.fillMaxWidth()
            )
            CompactTextField(
                value = entryText,
                onValueChange = { entryText = it },
                placeholder = "Entry (19:30-06:10 | night)",
                modifier = Modifier.fillMaxWidth()
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Row {
                    AppCheckbox(
                        checked = asleep,
                        onCheckedChange = { asleep = it }
                    )
                    AppText(
                        text = "Eingeschlafen",
                        style = AppTheme.typography.label,
                        color = AppTheme.colors.text
                    )
                }
                Row {
                    AppCheckbox(
                        checked = woke,
                        onCheckedChange = { woke = it }
                    )
                    AppText(
                        text = "Aufgewacht",
                        style = AppTheme.typography.label,
                        color = AppTheme.colors.text
                    )
                }
            }
            CompactButton(text = "Append") {
                scope.launch {
                    try {
                        val response = ApiClient.appendSleepTimes(child, entryText, asleep, woke)
                        message = response.message
                        entryText = ""
                        asleep = false
                        woke = false
                        refresh()
                    } catch (exc: Exception) {
                        message = "Append failed: ${exc.message}"
                    }
                }
            }
            CompactDivider()
            SectionTitle(text = "Recent entries")
            if (entries.isEmpty()) {
                AppText(
                    text = "No entries found.",
                    style = AppTheme.typography.label,
                    color = AppTheme.colors.muted
                )
            } else {
                entries.forEach { entry ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        AppText(
                            text = entry.text,
                            style = AppTheme.typography.bodySmall,
                            color = AppTheme.colors.text,
                            modifier = Modifier.weight(1f)
                        )
                        CompactButton(text = "Delete") {
                            scope.launch {
                                try {
                                    val response = ApiClient.deleteSleepEntry(entry.lineNo)
                                    message = response.message
                                    refresh()
                                } catch (exc: Exception) {
                                    message = "Delete failed: ${exc.message}"
                                }
                            }
                        }
                    }
                }
            }
            StatusMessage(text = message)
        }
    }
}
