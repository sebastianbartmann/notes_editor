package com.bartmann.noteseditor

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Notes
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.CalendarToday
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.NightsStay
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.unit.dp
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController

sealed class Screen(val route: String, val label: String) {
    data object Daily : Screen("daily", "Daily")
    data object Files : Screen("files", "Files")
    data object Sleep : Screen("sleep", "Sleep")
    data object Tools : Screen("tools", "Tools")
    data object Settings : Screen("settings", "Settings")
    data object ToolClaude : Screen("tool-claude", "Claude")
    data object ToolNoise : Screen("tool-noise", "Noise")
    data object ToolNotifications : Screen("tool-notifications", "Notifications")
}

@Composable
fun NotesEditorApp() {
    val person = UserSettings.person
    val navController = rememberNavController()
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentDestination = navBackStackEntry?.destination
    val currentRoute = currentDestination?.route
    val items = listOf(
        Screen.Daily,
        Screen.Files,
        Screen.Sleep,
        Screen.Tools
    )
    val icons = mapOf(
        Screen.Daily to Icons.Default.CalendarToday,
        Screen.Files to Icons.Default.Folder,
        Screen.Sleep to Icons.Default.NightsStay,
        Screen.Tools to Icons.Default.Build
    )

    LaunchedEffect(person) {
        if (person != null && currentRoute == Screen.Settings.route) {
            navController.navigate(Screen.Daily.route) {
                popUpTo(navController.graph.findStartDestination().id) { inclusive = true }
                launchSingleTop = true
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(appBackgroundBrush())
    ) {
        Box(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .padding(top = 32.dp)
        ) {
            NavHost(
                navController = navController,
                startDestination = if (person == null) Screen.Settings.route else Screen.Daily.route
            ) {
                composable(Screen.Settings.route) {
                    SettingsScreen(Modifier, androidx.compose.foundation.layout.PaddingValues())
                }
                composable(Screen.Daily.route) {
                    DailyScreen(
                        Modifier,
                        androidx.compose.foundation.layout.PaddingValues()
                    )
                }
                composable(Screen.Files.route) { FilesScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.Sleep.route) { SleepTimesScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.Tools.route) {
                    ToolsScreen(
                        Modifier,
                        androidx.compose.foundation.layout.PaddingValues(),
                        onOpenClaude = { navController.navigate(Screen.ToolClaude.route) },
                        onOpenNoise = { navController.navigate(Screen.ToolNoise.route) },
                        onOpenNotifications = { navController.navigate(Screen.ToolNotifications.route) },
                        onOpenSettings = { navController.navigate(Screen.Settings.route) }
                    )
                }
                composable(Screen.ToolClaude.route) { ToolClaudeScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.ToolNoise.route) { ToolNoiseScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.ToolNotifications.route) { ToolNotificationsScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
            }
        }
        if (person != null) {
            Box(modifier = Modifier.padding(bottom = 8.dp)) {
                BottomNavBar(
                    items = items,
                    icons = icons,
                    currentRoute = currentDestination,
                    onNavigate = { screen ->
                        if (currentRoute == screen.route) {
                            return@BottomNavBar
                        }
                        val popped = navController.popBackStack(screen.route, inclusive = false)
                        if (!popped) {
                            navController.navigate(screen.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    }
                )
            }
        }
    }
}

@Composable
private fun BottomNavBar(
    items: List<Screen>,
    icons: Map<Screen, androidx.compose.ui.graphics.vector.ImageVector>,
    currentRoute: androidx.navigation.NavDestination?,
    onNavigate: (Screen) -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(AppTheme.colors.panel)
            .border(1.dp, AppTheme.colors.panelBorder)
            .padding(vertical = 6.dp),
        horizontalArrangement = Arrangement.SpaceAround,
        verticalAlignment = Alignment.CenterVertically
    ) {
        items.forEach { screen ->
            val selected = currentRoute?.hierarchy?.any { it.route == screen.route } == true
            val color = if (selected) AppTheme.colors.accent else AppTheme.colors.muted
            val bgColor = if (selected) AppTheme.colors.accentDim else AppTheme.colors.panel
            Column(
                modifier = Modifier
                    .background(bgColor, shape = androidx.compose.foundation.shape.RoundedCornerShape(6.dp))
                    .clickable { onNavigate(screen) }
                    .padding(horizontal = 10.dp, vertical = 4.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Image(
                    painter = rememberVectorPainter(icons.getValue(screen)),
                    contentDescription = screen.label,
                    colorFilter = ColorFilter.tint(color),
                    modifier = Modifier.size(18.dp)
                )
                AppText(text = screen.label, style = AppTheme.typography.label, color = color)
            }
        }
    }
}
