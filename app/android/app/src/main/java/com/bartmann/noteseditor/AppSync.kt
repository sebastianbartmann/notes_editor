package com.bartmann.noteseditor

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object AppSync {
    var status by mutableStateOf<SyncStatus?>(null)
        private set
    var offline by mutableStateOf(false)
        private set

    suspend fun refreshStatus() {
        try {
            status = ApiClient.fetchSyncStatus()
            offline = false
        } catch (_: Exception) {
            offline = true
        }
    }

    suspend fun syncIfStale(timeoutMs: Int = 2_000, maxAgeMs: Long = 30_000) {
        // Best-effort: if status is missing/stale, ask the server to sync (it is rate-limited server-side).
        val lastPullAt = status?.lastPullAt
        val stale = if (lastPullAt == null) {
            true
        } else {
            val t = runCatching { java.time.Instant.parse(lastPullAt).toEpochMilli() }.getOrNull()
            t == null || (System.currentTimeMillis() - t) >= maxAgeMs
        }

        if (!stale) {
            return
        }

        try {
            ApiClient.sync(wait = true, timeoutMs = timeoutMs)
        } catch (_: Exception) {
            // ignore
        }
        refreshStatus()
    }
}

