package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
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

    Column(
        modifier = modifier
            .padding(padding)
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        ScreenTitle(text = "Sleep Times")
        Panel {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Refresh") { refresh() }
            }
            CompactOutlinedTextField(
                value = child,
                onValueChange = { child = it },
                label = "Child",
                modifier = Modifier.fillMaxWidth()
            )
            CompactOutlinedTextField(
                value = entryText,
                onValueChange = { entryText = it },
                label = "Entry (19:30-06:10 | night)",
                modifier = Modifier.fillMaxWidth()
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Row {
                    Checkbox(
                        checked = asleep,
                        onCheckedChange = { asleep = it },
                        colors = CheckboxDefaults.colors(
                            checkedColor = MaterialTheme.colorScheme.primary,
                            uncheckedColor = MaterialTheme.colorScheme.secondary
                        )
                    )
                    Text(text = "Eingeschlafen")
                }
                Row {
                    Checkbox(
                        checked = woke,
                        onCheckedChange = { woke = it },
                        colors = CheckboxDefaults.colors(
                            checkedColor = MaterialTheme.colorScheme.primary,
                            uncheckedColor = MaterialTheme.colorScheme.secondary
                        )
                    )
                    Text(text = "Aufgewacht")
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
                Text(text = "No entries found.", color = MaterialTheme.colorScheme.secondary)
            } else {
                entries.forEach { entry ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Text(text = entry.text, modifier = Modifier.weight(1f))
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
            if (message.isNotBlank()) {
                CompactDivider()
                Text(text = message, color = MaterialTheme.colorScheme.secondary)
            }
        }
    }
}
