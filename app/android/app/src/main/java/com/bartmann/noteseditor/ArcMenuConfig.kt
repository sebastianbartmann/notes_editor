package com.bartmann.noteseditor

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
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

/**
 * Represents a single menu item in the arc menu.
 *
 * @param id Unique identifier for the item
 * @param icon Material icon to display
 * @param label Text label shown below icon
 * @param route Navigation route (null for submenu triggers like "More")
 * @param externalUrl URL to open in browser (for items like Kiosk)
 * @param children Nested items for submenu (only "More" uses this)
 */
data class ArcMenuItem(
    val id: String,
    val icon: ImageVector,
    val label: String,
    val route: String?,
    val externalUrl: String? = null,
    val children: List<ArcMenuItem>? = null
)

enum class ArcMenuState {
    COLLAPSED,
    LEVEL1,
    LEVEL2
}

val arcMenuItems = listOf(
    ArcMenuItem(
        id = "daily",
        icon = Icons.Default.CalendarToday,
        label = "Daily",
        route = "daily"
    ),
    ArcMenuItem(
        id = "files",
        icon = Icons.Default.Folder,
        label = "Files",
        route = "files"
    ),
    ArcMenuItem(
        id = "claude",
        icon = Icons.AutoMirrored.Filled.Chat,
        label = "Claude",
        route = "tool-claude"
    ),
    ArcMenuItem(
        id = "more",
        icon = Icons.Default.MoreHoriz,
        label = "More",
        route = null,
        children = listOf(
            ArcMenuItem(
                id = "sleep",
                icon = Icons.Default.NightsStay,
                label = "Sleep",
                route = "sleep"
            ),
            ArcMenuItem(
                id = "noise",
                icon = Icons.AutoMirrored.Filled.VolumeUp,
                label = "Noise",
                route = "tool-noise"
            ),
            ArcMenuItem(
                id = "notifications",
                icon = Icons.Default.Notifications,
                label = "Notifs",
                route = "tool-notifications"
            ),
            ArcMenuItem(
                id = "settings",
                icon = Icons.Default.Settings,
                label = "Settings",
                route = "settings"
            ),
            ArcMenuItem(
                id = "kiosk",
                icon = Icons.AutoMirrored.Filled.OpenInNew,
                label = "Kiosk",
                route = null,
                externalUrl = "https://thirdpartycheck.com/admin/kiosk"
            ),
            ArcMenuItem(
                id = "back",
                icon = Icons.AutoMirrored.Filled.ArrowBack,
                label = "Back",
                route = null
            )
        )
    )
)
