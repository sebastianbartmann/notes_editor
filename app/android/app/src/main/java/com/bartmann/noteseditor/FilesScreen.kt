package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
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
fun FilesScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var currentPath by remember { mutableStateOf(".") }
    var entries by remember { mutableStateOf(listOf<FileEntry>()) }
    var selectedFilePath by remember { mutableStateOf<String?>(null) }
    var fileContent by remember { mutableStateOf("") }
    var newFilePath by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    var pinnedItems by remember { mutableStateOf(listOf<PinnedItem>()) }
    val scope = rememberCoroutineScope()

    fun loadEntries() {
        scope.launch {
            try {
                val response = ApiClient.listFiles(currentPath)
                entries = response.entries
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Load failed: ${exc.message}"
            }
        }
    }

    fun openFile(path: String) {
        scope.launch {
            try {
                val response = ApiClient.readFile(path)
                selectedFilePath = response.path
                fileContent = response.content
                pinnedItems = parsePinned(response.content)
                message = "Loaded file."
            } catch (exc: Exception) {
                message = "Read failed: ${exc.message}"
            }
        }
    }

    fun goBack() {
        if (selectedFilePath != null) {
            selectedFilePath = null
            fileContent = ""
            pinnedItems = emptyList()
            return
        }
        if (currentPath == "." || currentPath.isBlank()) return
        currentPath = parentPath(currentPath)
        loadEntries()
    }

    LaunchedEffect(currentPath) {
        loadEntries()
    }

    Column(
        modifier = modifier
            .padding(padding)
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        ScreenTitle(text = "Files")
        Panel {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Refresh") { loadEntries() }
                CompactButton(text = "Back") { goBack() }
            }

            if (selectedFilePath == null) {
                Text(text = "Path: $currentPath", color = MaterialTheme.colorScheme.secondary)
                CompactDivider()
                entries.forEach { entry ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Text(text = if (entry.isDir) "[DIR] ${entry.name}" else "[FILE] ${entry.name}")
                        CompactTextButton(text = "Open") {
                            if (entry.isDir) {
                                currentPath = entry.path
                            } else {
                                openFile(entry.path)
                            }
                        }
                    }
                }
                CompactDivider()
                SectionTitle(text = "Create file")
                CompactOutlinedTextField(
                    value = newFilePath,
                    onValueChange = { newFilePath = it },
                    label = "Vault path",
                    modifier = Modifier.fillMaxWidth()
                )
                CompactButton(text = "Create") {
                    scope.launch {
                        try {
                            val response = ApiClient.createFile(newFilePath.trim())
                            message = response.message
                            newFilePath = ""
                            loadEntries()
                        } catch (exc: Exception) {
                            message = "Create failed: ${exc.message}"
                        }
                    }
                }
            } else {
                Text(text = "Editing: $selectedFilePath", color = MaterialTheme.colorScheme.secondary)
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    CompactButton(text = "Save") {
                        scope.launch {
                            try {
                                val response = ApiClient.saveFile(selectedFilePath ?: "", fileContent)
                                message = response.message
                            } catch (exc: Exception) {
                                message = "Save failed: ${exc.message}"
                            }
                        }
                    }
                    CompactTextButton(text = "Delete") {
                        scope.launch {
                            try {
                                val response = ApiClient.deleteFile(selectedFilePath ?: "")
                                message = response.message
                                selectedFilePath = null
                                fileContent = ""
                                pinnedItems = emptyList()
                                loadEntries()
                            } catch (exc: Exception) {
                                message = "Delete failed: ${exc.message}"
                            }
                        }
                    }
                }
                CompactOutlinedTextField(
                    value = fileContent,
                    onValueChange = { fileContent = it },
                    label = "File content",
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 7
                )
                CompactDivider()
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
                                scope.launch {
                                    try {
                                        val response = ApiClient.unpinEntry(
                                            selectedFilePath ?: "",
                                            pinnedItem.lineNo
                                        )
                                        message = response.message
                                        openFile(selectedFilePath ?: "")
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
                Text(text = message, color = MaterialTheme.colorScheme.secondary)
            }
        }
    }
}

private fun parentPath(path: String): String {
    val normalized = path.trim().trimEnd('/')
    val idx = normalized.lastIndexOf('/')
    return if (idx <= 0) "." else normalized.substring(0, idx)
}
