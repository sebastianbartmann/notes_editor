package com.bartmann.noteseditor

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.automirrored.filled.OpenInNew
import androidx.compose.material.icons.automirrored.filled.VolumeUp
import androidx.compose.material.icons.filled.CalendarToday
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.NightsStay
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Settings
import androidx.compose.ui.graphics.vector.ImageVector

data class NavEntry(
    val id: String,
    val label: String,
    val icon: ImageVector,
    val route: String?,
    val externalUrl: String? = null
)

val selectableNavEntries = listOf(
    NavEntry("daily", "Daily", Icons.Default.CalendarToday, Screen.Daily.route),
    NavEntry("files", "Files", Icons.Default.Folder, Screen.Files.route),
    NavEntry("sleep", "Sleep", Icons.Default.NightsStay, Screen.Sleep.route),
    NavEntry("claude", "Claude", Icons.AutoMirrored.Filled.Chat, Screen.ToolClaude.route),
    NavEntry("noise", "Noise", Icons.AutoMirrored.Filled.VolumeUp, Screen.ToolNoise.route),
    NavEntry("notifications", "Notifications", Icons.Default.Notifications, Screen.ToolNotifications.route),
    NavEntry("settings", "Settings", Icons.Default.Settings, Screen.Settings.route),
    NavEntry(
        "kiosk",
        "Kiosk",
        Icons.AutoMirrored.Filled.OpenInNew,
        route = null,
        externalUrl = "https://thirdpartycheck.com/admin/kiosk"
    )
)

val fixedMoreEntry = NavEntry("more", "More", Icons.Default.MoreHoriz, Screen.Tools.route)

const val maxBottomNavItems = 4

val defaultBottomNavIds = listOf("daily", "files", "sleep")

fun sanitizeBottomNavIds(ids: List<String>): List<String> {
    val allowed = selectableNavEntries.map { it.id }.toSet()
    return ids.filter { it in allowed }.distinct().take(maxBottomNavItems)
}

fun sanitizeStoredBottomNavIds(ids: List<String>): List<String> {
    val normalized = sanitizeBottomNavIds(ids)
    return if (normalized.isEmpty()) defaultBottomNavIds else normalized
}

fun bottomNavEntriesFor(ids: List<String>): List<NavEntry> {
    val lookup = selectableNavEntries.associateBy { it.id }
    return ids.mapNotNull { lookup[it] }
}
