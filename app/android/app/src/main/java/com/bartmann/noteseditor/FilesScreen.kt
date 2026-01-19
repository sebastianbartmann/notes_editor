package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.clickable
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.unit.dp
import androidx.activity.compose.BackHandler
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FilesScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var rootPath by remember { mutableStateOf(".") }
    var entriesByPath by remember { mutableStateOf(mapOf<String, List<FileEntry>>()) }
    var expandedDirs by remember { mutableStateOf(setOf<String>()) }
    var selectedFilePath by remember { mutableStateOf<String?>(null) }
    var fileContent by remember { mutableStateOf("") }
    var isEditing by remember { mutableStateOf(false) }
    var newFilePath by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    var isRefreshing by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun loadEntries(path: String) {
        scope.launch {
            try {
                val response = ApiClient.listFiles(path)
                entriesByPath = entriesByPath.toMutableMap().apply { put(path, response.entries) }
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Load failed: ${exc.message}"
            }
            isRefreshing = false
        }
    }

    fun refresh() {
        expandedDirs = emptySet()
        entriesByPath = emptyMap()
        loadEntries(rootPath)
    }

    fun openFile(path: String) {
        scope.launch {
            try {
                val response = ApiClient.readFile(path)
                selectedFilePath = response.path
                fileContent = response.content
                isEditing = false
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
            isEditing = false
            return
        }
        if (rootPath == "." || rootPath.isBlank()) return
        rootPath = parentPath(rootPath)
        refresh()
    }

    fun toggleDir(path: String) {
        if (expandedDirs.contains(path)) {
            expandedDirs = expandedDirs - path
        } else {
            expandedDirs = expandedDirs + path
            loadEntries(path)
        }
    }

    LaunchedEffect(rootPath) {
        refresh()
    }

    BackHandler(enabled = selectedFilePath != null) {
        if (isEditing) {
            isEditing = false
        } else {
            goBack()
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
            scrollable = selectedFilePath == null
        ) {
            val panelModifier = if (selectedFilePath != null) {
            Modifier
                .fillMaxWidth()
                .weight(1f)
        } else {
            Modifier.fillMaxWidth()
        }
        Panel(modifier = panelModifier) {
            if (selectedFilePath == null) {
                SectionTitle(text = "Create file")
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    CompactTextField(
                        value = newFilePath,
                        onValueChange = { newFilePath = it },
                        placeholder = "Vault path",
                        modifier = Modifier.weight(1f)
                    )
                    CompactButton(
                        text = "Create",
                        onClick = {
                            scope.launch {
                                try {
                                    val response = ApiClient.createFile(newFilePath.trim())
                                    message = response.message
                                    newFilePath = ""
                                    refresh()
                                } catch (exc: Exception) {
                                    message = "Create failed: ${exc.message}"
                                }
                            }
                        }
                    )
                }
                CompactDivider()
                FileTree(
                    entries = entriesByPath[rootPath].orEmpty(),
                    entriesByPath = entriesByPath,
                    expandedDirs = expandedDirs,
                    onToggleDir = ::toggleDir,
                    onOpenFile = ::openFile,
                    level = 0
                )
            } else {
                AppText(
                    text = if (isEditing) "Editing: $selectedFilePath" else "File: $selectedFilePath",
                    style = AppTheme.typography.label,
                    color = AppTheme.colors.muted
                )
                if (isEditing) {
                    CompactTextField(
                        value = fileContent,
                        onValueChange = { fileContent = it },
                        placeholder = "File content",
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f),
                        minLines = 7
                    )
                } else {
                    NoteView(
                        content = fileContent,
                        onToggleTask = {},
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                    )
                }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    if (!isEditing) {
                        CompactButton(
                            text = "Delete",
                            onClick = {
                                scope.launch {
                                    try {
                                        val response = ApiClient.deleteFile(selectedFilePath ?: "")
                                        message = response.message
                                        selectedFilePath = null
                                        fileContent = ""
                                        refresh()
                                    } catch (exc: Exception) {
                                        message = "Delete failed: ${exc.message}"
                                    }
                                }
                            },
                            background = AppTheme.colors.danger,
                            border = AppTheme.colors.danger,
                            textColor = AppTheme.colors.text
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs)) {
                        if (isEditing) {
                            CompactButton(
                                text = "Save",
                                onClick = {
                                    scope.launch {
                                        try {
                                            val response = ApiClient.saveFile(selectedFilePath ?: "", fileContent)
                                            message = response.message
                                            isEditing = false
                                        } catch (exc: Exception) {
                                            message = "Save failed: ${exc.message}"
                                        }
                                    }
                                }
                            )
                            CompactTextButton(text = "Cancel") { isEditing = false }
                        } else {
                            CompactButton(text = "Edit", onClick = { isEditing = true })
                        }
                    }
                }
            }

            StatusMessage(text = message)
        }
        }
    }
}

@Composable
private fun FileTree(
    entries: List<FileEntry>,
    entriesByPath: Map<String, List<FileEntry>>,
    expandedDirs: Set<String>,
    onToggleDir: (String) -> Unit,
    onOpenFile: (String) -> Unit,
    level: Int
) {
    entries.forEach { entry ->
        val isExpanded = entry.isDir && expandedDirs.contains(entry.path)
        val prefix = if (entry.isDir) {
            if (isExpanded) "-" else "+"
        } else {
            ""
        }
        val displayName = if (entry.isDir && !entry.name.endsWith("/")) {
            "${entry.name}/"
        } else {
            entry.name
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable {
                    if (entry.isDir) {
                        onToggleDir(entry.path)
                    } else {
                        onOpenFile(entry.path)
                    }
                }
                .padding(
                    PaddingValues(start = (level * 14).dp, top = 2.dp, bottom = 2.dp)
                ),
            horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs)
        ) {
            Box(modifier = Modifier.width(10.dp)) {
                AppText(
                    text = prefix,
                    style = AppTheme.typography.label,
                    color = AppTheme.colors.muted
                )
            }
            AppText(
                text = displayName,
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.text
            )
        }
        if (entry.isDir && isExpanded) {
            FileTree(
                entries = entriesByPath[entry.path].orEmpty(),
                entriesByPath = entriesByPath,
                expandedDirs = expandedDirs,
                onToggleDir = onToggleDir,
                onOpenFile = onOpenFile,
                level = level + 1
            )
        }
    }
}

private fun parentPath(path: String): String {
    val normalized = path.trim().trimEnd('/')
    val idx = normalized.lastIndexOf('/')
    return if (idx <= 0) "." else normalized.substring(0, idx)
}
