package com.bartmann.noteseditor

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
import androidx.compose.ui.Modifier
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.TextButton
import androidx.compose.ui.graphics.Color
import java.time.Instant
import kotlinx.coroutines.launch

@Composable
fun SyncScreen(modifier: Modifier) {
    var output by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var showResetConfirm by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val syncSummary = syncHeaderSummary(AppSync.status)
    val indexSummary = indexHeaderSummary(AppSync.indexStatus)

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
            Column(verticalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs)) {
                AppText(
                    text = "Sync: ${syncSummary.text} (${syncSummary.detail})",
                    style = AppTheme.typography.bodySmall,
                    color = syncSummary.color
                )
                AppText(
                    text = "Index: ${indexSummary.text} (${indexSummary.detail})",
                    style = AppTheme.typography.bodySmall,
                    color = indexSummary.color
                )
            }
            CompactDivider()
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
                CompactTextButton(text = "Reset/Clean", modifier = Modifier.weight(1f)) {
                    if (!busy) showResetConfirm = true
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
            ) {
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

        if (showResetConfirm) {
            AlertDialog(
                onDismissRequest = { showResetConfirm = false },
                title = { AppText("Discard local changes?", AppTheme.typography.title, AppTheme.colors.text) },
                text = {
                    AppText(
                        "Reset/Clean will discard all local changes and remove untracked files.",
                        AppTheme.typography.bodySmall,
                        AppTheme.colors.muted
                    )
                },
                confirmButton = {
                    TextButton(onClick = {
                        showResetConfirm = false
                        if (!busy) runAction("Reset/Clean") { ApiClient.gitResetClean() }
                    }) {
                        AppText("Discard", AppTheme.typography.label, AppTheme.colors.danger)
                    }
                },
                dismissButton = {
                    TextButton(onClick = { showResetConfirm = false }) {
                        AppText("Cancel", AppTheme.typography.label, AppTheme.colors.muted)
                    }
                }
            )
        }
    }
}

private data class StatusSummary(
    val text: String,
    val detail: String,
    val color: Color
)

private fun syncHeaderSummary(status: SyncStatus?): StatusSummary {
    val now = System.currentTimeMillis()
    val lastPullMs = status?.lastPullAt?.let { runCatching { Instant.parse(it).toEpochMilli() }.getOrNull() }
    val lastPullAge = lastPullMs?.let { now - it } ?: Long.MAX_VALUE
    val lastErrMs = status?.lastErrorAt?.let { runCatching { Instant.parse(it).toEpochMilli() }.getOrNull() }
    val lastErrAge = lastErrMs?.let { now - it } ?: Long.MAX_VALUE

    val pending = status?.inProgress == true || status?.pendingPull == true || status?.pendingPush == true
    val stale = status?.lastPullAt == null || lastPullAge > 2 * 60_000
    val recentError = status?.lastError != null && lastErrAge <= 10 * 60_000
    val isSynced = status != null && !pending && !stale && !recentError

    return when {
        isSynced -> StatusSummary("synced", "recent pull", Color(0xFF2ECC71))
        status == null -> StatusSummary("not synced", "no status", Color(0xFFF1C40F))
        pending -> StatusSummary("not synced", "syncing", Color(0xFFF1C40F))
        recentError -> StatusSummary("not synced", "recent error", Color(0xFFF1C40F))
        status.lastPullAt == null -> StatusSummary("not synced", "never pulled", Color(0xFFF1C40F))
        stale -> StatusSummary("not synced", "stale", Color(0xFFF1C40F))
        else -> StatusSummary("not synced", "unknown", Color(0xFFF1C40F))
    }
}

private fun indexHeaderSummary(status: IndexStatus?): StatusSummary {
    return when {
        status == null -> StatusSummary("idle", "no status", Color(0xFF95A5A6))
        status.inProgress || status.pending -> {
            StatusSummary("running", status.lastReason ?: "working", Color(0xFFF1C40F))
        }
        !status.lastError.isNullOrBlank() -> StatusSummary("error", status.lastError, Color(0xFFE74C3C))
        !status.lastSuccessAt.isNullOrBlank() -> StatusSummary("ready", "last run succeeded", Color(0xFF2ECC71))
        else -> StatusSummary("idle", "no recent run", Color(0xFF95A5A6))
    }
}
