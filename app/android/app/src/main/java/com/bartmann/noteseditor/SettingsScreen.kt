package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
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
import kotlinx.coroutines.launch

@Composable
fun SettingsScreen(modifier: Modifier) {
    val currentPerson = UserSettings.person
    val scope = rememberCoroutineScope()
    var envContent by remember { mutableStateOf("") }
    var envStatus by remember { mutableStateOf("") }
    var isSavingEnv by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        try {
            val response = ApiClient.fetchEnv()
            envContent = response.content
            envStatus = ""
        } catch (exc: Exception) {
            envStatus = "Failed to load .env: ${exc.message}"
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
