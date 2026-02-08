package com.bartmann.noteseditor

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.ime
import androidx.compose.ui.Alignment
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController

sealed class Screen(val route: String, val label: String) {
    data object Daily : Screen("daily", "Daily")
    data object Files : Screen("files", "Files")
    data object Sleep : Screen("sleep", "Sleep")
    data object Tools : Screen("tools", "More")
    data object Settings : Screen("settings", "Settings")
    data object ToolClaude : Screen("tool-claude", "Claude")
    data object ToolNoise : Screen("tool-noise", "Noise")
    data object ToolNotifications : Screen("tool-notifications", "Notifications")
}

@Composable
fun BottomNavBar(
    currentRoute: String?,
    onNavigate: (String) -> Unit,
    onOpenExternal: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val navItems = bottomNavEntriesFor(UserSettings.bottomNavIds) + fixedMoreEntry
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(56.dp)
            .background(AppTheme.colors.panel),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically
    ) {
        navItems.forEach { item ->
            val isActive = currentRoute == item.route
            val iconColor = if (isActive) AppTheme.colors.accent else AppTheme.colors.muted
            val textColor = if (isActive) AppTheme.colors.accent else AppTheme.colors.muted

            Column(
                modifier = Modifier
                    .weight(1f)
                    .clickable {
                        when {
                            item.externalUrl != null -> onOpenExternal(item.externalUrl)
                            item.route != null -> onNavigate(item.route)
                        }
                    },
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center
            ) {
                Image(
                    painter = rememberVectorPainter(item.icon),
                    contentDescription = item.label,
                    colorFilter = ColorFilter.tint(iconColor),
                    modifier = Modifier.size(24.dp)
                )
                AppText(
                    text = item.label,
                    style = AppTheme.typography.label,
                    color = textColor
                )
            }
        }
    }
}

@Composable
fun NotesEditorApp() {
    val person = UserSettings.person
    val density = LocalDensity.current
    val isKeyboardVisible = WindowInsets.ime.getBottom(density) > 0
    val navController = rememberNavController()
    val uriHandler = LocalUriHandler.current
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentDestination = navBackStackEntry?.destination
    val currentRoute = currentDestination?.route

    fun navigateByRoute(route: String) {
        if (currentRoute == route) {
            return
        }
        val popped = navController.popBackStack(route, inclusive = false)
        if (!popped) {
            navController.navigate(route) {
                launchSingleTop = true
            }
        }
    }

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
            .background(appBackgroundColor())
            .imePadding()
    ) {
        Box(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .windowInsetsPadding(WindowInsets.statusBars)
                .windowInsetsPadding(WindowInsets.navigationBars)
        ) {
            NavHost(
                navController = navController,
                startDestination = if (person == null) Screen.Settings.route else Screen.Daily.route
            ) {
                composable(Screen.Settings.route) {
                    SettingsScreen(Modifier)
                }
                composable(Screen.Daily.route) {
                    DailyScreen(Modifier)
                }
                composable(Screen.Files.route) {
                    FilesScreen(Modifier)
                }
                composable(Screen.Sleep.route) {
                    SleepTimesScreen(Modifier)
                }
                composable(Screen.Tools.route) {
                    ToolsScreen(
                        modifier = Modifier,
                        onNavigate = { route -> navigateByRoute(route) }
                    )
                }
                composable(Screen.ToolClaude.route) {
                    ToolClaudeScreen(Modifier)
                }
                composable(Screen.ToolNoise.route) {
                    ToolNoiseScreen(Modifier)
                }
                composable(Screen.ToolNotifications.route) {
                    ToolNotificationsScreen(Modifier)
                }
            }
        }

        // Keyboard accessory bar (only when keyboard visible)
        KeyboardAccessoryBar()

        // Bottom navigation bar (hide when keyboard visible or person not set)
        if (person != null && !isKeyboardVisible) {
            BottomNavBar(
                currentRoute = currentRoute,
                onNavigate = { route -> navigateByRoute(route) },
                onOpenExternal = { url -> uriHandler.openUri(url) },
                modifier = Modifier.windowInsetsPadding(WindowInsets.navigationBars)
            )
        }
    }
}
