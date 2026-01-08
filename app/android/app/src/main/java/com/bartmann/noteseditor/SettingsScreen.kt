package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier

@Composable
fun SettingsScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    val currentPerson = UserSettings.person
    ScreenLayout(
        modifier = modifier,
        padding = padding
    ) {
        ScreenTitle(text = "Settings")
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
