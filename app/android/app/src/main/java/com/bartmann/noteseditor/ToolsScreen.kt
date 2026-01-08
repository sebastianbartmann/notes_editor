package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@Composable
fun ToolsScreen(
    modifier: Modifier,
    padding: androidx.compose.foundation.layout.PaddingValues,
    onOpenClaude: () -> Unit,
    onOpenNoise: () -> Unit,
    onOpenNotifications: () -> Unit,
    onOpenSettings: () -> Unit
) {
    ScreenLayout(
        modifier = modifier,
        padding = padding
    ) {
        ScreenTitle(text = "Tools")
        Panel {
            ToolGrid(
                tools = listOf(
                    "Claude" to onOpenClaude,
                    "Noise" to onOpenNoise,
                    "Notifications" to onOpenNotifications,
                    "Settings" to onOpenSettings
                )
            )
        }
    }
}

@Composable
private fun ToolGrid(tools: List<Pair<String, () -> Unit>>) {
    val rows = tools.chunked(2)
    rows.forEach { row ->
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
        ) {
            row.forEach { (label, onClick) ->
                CompactButton(
                    text = label,
                    onClick = onClick,
                    modifier = Modifier
                        .weight(1f)
                        .height(72.dp),
                    background = AppTheme.colors.input,
                    border = AppTheme.colors.panelBorder,
                    textColor = AppTheme.colors.text
                )
            }
            if (row.size == 1) {
                androidx.compose.foundation.layout.Spacer(modifier = Modifier.weight(1f))
            }
        }
    }
}
