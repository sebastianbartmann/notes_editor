package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.Alignment
import androidx.compose.ui.text.input.ImeAction
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
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
    var taskInputMode by remember { mutableStateOf<String?>(null) }
    var taskInputText by remember { mutableStateOf("") }
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

    BackHandler(enabled = isEditing || taskInputMode != null) {
        if (taskInputMode != null) {
            taskInputMode = null
            taskInputText = ""
        } else {
            isEditing = false
        }
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
                        } else if (taskInputMode != null) {
                            val focusRequester = remember { FocusRequester() }
                            LaunchedEffect(Unit) {
                                focusRequester.requestFocus()
                            }
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(4.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                CompactTextField(
                                    value = taskInputText,
                                    onValueChange = { taskInputText = it },
                                    placeholder = "Task description",
                                    modifier = Modifier
                                        .weight(1f)
                                        .focusRequester(focusRequester),
                                    minLines = 1,
                                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                                    keyboardActions = KeyboardActions(
                                        onDone = {
                                            val category = taskInputMode
                                            if (category != null) {
                                                scope.launch {
                                                    try {
                                                        val response = ApiClient.addTodo(category, taskInputText)
                                                        message = response.message
                                                        refresh()
                                                    } catch (exc: Exception) {
                                                        message = "Add failed: ${exc.message}"
                                                    }
                                                }
                                            }
                                            taskInputMode = null
                                            taskInputText = ""
                                        }
                                    )
                                )
                                IconButton(
                                    onClick = {
                                        val category = taskInputMode
                                        if (category != null) {
                                            scope.launch {
                                                try {
                                                    val response = ApiClient.addTodo(category, taskInputText)
                                                    message = response.message
                                                    refresh()
                                                } catch (exc: Exception) {
                                                    message = "Add failed: ${exc.message}"
                                                }
                                            }
                                        }
                                        taskInputMode = null
                                        taskInputText = ""
                                    },
                                    modifier = Modifier.size(32.dp)
                                ) {
                                    Icon(
                                        Icons.Default.Check,
                                        contentDescription = "Save",
                                        tint = AppTheme.colors.accent
                                    )
                                }
                                IconButton(
                                    onClick = {
                                        taskInputMode = null
                                        taskInputText = ""
                                    },
                                    modifier = Modifier.size(32.dp)
                                ) {
                                    Icon(
                                        Icons.Default.Close,
                                        contentDescription = "Cancel",
                                        tint = AppTheme.colors.muted
                                    )
                                }
                            }
                        } else {
                            CompactButton(text = "Edit") { isEditing = true }
                            CompactTextButton(text = "Work task") { taskInputMode = "work" }
                            CompactTextButton(text = "Priv task") { taskInputMode = "priv" }
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
