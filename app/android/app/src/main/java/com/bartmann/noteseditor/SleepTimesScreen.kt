package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.clickable
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
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SleepTimesScreen(modifier: Modifier) {
    var entries by remember { mutableStateOf(listOf<SleepEntry>()) }
    var child by remember { mutableStateOf("Fabian") }
    var entryText by remember { mutableStateOf("") }
    var asleep by remember { mutableStateOf(false) }
    var woke by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf("") }
    var isRefreshing by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun refresh() {
        scope.launch {
            try {
                AppSync.syncIfStale(timeoutMs = 2_000, maxAgeMs = 30_000)
                val response = ApiClient.fetchSleepTimes()
                entries = response.entries
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
                        AppCheckbox(
                            checked = child == "Thomas",
                            size = 18
                        )
                        AppText(
                            text = "Thomas",
                            style = AppTheme.typography.body,
                            color = AppTheme.colors.text
                        )
                    }
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.clickable {
                            asleep = !asleep
                            if (asleep) {
                                woke = false
                            }
                        }
                    ) {
                        AppCheckbox(
                            checked = asleep,
                            size = 18
                        )
                        AppText(
                            text = "Eingeschlafen",
                            style = AppTheme.typography.body,
                            color = AppTheme.colors.text
                        )
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
                        AppCheckbox(
                            checked = child == "Fabian",
                            size = 18
                        )
                        AppText(
                            text = "Fabian",
                            style = AppTheme.typography.body,
                            color = AppTheme.colors.text
                        )
                    }
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.clickable {
                            woke = !woke
                            if (woke) {
                                asleep = false
                            }
                        }
                    ) {
                        AppCheckbox(
                            checked = woke,
                            size = 18
                        )
                        AppText(
                            text = "Aufgewacht",
                            style = AppTheme.typography.body,
                            color = AppTheme.colors.text
                        )
                    }
                }
            }
            CompactTextField(
                value = entryText,
                onValueChange = { entryText = it },
                placeholder = "Entry (19:30-06:10 | night)",
                modifier = Modifier.fillMaxWidth()
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End
            ) {
                CompactButton(
                    text = "Append",
                    onClick = {
                        val status = when {
                            asleep -> "eingeschlafen"
                            woke -> "aufgewacht"
                            else -> {
                                message = "Please select eingeschlafen or aufgewacht"
                                return@CompactButton
                            }
                        }
                        scope.launch {
                            try {
                                val response = ApiClient.appendSleepTimes(child, entryText, status)
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
                )
            }
            StatusMessage(text = message)
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
                            text = "${entry.date} | ${entry.child} | ${entry.time} | ${entry.status}",
                            style = AppTheme.typography.bodySmall,
                            color = AppTheme.colors.text,
                            modifier = Modifier.weight(1f)
                        )
                        CompactButton(
                            text = "Delete",
                            background = AppTheme.colors.danger,
                            border = AppTheme.colors.danger,
                            textColor = AppTheme.colors.text,
                            onClick = {
                                scope.launch {
                                    try {
                                        val response = ApiClient.deleteSleepEntry(entry.line)
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
    }
}
