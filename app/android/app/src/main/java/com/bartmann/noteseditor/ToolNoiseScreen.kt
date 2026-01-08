package com.bartmann.noteseditor

import android.content.Context
import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat

@Composable
fun ToolNoiseScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    val context = LocalContext.current
    var noisePlaying by remember { mutableStateOf(false) }

    ScreenLayout(
        modifier = modifier,
        padding = padding
    ) {
        ScreenTitle(text = "Noise")
        Panel {
            SectionTitle(text = "Playback")
            Row(horizontalArrangement = Arrangement.spacedBy(AppTheme.spacing.xs)) {
                CompactButton(text = "Play") {
                    startNoise(context)
                    noisePlaying = true
                }
                CompactButton(text = "Stop") {
                    stopNoise(context)
                    noisePlaying = false
                }
            }
            AppText(
                text = if (noisePlaying) "Playing" else "Stopped",
                style = AppTheme.typography.label,
                color = AppTheme.colors.muted
            )
        }
    }
}

private fun startNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
    ContextCompat.startForegroundService(context, intent)
}

private fun stopNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
    context.stopService(intent)
}
