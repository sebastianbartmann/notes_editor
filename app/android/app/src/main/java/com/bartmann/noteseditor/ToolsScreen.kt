package com.bartmann.noteseditor

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.OpenInNew
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.unit.dp

@Composable
fun ToolsScreen(
    modifier: Modifier,
    onNavigate: (String) -> Unit
) {
    val uriHandler = LocalUriHandler.current
    val hiddenIds = UserSettings.bottomNavIds.toSet()
    val items = selectableNavEntries.filter { it.id !in hiddenIds }

    ScreenLayout(modifier = modifier) {
        ScreenHeader(title = "More")

        Panel(fill = false) {
            items.forEach { item ->
                ToolRow(
                    item = item,
                    onClick = {
                        if (item.externalUrl != null) {
                            uriHandler.openUri(item.externalUrl)
                        } else if (item.route != null) {
                            onNavigate(item.route)
                        }
                    }
                )
            }
        }
    }
}

@Composable
private fun ToolRow(
    item: NavEntry,
    onClick: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .background(AppTheme.colors.input, RoundedCornerShape(6.dp))
            .padding(horizontal = AppTheme.spacing.sm, vertical = AppTheme.spacing.sm),
        horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Image(
            painter = rememberVectorPainter(item.icon),
            contentDescription = item.label,
            colorFilter = ColorFilter.tint(AppTheme.colors.accent),
            modifier = Modifier.size(24.dp)
        )
        AppText(
            text = item.label,
            style = AppTheme.typography.body,
            color = AppTheme.colors.text,
            modifier = Modifier.weight(1f)
        )
        if (item.externalUrl != null) {
            Image(
                painter = rememberVectorPainter(Icons.AutoMirrored.Filled.OpenInNew),
                contentDescription = "Opens in browser",
                colorFilter = ColorFilter.tint(AppTheme.colors.muted),
                modifier = Modifier.size(16.dp)
            )
        }
    }
}
