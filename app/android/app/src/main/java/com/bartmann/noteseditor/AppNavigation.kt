package com.bartmann.noteseditor

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.ime
import androidx.compose.ui.Alignment
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
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

    companion object {
        fun titleForRoute(route: String?): String = when (route) {
            Daily.route -> Daily.label
            Files.route -> Files.label
            Sleep.route -> Sleep.label
            Settings.route -> Settings.label
            ToolClaude.route -> ToolClaude.label
            ToolNoise.route -> ToolNoise.label
            ToolNotifications.route -> ToolNotifications.label
            else -> ""
        }
    }
}

@Composable
fun BottomInfoBar(
    currentRoute: String?,
    menuState: ArcMenuState,
    onMenuButtonClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(56.dp)
            .padding(start = 16.dp, end = 80.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        AppText(
            text = Screen.titleForRoute(currentRoute),
            style = AppTheme.typography.title,
            color = AppTheme.colors.text,
            modifier = Modifier.weight(1f)
        )
        ArcMenuButton(
            isExpanded = menuState != ArcMenuState.COLLAPSED,
            onClick = onMenuButtonClick
        )
    }
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
        // Bottom padding for content when BottomInfoBar is visible
        val bottomBarPadding = if (person != null && !isKeyboardVisible) {
            androidx.compose.foundation.layout.PaddingValues(bottom = 56.dp)
        } else {
            androidx.compose.foundation.layout.PaddingValues()
        }

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
                    SettingsScreen(Modifier, bottomBarPadding)
                }
                composable(Screen.Daily.route) {
                    DailyScreen(Modifier, bottomBarPadding)
                }
                composable(Screen.Files.route) { FilesScreen(Modifier, bottomBarPadding) }
                composable(Screen.Sleep.route) { SleepTimesScreen(Modifier, bottomBarPadding) }
                composable(Screen.ToolClaude.route) { ToolClaudeScreen(Modifier, bottomBarPadding) }
                composable(Screen.ToolNoise.route) { ToolNoiseScreen(Modifier, bottomBarPadding) }
                composable(Screen.ToolNotifications.route) { ToolNotificationsScreen(Modifier, bottomBarPadding) }
            }

            // Bottom info bar with title and menu button
            if (person != null && !isKeyboardVisible) {
                BottomInfoBar(
                    currentRoute = currentRoute,
                    menuState = menuState,
                    onMenuButtonClick = {
                        menuState = if (menuState == ArcMenuState.COLLAPSED)
                            ArcMenuState.LEVEL1 else ArcMenuState.COLLAPSED
                    },
                    modifier = Modifier.align(Alignment.BottomCenter)
                )
            }

            // Arc menu overlay (scrim + expanded items only, no button)
            if (person != null && !isKeyboardVisible && menuState != ArcMenuState.COLLAPSED) {
                ArcMenu(
                    items = arcMenuItems,
                    currentRoute = currentRoute,
                    menuState = menuState,
                    onStateChange = { menuState = it },
                    onNavigate = { route -> navigateByRoute(route) },
                    onOpenExternal = { url -> uriHandler.openUri(url) },
                    showButton = false,
                    modifier = Modifier.fillMaxSize()
                )
            }
        }
        KeyboardAccessoryBar()
    }
}

