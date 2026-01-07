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
fun DailyScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var content by remember { mutableStateOf("") }
    var appendText by remember { mutableStateOf("") }
    var pinned by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf("") }
    var path by remember { mutableStateOf("") }
    var date by remember { mutableStateOf("") }
    var todos by remember { mutableStateOf(listOf<TodoItem>()) }
    var pinnedItems by remember { mutableStateOf(listOf<PinnedItem>()) }
    val scope = rememberCoroutineScope()

    fun refresh() {
        scope.launch {
            try {
                val daily = ApiClient.fetchDaily()
                content = daily.content
                path = daily.path
                date = daily.date
                todos = parseTodos(daily.content)
                pinnedItems = parsePinned(daily.content)
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Failed to load: ${exc.message}"
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
        ScreenTitle(text = "Daily $date")
        Panel {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Refresh") { refresh() }
                CompactButton(text = "Save") {
                    scope.launch {
                        try {
                            val response = ApiClient.saveDaily(content)
                            message = response.message
                        } catch (exc: Exception) {
                            message = "Save failed: ${exc.message}"
                        }
                    }
                }
                CompactTextButton(text = "Clear pinned") {
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
            }
            CompactDivider()
            SectionTitle(text = "Current note")
            CompactOutlinedTextField(
                value = content,
                onValueChange = { content = it },
                label = "Daily note",
                modifier = Modifier.fillMaxWidth(),
                minLines = 7
            )
            CompactDivider()
            SectionTitle(text = "Quick append")
            CompactOutlinedTextField(
                value = appendText,
                onValueChange = { appendText = it },
                label = "Append text",
                modifier = Modifier.fillMaxWidth(),
                minLines = 3
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Row {
                    Checkbox(
                        checked = pinned,
                        onCheckedChange = { pinned = it },
                        colors = CheckboxDefaults.colors(
                            checkedColor = MaterialTheme.colorScheme.primary,
                            uncheckedColor = MaterialTheme.colorScheme.secondary
                        )
                    )
                    Text(text = "Pinned")
                }
                CompactButton(text = "Append") {
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
        Panel {
            SectionTitle(text = "Todos")
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Add work") {
                    scope.launch {
                        try {
                            val response = ApiClient.addTodo("work")
                            message = response.message
                            refresh()
                        } catch (exc: Exception) {
                            message = "Add failed: ${exc.message}"
                        }
                    }
                }
                CompactButton(text = "Add priv") {
                    scope.launch {
                        try {
                            val response = ApiClient.addTodo("priv")
                            message = response.message
                            refresh()
                        } catch (exc: Exception) {
                            message = "Add failed: ${exc.message}"
                        }
                    }
                }
            }
            if (todos.isEmpty()) {
                Text(text = "No todos found.", color = MaterialTheme.colorScheme.secondary)
            } else {
                todos.forEach { todo ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Checkbox(
                            checked = todo.done,
                            onCheckedChange = {
                                if (path.isNotBlank()) {
                                    scope.launch {
                                        try {
                                            val response = ApiClient.toggleTodo(path, todo.lineNo)
                                            message = response.message
                                            refresh()
                                        } catch (exc: Exception) {
                                            message = "Toggle failed: ${exc.message}"
                                        }
                                    }
                                }
                            },
                            colors = CheckboxDefaults.colors(
                                checkedColor = MaterialTheme.colorScheme.primary,
                                uncheckedColor = MaterialTheme.colorScheme.secondary
                            )
                        )
                        Text(text = todo.text)
                    }
                }
            }
        }
        Panel {
            SectionTitle(text = "Pinned entries")
            if (pinnedItems.isEmpty()) {
                Text(text = "No pinned entries found.", color = MaterialTheme.colorScheme.secondary)
            } else {
                pinnedItems.forEach { pinnedItem ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Text(text = pinnedItem.header, modifier = Modifier.weight(1f))
                        CompactTextButton(text = "Unpin") {
                            if (path.isNotBlank()) {
                                scope.launch {
                                    try {
                                        val response = ApiClient.unpinEntry(path, pinnedItem.lineNo)
                                        message = response.message
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Unpin failed: ${exc.message}"
                                    }
                                }
                            }
                        }
                    }
                }
            }
            if (message.isNotBlank()) {
                CompactDivider()
                Text(text = message)
            }
        }
    }
}
