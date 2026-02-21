package com.bartmann.noteseditor

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

private enum class SleepTab { Log, History, Summary }

private val dateFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd")
private val timeFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("HH:mm")

private fun localDateTimeToIso(value: LocalDateTime): String =
    value.atZone(ZoneId.systemDefault()).toInstant().toString()

private fun entryToLocalDateTime(entry: SleepEntry): LocalDateTime? {
    val zone = ZoneId.systemDefault()
    val occurredAt = entry.occurredAt
    if (!occurredAt.isNullOrBlank()) {
        return runCatching {
            Instant.parse(occurredAt).atZone(zone).toLocalDateTime()
        }.getOrNull()
    }

    if (entry.time.isBlank() || entry.time == "-") {
        return null
    }

    return runCatching {
        LocalDateTime.parse("${entry.date}T${entry.time}", DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm"))
    }.getOrNull()
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SleepTimesScreen(modifier: Modifier) {
    var tab by remember { mutableStateOf(SleepTab.Log) }
    var entries by remember { mutableStateOf(listOf<SleepEntry>()) }
    var summary by remember { mutableStateOf(SleepSummaryResponse()) }

    var child by remember { mutableStateOf("Fabian") }
    var status by remember { mutableStateOf("eingeschlafen") }
    var occurredAt by remember { mutableStateOf(LocalDateTime.now().withSecond(0).withNano(0)) }
    var notes by remember { mutableStateOf("") }

    var editingId by remember { mutableStateOf<String?>(null) }
    var editingChild by remember { mutableStateOf("Fabian") }
    var editingStatus by remember { mutableStateOf("eingeschlafen") }
    var editingOccurredAt by remember { mutableStateOf(LocalDateTime.now().withSecond(0).withNano(0)) }
    var editingNotes by remember { mutableStateOf("") }

    var showDatePicker by remember { mutableStateOf(false) }
    var showTimePicker by remember { mutableStateOf(false) }
    var showEditingDatePicker by remember { mutableStateOf(false) }
    var showEditingTimePicker by remember { mutableStateOf(false) }

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

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
                        ) {
                            CompactButton(
                                text = occurredAt.format(dateFormatter),
                                modifier = Modifier.weight(1f),
                                onClick = { showDatePicker = true }
                            )
                            CompactButton(
                                text = occurredAt.format(timeFormatter),
                                modifier = Modifier.weight(1f),
                                onClick = { showTimePicker = true }
                            )
                        }

                        CompactTextField(
                            value = notes,
                            onValueChange = { notes = it },
                            placeholder = "Optional notes",
                            modifier = Modifier.fillMaxWidth()
                        )

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.End
                        ) {
                            CompactButton(
                                text = "Add",
                                onClick = {
                                    scope.launch {
                                        try {
                                            val response = ApiClient.appendSleepTimes(
                                                child = child,
                                                status = status,
                                                occurredAt = localDateTimeToIso(occurredAt),
                                                notes = notes.trim()
                                            )
                                            message = response.message
                                            notes = ""
                                            occurredAt = LocalDateTime.now().withSecond(0).withNano(0)
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
                                                    modifier = Modifier.clickable { editingChild = "Thomas" }
                                                ) {
                                                    AppCheckbox(checked = editingChild == "Thomas", size = 18)
                                                    AppText(text = "Thomas", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                                }
                                                Row(
                                                    verticalAlignment = Alignment.CenterVertically,
                                                    modifier = Modifier.clickable { editingStatus = "eingeschlafen" }
                                                ) {
                                                    AppCheckbox(checked = editingStatus == "eingeschlafen", size = 18)
                                                    AppText(text = "Eingeschlafen", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                                }
                                            }
                                            Column(
                                                modifier = Modifier.weight(1f),
                                                verticalArrangement = Arrangement.spacedBy(6.dp)
                                            ) {
                                                Row(
                                                    verticalAlignment = Alignment.CenterVertically,
                                                    modifier = Modifier.clickable { editingChild = "Fabian" }
                                                ) {
                                                    AppCheckbox(checked = editingChild == "Fabian", size = 18)
                                                    AppText(text = "Fabian", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                                }
                                                Row(
                                                    verticalAlignment = Alignment.CenterVertically,
                                                    modifier = Modifier.clickable { editingStatus = "aufgewacht" }
                                                ) {
                                                    AppCheckbox(checked = editingStatus == "aufgewacht", size = 18)
                                                    AppText(text = "Aufgewacht", style = AppTheme.typography.body, color = AppTheme.colors.text)
                                                }
                                            }
                                        }

                                        Row(
                                            modifier = Modifier.fillMaxWidth(),
                                            horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
                                        ) {
                                            CompactButton(
                                                text = editingOccurredAt.format(dateFormatter),
                                                modifier = Modifier.weight(1f),
                                                onClick = { showEditingDatePicker = true }
                                            )
                                            CompactButton(
                                                text = editingOccurredAt.format(timeFormatter),
                                                modifier = Modifier.weight(1f),
                                                onClick = { showEditingTimePicker = true }
                                            )
                                        }

                                        CompactTextField(
                                            value = editingNotes,
                                            onValueChange = { editingNotes = it },
                                            placeholder = "Optional notes",
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
                                                                status = editingStatus,
                                                                occurredAt = localDateTimeToIso(editingOccurredAt),
                                                                notes = editingNotes.trim()
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
                                                editingChild = if (entry.child == "Thomas") "Thomas" else "Fabian"
                                                editingStatus = if (entry.status == "aufgewacht") "aufgewacht" else "eingeschlafen"
                                                editingOccurredAt = entryToLocalDateTime(entry) ?: LocalDateTime.now().withSecond(0).withNano(0)
                                                editingNotes = entry.notes ?: ""
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
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            SectionTitle(text = "Average bed/wake")
                            CompactButton(
                                text = "Export",
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
                        }

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

    if (showDatePicker) {
        val dateState = rememberDatePickerState(
            initialSelectedDateMillis = occurredAt.atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
        )
        DatePickerDialog(
            onDismissRequest = { showDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    val selectedMillis = dateState.selectedDateMillis
                    if (selectedMillis != null) {
                        val selectedDate = Instant.ofEpochMilli(selectedMillis)
                            .atZone(ZoneId.systemDefault())
                            .toLocalDate()
                        occurredAt = occurredAt
                            .withYear(selectedDate.year)
                            .withMonth(selectedDate.monthValue)
                            .withDayOfMonth(selectedDate.dayOfMonth)
                    }
                    showDatePicker = false
                }) {
                    AppText(text = "OK", style = AppTheme.typography.label, color = AppTheme.colors.accent)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDatePicker = false }) {
                    AppText(text = "Cancel", style = AppTheme.typography.label, color = AppTheme.colors.muted)
                }
            }
        ) {
            DatePicker(state = dateState)
        }
    }

    if (showTimePicker) {
        val timeState = rememberTimePickerState(
            initialHour = occurredAt.hour,
            initialMinute = occurredAt.minute,
            is24Hour = true
        )
        AlertDialog(
            onDismissRequest = { showTimePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    occurredAt = occurredAt.withHour(timeState.hour).withMinute(timeState.minute)
                    showTimePicker = false
                }) {
                    AppText(text = "OK", style = AppTheme.typography.label, color = AppTheme.colors.accent)
                }
            },
            dismissButton = {
                TextButton(onClick = { showTimePicker = false }) {
                    AppText(text = "Cancel", style = AppTheme.typography.label, color = AppTheme.colors.muted)
                }
            },
            text = {
                TimePicker(state = timeState)
            }
        )
    }

    if (showEditingDatePicker) {
        val dateState = rememberDatePickerState(
            initialSelectedDateMillis = editingOccurredAt.atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
        )
        DatePickerDialog(
            onDismissRequest = { showEditingDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    val selectedMillis = dateState.selectedDateMillis
                    if (selectedMillis != null) {
                        val selectedDate = Instant.ofEpochMilli(selectedMillis)
                            .atZone(ZoneId.systemDefault())
                            .toLocalDate()
                        editingOccurredAt = editingOccurredAt
                            .withYear(selectedDate.year)
                            .withMonth(selectedDate.monthValue)
                            .withDayOfMonth(selectedDate.dayOfMonth)
                    }
                    showEditingDatePicker = false
                }) {
                    AppText(text = "OK", style = AppTheme.typography.label, color = AppTheme.colors.accent)
                }
            },
            dismissButton = {
                TextButton(onClick = { showEditingDatePicker = false }) {
                    AppText(text = "Cancel", style = AppTheme.typography.label, color = AppTheme.colors.muted)
                }
            }
        ) {
            DatePicker(state = dateState)
        }
    }

    if (showEditingTimePicker) {
        val timeState = rememberTimePickerState(
            initialHour = editingOccurredAt.hour,
            initialMinute = editingOccurredAt.minute,
            is24Hour = true
        )
        AlertDialog(
            onDismissRequest = { showEditingTimePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    editingOccurredAt = editingOccurredAt.withHour(timeState.hour).withMinute(timeState.minute)
                    showEditingTimePicker = false
                }) {
                    AppText(text = "OK", style = AppTheme.typography.label, color = AppTheme.colors.accent)
                }
            },
            dismissButton = {
                TextButton(onClick = { showEditingTimePicker = false }) {
                    AppText(text = "Cancel", style = AppTheme.typography.label, color = AppTheme.colors.muted)
                }
            },
            text = {
                TimePicker(state = timeState)
            }
        )
    }
}
