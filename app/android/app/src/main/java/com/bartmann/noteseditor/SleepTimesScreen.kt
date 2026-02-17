package com.bartmann.noteseditor

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

private enum class SleepTab { Log, History, Summary }

private fun localNowInputValue(): String =
    LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm"))

private fun localInputToIso(value: String): String? =
    runCatching {
        val dt = LocalDateTime.parse(value, DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm"))
        dt.atZone(ZoneId.systemDefault()).toInstant().toString()
    }.getOrNull()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SleepTimesScreen(modifier: Modifier) {
    var tab by remember { mutableStateOf(SleepTab.Log) }
    var entries by remember { mutableStateOf(listOf<SleepEntry>()) }
    var summary by remember { mutableStateOf(SleepSummaryResponse()) }

    var child by remember { mutableStateOf("Fabian") }
    var status by remember { mutableStateOf("eingeschlafen") }
    var entryText by remember { mutableStateOf("") }
    var occurredAtInput by remember { mutableStateOf(localNowInputValue()) }

    var editingId by remember { mutableStateOf<String?>(null) }
    var editingChild by remember { mutableStateOf("Fabian") }
    var editingStatus by remember { mutableStateOf("eingeschlafen") }
    var editingTime by remember { mutableStateOf("") }
    var editingOccurredAt by remember { mutableStateOf("") }

    var message by remember { mutableStateOf("") }
    var isRefreshing by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun refresh() {
        scope.launch {
            try {
                AppSync.syncIfStale(timeoutMs = 2_000, maxAgeMs = 30_000)
                val times = ApiClient.fetchSleepTimes()
                val summaryResp = ApiClient.fetchSleepSummary()
                entries = times.entries
                summary = summaryResp
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Load failed: ${exc.message}"
            }
            isRefreshing = false
        }
    }

    LaunchedEffect(Unit) {
        refresh()
    }

    PullToRefreshBox(
        isRefreshing = isRefreshing,
        onRefresh = {
            isRefreshing = true
            refresh()
        },
        modifier = modifier.fillMaxSize()
    ) {
        ScreenLayout(modifier = Modifier) {
            ScreenHeader(
                title = "Sleep",
                actionButton = {
                    IconButton(onClick = {
                        isRefreshing = true
                        refresh()
                    }) {
                        Icon(
                            Icons.Default.Refresh,
                            contentDescription = "Reload",
                            tint = AppTheme.colors.accent
                        )
                    }
                }
            )

            Panel {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
                ) {
                    CompactButton(text = "Log", modifier = Modifier.weight(1f), onClick = { tab = SleepTab.Log })
                    CompactButton(text = "History", modifier = Modifier.weight(1f), onClick = { tab = SleepTab.History })
                    CompactButton(text = "Summary", modifier = Modifier.weight(1f), onClick = { tab = SleepTab.Summary })
                }

                CompactButton(
                    text = "Export sleep data to markdown",
                    modifier = Modifier.fillMaxWidth(),
                    onClick = {
                        scope.launch {
                            try {
                                val resp = ApiClient.exportSleepMarkdown()
                                message = resp.message
                            } catch (exc: Exception) {
                                message = "Export failed: ${exc.message}"
                            }
                        }
                    }
                )

                when (tab) {
                    SleepTab.Log -> {
                        SectionTitle(text = "Log")
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.lg)
                        ) {
                            Column(
                                modifier = Modifier.weight(1f),
                                verticalArrangement = Arrangement.spacedBy(6.dp)
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    modifier = Modifier.clickable { child = "Thomas" }
                                ) {
                                    AppCheckbox(checked = child == "Thomas", size = 18)
                                    AppText(text = "Thomas", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                }
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    modifier = Modifier.clickable { status = "eingeschlafen" }
                                ) {
                                    AppCheckbox(checked = status == "eingeschlafen", size = 18)
                                    AppText(text = "Eingeschlafen", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                }
                            }
                            Column(
                                modifier = Modifier.weight(1f),
                                verticalArrangement = Arrangement.spacedBy(6.dp)
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    modifier = Modifier.clickable { child = "Fabian" }
                                ) {
                                    AppCheckbox(checked = child == "Fabian", size = 18)
                                    AppText(text = "Fabian", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                }
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    modifier = Modifier.clickable { status = "aufgewacht" }
                                ) {
                                    AppCheckbox(checked = status == "aufgewacht", size = 18)
                                    AppText(text = "Aufgewacht", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                }
                            }
                        }

                        CompactTextField(
                            value = occurredAtInput,
                            onValueChange = { occurredAtInput = it },
                            placeholder = "When (yyyy-MM-ddTHH:mm)",
                            modifier = Modifier.fillMaxWidth()
                        )
                        CompactTextField(
                            value = entryText,
                            onValueChange = { entryText = it },
                            placeholder = "Legacy time/raw note",
                            modifier = Modifier.fillMaxWidth()
                        )

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.End
                        ) {
                            CompactButton(
                                text = "Add",
                                onClick = {
                                    if (entryText.isBlank() && occurredAtInput.isBlank()) {
                                        message = "Time is required"
                                        return@CompactButton
                                    }
                                    scope.launch {
                                        try {
                                            val response = ApiClient.appendSleepTimes(
                                                child = child,
                                                time = entryText,
                                                status = status,
                                                occurredAt = localInputToIso(occurredAtInput)
                                            )
                                            message = response.message
                                            entryText = ""
                                            occurredAtInput = localNowInputValue()
                                            refresh()
                                            tab = SleepTab.History
                                        } catch (exc: Exception) {
                                            message = "Append failed: ${exc.message}"
                                        }
                                    }
                                }
                            )
                        }
                    }

                    SleepTab.History -> {
                        SectionTitle(text = "Recent entries")
                        if (entries.isEmpty()) {
                            AppText(
                                text = "No entries found.",
                                style = AppTheme.typography.label,
                                color = AppTheme.colors.muted
                            )
                        } else {
                            entries.forEach { entry ->
                                if (editingId == entry.id) {
                                    Column(modifier = Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                        CompactTextField(
                                            value = editingChild,
                                            onValueChange = { editingChild = it },
                                            placeholder = "Child",
                                            modifier = Modifier.fillMaxWidth()
                                        )
                                        CompactTextField(
                                            value = editingStatus,
                                            onValueChange = { editingStatus = it },
                                            placeholder = "Status",
                                            modifier = Modifier.fillMaxWidth()
                                        )
                                        CompactTextField(
                                            value = editingOccurredAt,
                                            onValueChange = { editingOccurredAt = it },
                                            placeholder = "When (yyyy-MM-ddTHH:mm)",
                                            modifier = Modifier.fillMaxWidth()
                                        )
                                        CompactTextField(
                                            value = editingTime,
                                            onValueChange = { editingTime = it },
                                            placeholder = "Time",
                                            modifier = Modifier.fillMaxWidth()
                                        )
                                        Row(
                                            modifier = Modifier.fillMaxWidth(),
                                            horizontalArrangement = Arrangement.spacedBy(6.dp, Alignment.End)
                                        ) {
                                            CompactButton(
                                                text = "Cancel",
                                                onClick = {
                                                    editingId = null
                                                }
                                            )
                                            CompactButton(
                                                text = "Save",
                                                onClick = {
                                                    val id = editingId ?: return@CompactButton
                                                    scope.launch {
                                                        try {
                                                            ApiClient.updateSleepEntry(
                                                                id = id,
                                                                child = editingChild,
                                                                time = editingTime,
                                                                status = editingStatus,
                                                                occurredAt = localInputToIso(editingOccurredAt)
                                                            )
                                                            editingId = null
                                                            refresh()
                                                        } catch (exc: Exception) {
                                                            message = "Update failed: ${exc.message}"
                                                        }
                                                    }
                                                }
                                            )
                                        }
                                        CompactDivider()
                                    }
                                } else {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                                    ) {
                                        AppText(
                                            text = "${entry.date} | ${entry.child} | ${entry.time} | ${entry.status}${if (!entry.notes.isNullOrBlank()) " | ${entry.notes}" else ""}",
                                            style = AppTheme.typography.bodySmall,
                                            color = AppTheme.colors.text,
                                            modifier = Modifier.weight(1f)
                                        )
                                        CompactButton(
                                            text = "Edit",
                                            onClick = {
                                                editingId = entry.id
                                                editingChild = entry.child
                                                editingStatus = entry.status
                                                editingTime = entry.time
                                                editingOccurredAt = ""
                                            }
                                        )
                                        CompactButton(
                                            text = "Delete",
                                            background = AppTheme.colors.danger,
                                            border = AppTheme.colors.danger,
                                            textColor = AppTheme.colors.text,
                                            onClick = {
                                                scope.launch {
                                                    try {
                                                        val response = ApiClient.deleteSleepEntry(entry.id)
                                                        message = response.message
                                                        refresh()
                                                    } catch (exc: Exception) {
                                                        message = "Delete failed: ${exc.message}"
                                                    }
                                                }
                                            }
                                        )
                                    }
                                }
                            }
                        }
                    }

                    SleepTab.Summary -> {
                        SectionTitle(text = "Average bed/wake")
                        if (summary.averages.isEmpty()) {
                            AppText(
                                text = "Not enough paired data yet.",
                                style = AppTheme.typography.label,
                                color = AppTheme.colors.muted
                            )
                        } else {
                            summary.averages.forEach { avg ->
                                AppText(
                                    text = "${avg.child} (${avg.days}d): bed ${avg.averageBedtime}, wake ${avg.averageWakeTime}",
                                    style = AppTheme.typography.bodySmall,
                                    color = AppTheme.colors.text
                                )
                            }
                        }

                        CompactDivider()
                        SectionTitle(text = "Night durations")
                        if (summary.nights.isEmpty()) {
                            AppText(
                                text = "No completed nights.",
                                style = AppTheme.typography.label,
                                color = AppTheme.colors.muted
                            )
                        } else {
                            summary.nights.forEach { night ->
                                AppText(
                                    text = "${night.nightDate} | ${night.child} | ${night.durationMinutes} min | ${night.bedtime} - ${night.wakeTime}",
                                    style = AppTheme.typography.bodySmall,
                                    color = AppTheme.colors.text
                                )
                            }
                        }
                    }
                }

                StatusMessage(text = message)
            }
        }
    }
}
