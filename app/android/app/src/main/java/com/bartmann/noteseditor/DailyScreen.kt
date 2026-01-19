package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.ui.Alignment
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DailyScreen(
    modifier: Modifier,
    padding: androidx.compose.foundation.layout.PaddingValues
) {
    var content by remember { mutableStateOf("") }
    var appendText by remember { mutableStateOf("") }
    var pinned by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf("") }
    var path by remember { mutableStateOf("") }
    var date by remember { mutableStateOf("") }
    var isEditing by remember { mutableStateOf(false) }
    var isRefreshing by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun refresh(keepEditing: Boolean = false) {
        scope.launch {
            try {
                val daily = ApiClient.fetchDaily()
                content = daily.content
                path = daily.path
                date = daily.date
                if (!keepEditing) {
                    isEditing = false
                }
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Failed to load: ${exc.message}"
            }
            isRefreshing = false
        }
    }

    LaunchedEffect(Unit) {
        refresh()
    }

    BackHandler(enabled = isEditing) {
        isEditing = false
    }

    PullToRefreshBox(
        isRefreshing = isRefreshing,
        onRefresh = {
            isRefreshing = true
            refresh()
        },
        modifier = modifier.fillMaxSize()
    ) {
        ScreenLayout(
            modifier = Modifier,
            padding = padding,
            scrollable = false
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End
            ) {
                AppText(
                    text = "Today: $date",
                    style = AppTheme.typography.label,
                    color = AppTheme.colors.muted
                )
            }
            Panel(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
            ) {
                SectionTitle(text = "Current note")
                if (isEditing) {
                    CompactTextField(
                        value = content,
                        onValueChange = { content = it },
                        placeholder = "Edit note",
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f),
                        minLines = 10
                    )
                } else {
                    NoteView(
                        content = content,
                        onToggleTask = { lineNo ->
                            if (path.isNotBlank()) {
                                scope.launch {
                                    try {
                                        val response = ApiClient.toggleTodo(path, lineNo)
                                        message = response.message
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Toggle failed: ${exc.message}"
                                    }
                                }
                            }
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                    )
                }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End
                ) {
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        if (isEditing) {
                            CompactButton(text = "Save") {
                                scope.launch {
                                    try {
                                        val response = ApiClient.saveDaily(content)
                                        message = response.message
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Save failed: ${exc.message}"
                                    }
                                }
                            }
                            CompactTextButton(text = "Cancel") { isEditing = false }
                        } else {
                            CompactButton(text = "Edit") { isEditing = true }
                            CompactTextButton(text = "Work task") {
                                scope.launch {
                                    try {
                                        val response = ApiClient.addTodo("work")
                                        message = response.message
                                        refresh(keepEditing = true)
                                        isEditing = true
                                    } catch (exc: Exception) {
                                        message = "Add failed: ${exc.message}"
                                    }
                                }
                            }
                            CompactTextButton(text = "Priv task") {
                                scope.launch {
                                    try {
                                        val response = ApiClient.addTodo("priv")
                                        message = response.message
                                        refresh(keepEditing = true)
                                        isEditing = true
                                    } catch (exc: Exception) {
                                        message = "Add failed: ${exc.message}"
                                    }
                                }
                            }
                        }
                    }
                }
                if (!isEditing) {
                    CompactDivider()
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        CompactButton(
                            text = "Clear",
                            background = AppTheme.colors.danger,
                            border = AppTheme.colors.danger,
                            textColor = AppTheme.colors.text,
                            onClick = {
                                scope.launch {
                                    try {
                                        val response = ApiClient.clearPinned()
                                        message = response.message
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Clear failed: ${exc.message}"
                                    }
                                }
                            }
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier.clickable { pinned = !pinned }
                            ) {
                                AppCheckbox(
                                    checked = pinned
                                )
                                AppText(
                                    text = "Pin",
                                    style = AppTheme.typography.label,
                                    color = AppTheme.colors.text
                                )
                            }
                            CompactButton(text = "Add") {
                                scope.launch {
                                    try {
                                        val response = ApiClient.appendDaily(appendText, pinned)
                                        message = response.message
                                        appendText = ""
                                        pinned = false
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Append failed: ${exc.message}"
                                    }
                                }
                            }
                        }
                    }
                    CompactTextField(
                        value = appendText,
                        onValueChange = { appendText = it },
                        placeholder = "Write something...",
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 6
                    )
                }
                StatusMessage(text = message, showDivider = false)
            }
        }
    }
}
