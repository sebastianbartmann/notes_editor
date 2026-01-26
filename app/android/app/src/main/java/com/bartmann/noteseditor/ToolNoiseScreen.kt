package com.bartmann.noteseditor

import android.content.Context
import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.ContextCompat

@Composable
fun ToolNoiseScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    val context = LocalContext.current
    val noisePlaying by NoisePlaybackState::isPlaying

    ScreenLayout(
        modifier = modifier,
        padding = padding,
        scrollable = false
    ) {
        Panel(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
        ) {
            Column(
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1.2f)
                        .background(AppTheme.colors.note, shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp))
                        .border(1.dp, AppTheme.colors.panelBorder, shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp)),
                    contentAlignment = Alignment.Center
                ) {
                    AppText(
                        text = if (noisePlaying) "Playing" else "Stopped",
                        style = AppTheme.typography.title.copy(
                            fontSize = 24.sp,
                            lineHeight = 28.sp,
                            fontWeight = FontWeight.SemiBold
                        ),
                        color = AppTheme.colors.text
                    )
                }
                LargeActionButton(
                    text = if (noisePlaying) "Pause" else "Play",
                    background = AppTheme.colors.accentDim,
                    border = AppTheme.colors.accent,
                    onClick = { toggleNoise(context) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                )
                LargeActionButton(
                    text = "Stop",
                    background = AppTheme.colors.danger,
                    border = AppTheme.colors.danger,
                    onClick = { stopNoise(context) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                )
            }
        }
    }
}

private fun stopNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
        .setAction(NoiseService.ACTION_STOP)
    context.startService(intent)
}

private fun toggleNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
        .setAction(NoiseService.ACTION_TOGGLE)
    ContextCompat.startForegroundService(context, intent)
}

@Composable
private fun LargeActionButton(
    text: String,
    background: androidx.compose.ui.graphics.Color,
    border: androidx.compose.ui.graphics.Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .background(background, shape = androidx.compose.foundation.shape.RoundedCornerShape(10.dp))
            .border(1.dp, border, shape = androidx.compose.foundation.shape.RoundedCornerShape(10.dp))
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center
    ) {
        AppText(
            text = text,
            style = AppTheme.typography.title.copy(
                fontSize = 22.sp,
                lineHeight = 26.sp,
                fontWeight = FontWeight.SemiBold
            ),
            color = AppTheme.colors.text
        )
    }
}
