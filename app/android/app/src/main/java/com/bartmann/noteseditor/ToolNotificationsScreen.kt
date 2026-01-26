package com.bartmann.noteseditor

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat

@Composable
fun ToolNotificationsScreen(modifier: Modifier) {
    val context = LocalContext.current

    ScreenLayout(modifier = modifier) {
        ScreenHeader(title = "Notifications")

        Panel {
            SectionTitle(text = "Test")
            CompactButton(text = "Send test notification") {
                sendTestNotification(context)
            }
        }
    }
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
