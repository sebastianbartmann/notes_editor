package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.ui.Modifier
import kotlinx.coroutines.launch

@Composable
fun SyncScreen(modifier: Modifier) {
    var output by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun refreshStatus() {
        scope.launch {
            try {
                val res = ApiClient.fetchGitStatus()
                output = if (res.output.isBlank()) "(clean)" else res.output
            } catch (exc: Exception) {
                error = "Status failed: ${exc.message}"
            }
        }
    }

    fun runAction(label: String, action: suspend () -> GitActionResponse) {
        scope.launch {
            busy = true
            error = ""
            try {
                val res = action()
                message = "$label: ${res.message}"
                output = if (res.output.isNullOrBlank()) "(clean)" else res.output
                AppSync.refreshStatus()
            } catch (exc: Exception) {
                error = "$label failed: ${exc.message}"
                refreshStatus()
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(Unit) {
        refreshStatus()
    }

    ScreenLayout(
        modifier = modifier.fillMaxSize(),
        scrollable = false
    ) {
        ScreenHeader(title = "Sync")

        Panel(modifier = Modifier.fillMaxWidth(), fill = false) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
            ) {
                CompactTextButton(text = "Commit+Push", modifier = Modifier.weight(1f)) {
                    if (!busy) runAction("Commit+Push") { ApiClient.gitCommitPush() }
                }
                CompactTextButton(text = "Commit", modifier = Modifier.weight(1f)) {
                    if (!busy) runAction("Commit") { ApiClient.gitCommit() }
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
            ) {
                CompactTextButton(text = "Push", modifier = Modifier.weight(1f)) {
                    if (!busy) runAction("Push") { ApiClient.gitPush() }
                }
                CompactTextButton(text = "Pull", modifier = Modifier.weight(1f)) {
                    if (!busy) runAction("Pull") { ApiClient.gitPull() }
                }
                CompactTextButton(text = "Refresh", modifier = Modifier.weight(1f)) {
                    if (!busy) {
                        error = ""
                        refreshStatus()
                    }
                }
            }
            if (message.isNotBlank()) {
                AppText(
                    text = message,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.muted
                )
            }
            if (error.isNotBlank()) {
                AppText(
                    text = error,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.danger
                )
            }
        }

        Panel(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f),
            fill = false
        ) {
            AppText(
                text = output.ifBlank { "(empty)" },
                style = AppTheme.typography.body,
                color = AppTheme.colors.text
            )
        }
    }
}
