package com.bartmann.noteseditor

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.unit.dp
import java.io.IOException
import kotlinx.coroutines.launch

@Composable
fun SettingsScreen(modifier: Modifier) {
    val currentPerson = UserSettings.person
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var envContent by remember { mutableStateOf("") }
    var envStatus by remember { mutableStateOf("") }
    var isSavingEnv by remember { mutableStateOf(false) }
    var runtimeMode by remember { mutableStateOf("gateway_subscription") }
    var agentPrompt by remember { mutableStateOf("") }
    var agentStatus by remember { mutableStateOf("") }
    var isSavingAgent by remember { mutableStateOf(false) }
    var backupStatus by remember { mutableStateOf("") }
    var isSavingBackup by remember { mutableStateOf(false) }
    var navStatus by remember { mutableStateOf("") }
    val selectedNavIds = UserSettings.bottomNavIds
    val backupLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.CreateDocument("application/zip")
    ) { uri ->
        if (uri == null) {
            isSavingBackup = false
            backupStatus = "Backup canceled"
            return@rememberLauncherForActivityResult
        }
        scope.launch {
            try {
                val output = context.contentResolver.openOutputStream(uri)
                    ?: throw IOException("Unable to open destination file")
                output.use {
                    ApiClient.downloadVaultBackupTo(it)
                }
                backupStatus = "Backup saved"
            } catch (exc: Exception) {
                backupStatus = "Backup failed: ${exc.message}"
            } finally {
                isSavingBackup = false
            }
        }
    }

    fun moveNavItem(id: String, delta: Int) {
        val currentIndex = selectedNavIds.indexOf(id)
        if (currentIndex == -1) return
        val targetIndex = currentIndex + delta
        if (targetIndex !in selectedNavIds.indices) return
        val updated = selectedNavIds.toMutableList()
        val item = updated.removeAt(currentIndex)
        updated.add(targetIndex, item)
        UserSettings.updateBottomNav(updated)
    }

    LaunchedEffect(Unit) {
        try {
            val response = ApiClient.fetchEnv()
            envContent = response.content
            envStatus = ""
        } catch (exc: Exception) {
            envStatus = "Failed to load .env: ${exc.message}"
        }
    }

    LaunchedEffect(currentPerson) {
        if (currentPerson == null) return@LaunchedEffect
        try {
            val config = ApiClient.fetchAgentConfig()
            runtimeMode = config.runtimeMode
            agentPrompt = config.prompt
            agentStatus = ""
        } catch (exc: Exception) {
            agentStatus = "Failed to load agent config: ${exc.message}"
        }
    }

    ScreenLayout(modifier = modifier) {
        ScreenHeader(title = "Settings")

        Panel {
            SectionTitle(text = "Person")
            AppText(
                text = "Choose the person root for this device.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                PersonButton(label = "Sebastian", value = "sebastian", currentPerson = currentPerson)
                PersonButton(label = "Petra", value = "petra", currentPerson = currentPerson)
            }
            if (currentPerson == null) {
                AppText(
                    text = "Select a person to unlock the rest of the app.",
                    style = AppTheme.typography.label,
                    color = AppTheme.colors.muted
                )
            }
            CompactDivider()
            SectionTitle(text = "Theme")
            AppText(
                text = "Pick a color theme for this device.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                ThemeButton(label = "Dark", value = "dark")
                ThemeButton(label = "Light", value = "light")
            }
            CompactDivider()
            SectionTitle(text = "Text size")
            AppText(
                text = "Global reading/editing size for notes, files, and agent chat.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CompactButton(
                    text = "A-",
                    modifier = Modifier
                        .width(64.dp)
                        .testTag("text-scale-down"),
                    onClick = {
                        if (UserSettings.textScale > MIN_TEXT_SCALE) {
                            UserSettings.stepTextScale(-1)
                        }
                    }
                )
                CompactButton(
                    text = "Reset",
                    modifier = Modifier.weight(1f),
                    onClick = { UserSettings.resetTextScale() }
                )
                CompactButton(
                    text = "A+",
                    modifier = Modifier
                        .width(64.dp)
                        .testTag("text-scale-up"),
                    onClick = {
                        if (UserSettings.textScale < MAX_TEXT_SCALE) {
                            UserSettings.stepTextScale(1)
                        }
                    }
                )
            }
            AppText(
                text = "Current: ${textScaleLabel(UserSettings.textScale)} (${textScaleStepLabel()})",
                style = AppTheme.typography.label,
                color = AppTheme.colors.muted,
                modifier = Modifier.testTag("text-scale-current")
            )
            CompactDivider()
            SectionTitle(text = "Agent")
            AppText(
                text = "Per-person runtime mode and system prompt (agents.md).",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                RuntimeModeButton(
                    label = "Anthropic",
                    value = "anthropic_api_key",
                    runtimeMode = runtimeMode,
                    onSelect = { runtimeMode = "anthropic_api_key" }
                )
                RuntimeModeButton(
                    label = "Gateway (Pi)",
                    value = "gateway_subscription",
                    runtimeMode = runtimeMode,
                    onSelect = { runtimeMode = "gateway_subscription" }
                )
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(AppTheme.colors.input, RoundedCornerShape(6.dp))
                    .clickable { UserSettings.updateAgentVerboseOutput(!UserSettings.agentVerboseOutput) }
                    .padding(horizontal = AppTheme.spacing.sm, vertical = AppTheme.spacing.sm),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                AppCheckbox(checked = UserSettings.agentVerboseOutput)
                AppText(
                    text = "Enable verbose output (tool calls, gateway status, usage)",
                    style = AppTheme.typography.body,
                    color = AppTheme.colors.text
                )
            }
            CompactTextField(
                value = agentPrompt,
                onValueChange = { agentPrompt = it },
                placeholder = "Enter agents.md prompt...",
                modifier = Modifier
                    .fillMaxWidth()
                    .height(180.dp),
                minLines = 6
            )
            CompactButton(
                text = if (isSavingAgent) "Saving..." else "Save agent",
                modifier = Modifier.fillMaxWidth(),
                onClick = {
                    if (isSavingAgent || currentPerson == null) return@CompactButton
                    isSavingAgent = true
                    agentStatus = ""
                    scope.launch {
                        try {
                            ApiClient.saveAgentConfig(runtimeMode, agentPrompt)
                            agentStatus = "Saved agent settings"
                        } catch (exc: Exception) {
                            agentStatus = "Save failed: ${exc.message}"
                        } finally {
                            isSavingAgent = false
                        }
                    }
                }
            )
            if (agentStatus.isNotBlank()) {
                StatusMessage(text = agentStatus, showDivider = false)
            }
            CompactDivider()
            SectionTitle(text = "Backup")
            AppText(
                text = "Download a compressed copy of the selected person's vault.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            CompactButton(
                text = if (isSavingBackup) "Preparing backup..." else "Download backup (.zip)",
                modifier = Modifier.fillMaxWidth(),
                onClick = {
                    if (isSavingBackup || currentPerson == null) return@CompactButton
                    isSavingBackup = true
                    backupStatus = ""
                    backupLauncher.launch("${currentPerson}-vault-backup.zip")
                }
            )
            if (backupStatus.isNotBlank()) {
                StatusMessage(text = backupStatus, showDivider = false)
            }
            CompactDivider()
            SectionTitle(text = "Navigation")
            AppText(
                text = "Choose up to $maxBottomNavItems items for the footer. The rest show under More.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            selectableNavEntries.forEach { item ->
                val isSelected = selectedNavIds.contains(item.id)
                val isBlocked = !isSelected && selectedNavIds.size >= maxBottomNavItems
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(AppTheme.colors.input, RoundedCornerShape(6.dp))
                        .clickable {
                            navStatus = ""
                            if (isSelected) {
                                UserSettings.updateBottomNav(selectedNavIds.filter { it != item.id })
                            } else if (!isBlocked) {
                                UserSettings.updateBottomNav(selectedNavIds + item.id)
                            } else {
                                navStatus = "Remove one item to add another."
                            }
                        }
                        .padding(horizontal = AppTheme.spacing.sm, vertical = AppTheme.spacing.sm),
                    horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    AppCheckbox(checked = isSelected)
                    Image(
                        painter = rememberVectorPainter(item.icon),
                        contentDescription = item.label,
                        colorFilter = ColorFilter.tint(AppTheme.colors.accent),
                        modifier = Modifier.size(20.dp)
                    )
                    AppText(
                        text = item.label,
                        style = AppTheme.typography.body,
                        color = if (isBlocked) AppTheme.colors.muted else AppTheme.colors.text,
                        modifier = Modifier.weight(1f)
                    )
                    if (isSelected) {
                        val index = selectedNavIds.indexOf(item.id)
                        val canMoveUp = index > 0
                        val canMoveDown = index < selectedNavIds.lastIndex
                        IconButton(
                            onClick = { moveNavItem(item.id, -1) },
                            enabled = canMoveUp
                        ) {
                            Icon(
                                imageVector = Icons.Default.KeyboardArrowUp,
                                contentDescription = "Move up",
                                tint = if (canMoveUp) AppTheme.colors.accent else AppTheme.colors.muted
                            )
                        }
                        IconButton(
                            onClick = { moveNavItem(item.id, 1) },
                            enabled = canMoveDown
                        ) {
                            Icon(
                                imageVector = Icons.Default.KeyboardArrowDown,
                                contentDescription = "Move down",
                                tint = if (canMoveDown) AppTheme.colors.accent else AppTheme.colors.muted
                            )
                        }
                    }
                }
            }
            AppText(
                text = "${selectedNavIds.size} of $maxBottomNavItems selected",
                style = AppTheme.typography.label,
                color = AppTheme.colors.muted
            )
            if (navStatus.isNotBlank()) {
                StatusMessage(text = navStatus, showDivider = false)
            }
        }
        Panel(fill = false) {
            SectionTitle(text = "Server .env")
            AppText(
                text = "Edit environment variables stored on the server.",
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.muted
            )
            CompactTextField(
                value = envContent,
                onValueChange = { envContent = it },
                placeholder = "LINKEDIN_ACCESS_TOKEN=...\nLINKEDIN_CLIENT_ID=...\n",
                modifier = Modifier
                    .fillMaxWidth()
                    .height(220.dp),
                minLines = 8
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CompactButton(
                    text = if (isSavingEnv) "Saving..." else "Save",
                    modifier = Modifier.weight(1f),
                    onClick = {
                        if (isSavingEnv) return@CompactButton
                        isSavingEnv = true
                        envStatus = ""
                        scope.launch {
                            try {
                                ApiClient.saveEnv(envContent)
                                envStatus = "Saved .env"
                            } catch (exc: Exception) {
                                envStatus = "Save failed: ${exc.message}"
                            } finally {
                                isSavingEnv = false
                            }
                        }
                    }
                )
            }
            if (envStatus.isNotBlank()) {
                StatusMessage(text = envStatus, showDivider = false)
            }
        }
    }
}

@Composable
private fun RowScope.PersonButton(label: String, value: String, currentPerson: String?) {
    val selected = currentPerson == value
    CompactButton(
        text = label,
        modifier = Modifier.weight(1f),
        background = if (selected) AppTheme.colors.accentDim else AppTheme.colors.input,
        border = if (selected) AppTheme.colors.accent else AppTheme.colors.panelBorder,
        textColor = AppTheme.colors.text,
        onClick = { UserSettings.updatePerson(value) }
    )
}

@Composable
private fun RowScope.ThemeButton(label: String, value: String) {
    val selected = UserSettings.theme == value
    CompactButton(
        text = label,
        modifier = Modifier.weight(1f),
        background = if (selected) AppTheme.colors.accentDim else AppTheme.colors.input,
        border = if (selected) AppTheme.colors.accent else AppTheme.colors.panelBorder,
        textColor = AppTheme.colors.text,
        onClick = { UserSettings.updateTheme(value) }
    )
}

private fun textScaleLabel(scale: Float): String {
    val pct = (scale * 100f).toInt()
    return "$pct%"
}

private fun textScaleStepLabel(): String {
    val stepPct = (TEXT_SCALE_STEP * 100f).toInt()
    return "step ${stepPct}%"
}

@Composable
private fun RowScope.RuntimeModeButton(
    label: String,
    value: String,
    runtimeMode: String,
    onSelect: () -> Unit
) {
    val selected = runtimeMode == value
    CompactButton(
        text = label,
        modifier = Modifier.weight(1f),
        background = if (selected) AppTheme.colors.accentDim else AppTheme.colors.input,
        border = if (selected) AppTheme.colors.accent else AppTheme.colors.panelBorder,
        textColor = AppTheme.colors.text,
        onClick = onSelect
    )
}
