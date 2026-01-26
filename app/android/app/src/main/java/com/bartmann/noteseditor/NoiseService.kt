package com.bartmann.noteseditor

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.media.MediaPlayer
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.media.app.NotificationCompat.MediaStyle
import androidx.media.session.MediaButtonReceiver
import android.support.v4.media.session.MediaSessionCompat
import android.support.v4.media.session.PlaybackStateCompat

class NoiseService : Service() {
    private var currentPlayer: MediaPlayer? = null
    private var nextPlayer: MediaPlayer? = null
    private var mediaSession: MediaSessionCompat? = null
    private var isPlaying = false

    override fun onCreate() {
        super.onCreate()
        createChannel()
        mediaSession = MediaSessionCompat(this, "NoiseService").apply {
            setCallback(object : MediaSessionCompat.Callback() {
                override fun onPlay() {
                    handlePlay()
                }

                override fun onPause() {
                    handlePause()
                }

                override fun onStop() {
                    handleStop()
                }
            })
            isActive = true
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_PLAY -> handlePlay()
            ACTION_PAUSE -> handlePause()
            ACTION_STOP -> handleStop()
            ACTION_TOGGLE -> if (isPlaying) handlePause() else handlePlay()
            else -> handlePlay()
        }
        return START_STICKY
    }

    override fun onDestroy() {
        stopPlayers()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun buildNotification(): Notification {
        val contentIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val toggleIntent = PendingIntent.getService(
            this,
            1,
            Intent(this, NoiseService::class.java).setAction(ACTION_TOGGLE),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val stopIntent = PendingIntent.getService(
            this,
            2,
            Intent(this, NoiseService::class.java).setAction(ACTION_STOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val deleteIntent = PendingIntent.getService(
            this,
            3,
            Intent(this, NoiseService::class.java).setAction(ACTION_STOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val playPauseIcon = if (isPlaying) {
            android.R.drawable.ic_media_pause
        } else {
            android.R.drawable.ic_media_play
        }
        val playPauseLabel = if (isPlaying) "Pause" else "Play"

        val state = if (isPlaying) PlaybackStateCompat.STATE_PLAYING else PlaybackStateCompat.STATE_PAUSED
        mediaSession?.setPlaybackState(
            PlaybackStateCompat.Builder()
                .setActions(
                    PlaybackStateCompat.ACTION_PLAY
                        or PlaybackStateCompat.ACTION_PAUSE
                        or PlaybackStateCompat.ACTION_PLAY_PAUSE
                        or PlaybackStateCompat.ACTION_STOP
                )
                .setState(state, PlaybackStateCompat.PLAYBACK_POSITION_UNKNOWN, 1.0f)
                .build()
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("White Noise")
            .setContentText(if (isPlaying) "Playing" else "Paused")
            .setSmallIcon(R.drawable.ic_launcher)
            .setContentIntent(contentIntent)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setDeleteIntent(deleteIntent)
            .addAction(NotificationCompat.Action(playPauseIcon, playPauseLabel, toggleIntent))
            .addAction(NotificationCompat.Action(android.R.drawable.ic_menu_close_clear_cancel, "Stop", stopIntent))
            .setStyle(
                MediaStyle()
                    .setMediaSession(mediaSession?.sessionToken)
                    .setShowActionsInCompactView(0, 1)
            )
            .build()
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Noise Playback",
            NotificationManager.IMPORTANCE_LOW
        )
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(channel)
    }

    private fun ensurePlayer() {
        if (currentPlayer == null) {
            currentPlayer = createPlayer().also { attachCompletionListener(it) }
        }
        if (nextPlayer == null) {
            nextPlayer = createPlayer()
        }
        currentPlayer?.setNextMediaPlayer(nextPlayer)
    }

    private fun createPlayer(): MediaPlayer = MediaPlayer.create(this, R.raw.noise)

    private fun attachCompletionListener(player: MediaPlayer) {
        player.setOnCompletionListener {
            val completed = currentPlayer
            if (completed != player) return@setOnCompletionListener
            val previous = currentPlayer
            currentPlayer = nextPlayer
            currentPlayer?.let { attachCompletionListener(it) }
            nextPlayer = createPlayer()
            currentPlayer?.setNextMediaPlayer(nextPlayer)
            previous?.release()
        }
    }

    private fun stopPlayers() {
        currentPlayer?.release()
        currentPlayer = null
        nextPlayer?.release()
        nextPlayer = null
    }

    private fun handlePlay() {
        ensurePlayer()
        currentPlayer?.start()
        isPlaying = true
        NoisePlaybackState.isPlaying = true
        startForeground(NOTIFICATION_ID, buildNotification())
    }

    private fun handlePause() {
        currentPlayer?.pause()
        isPlaying = false
        NoisePlaybackState.isPlaying = false
        startForeground(NOTIFICATION_ID, buildNotification())
    }

    private fun handleStop() {
        isPlaying = false
        NoisePlaybackState.isPlaying = false
        stopPlayers()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    companion object {
        const val CHANNEL_ID = "noise_playback"
        const val NOTIFICATION_ID = 2001
        const val ACTION_TOGGLE = "com.bartmann.noteseditor.noise.TOGGLE"
        const val ACTION_PLAY = "com.bartmann.noteseditor.noise.PLAY"
        const val ACTION_PAUSE = "com.bartmann.noteseditor.noise.PAUSE"
        const val ACTION_STOP = "com.bartmann.noteseditor.noise.STOP"
    }
}
