package com.bartmann.noteseditor

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.ime
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController

sealed class Screen(val route: String, val label: String) {
    data object Daily : Screen("daily", "Daily")
    data object Files : Screen("files", "Files")
    data object Sleep : Screen("sleep", "Sleep")
    data object Settings : Screen("settings", "Settings")
    data object ToolClaude : Screen("tool-claude", "Claude")
    data object ToolNoise : Screen("tool-noise", "Noise")
    data object ToolNotifications : Screen("tool-notifications", "Notifications")
}

@Composable
fun NotesEditorApp() {
    val person = UserSettings.person
    val density = LocalDensity.current
    val isKeyboardVisible = WindowInsets.ime.getBottom(density) > 0
    val navController = rememberNavController()
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentDestination = navBackStackEntry?.destination
    val currentRoute = currentDestination?.route
    val uriHandler = LocalUriHandler.current
    var menuState by remember { mutableStateOf(ArcMenuState.COLLAPSED) }

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
            .background(appBackgroundBrush())
            .imePadding()
    ) {
        Box(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .windowInsetsPadding(WindowInsets.statusBars)
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
                composable(Screen.ToolClaude.route) { ToolClaudeScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.ToolNoise.route) { ToolNoiseScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
                composable(Screen.ToolNotifications.route) { ToolNotificationsScreen(Modifier, androidx.compose.foundation.layout.PaddingValues()) }
            }

            // Arc menu overlay - positioned within the Box so it overlays content
            if (person != null && !isKeyboardVisible) {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .windowInsetsPadding(WindowInsets.navigationBars)
                ) {
                    ArcMenu(
                        items = arcMenuItems,
                        currentRoute = currentRoute,
                        menuState = menuState,
                        onStateChange = { menuState = it },
                        onNavigate = { route -> navigateByRoute(route) },
                        onOpenExternal = { url -> uriHandler.openUri(url) }
                    )
                }
            }
        }
        KeyboardAccessoryBar()
    }
}

