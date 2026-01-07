package com.bartmann.noteseditor

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import kotlinx.coroutines.launch

@Composable
fun ToolsScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    val context = LocalContext.current
    var prompt by remember { mutableStateOf("") }
    var responseText by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    var noisePlaying by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Column(
        modifier = modifier
            .padding(padding)
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        ScreenTitle(text = "Tools")
        Panel {
            SectionTitle(text = "Claude")
            CompactOutlinedTextField(
                value = prompt,
                onValueChange = { prompt = it },
                label = "Prompt",
                modifier = Modifier.fillMaxWidth(),
                minLines = 3
            )
            CompactButton(text = "Run") {
                scope.launch {
                    try {
                        val response = ApiClient.runClaude(prompt)
                        responseText = response.response
                        message = response.message
                    } catch (exc: Exception) {
                        message = "Claude failed: ${exc.message}"
                    }
                }
            }
            if (responseText.isNotBlank()) {
                Text(text = responseText)
            }
            CompactDivider()
            SectionTitle(text = "Noise")
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Play") {
                    startNoise(context)
                    noisePlaying = true
                    message = "Noise started."
                }
                CompactButton(text = "Stop") {
                    stopNoise(context)
                    noisePlaying = false
                    message = "Noise stopped."
                }
            }
            Text(text = if (noisePlaying) "Playing" else "Stopped")
            CompactDivider()
            SectionTitle(text = "Notifications")
            CompactButton(text = "Send test notification") {
                sendTestNotification(context)
                message = "Notification sent."
            }
            if (message.isNotBlank()) {
                CompactDivider()
                Text(text = message, color = MaterialTheme.colorScheme.secondary)
            }
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

private fun sendTestNotification(context: Context) {
    val channelId = "notes_general"
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
        val channel = NotificationChannel(
            channelId,
            "Notes alerts",
            NotificationManager.IMPORTANCE_DEFAULT
        )
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(channel)
    }

    val notification = NotificationCompat.Builder(context, channelId)
        .setSmallIcon(R.drawable.ic_launcher)
        .setContentTitle("Notes Editor")
        .setContentText("Notification test")
        .setAutoCancel(true)
        .build()

    NotificationManagerCompat.from(context).notify(3001, notification)
}
