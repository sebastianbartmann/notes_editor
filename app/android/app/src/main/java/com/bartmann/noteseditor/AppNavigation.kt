package com.bartmann.noteseditor

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.CalendarToday
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.NightsStay
import androidx.compose.material.icons.automirrored.filled.Notes
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController

sealed class Screen(val route: String, val label: String) {
    data object Daily : Screen("daily", "Daily")
    data object Petra : Screen("petra", "Petra")
    data object Files : Screen("files", "Files")
    data object Sleep : Screen("sleep", "Sleep")
    data object Tools : Screen("tools", "Tools")
}

@Composable
fun NotesEditorApp() {
    val navController = rememberNavController()
    val items = listOf(
        Screen.Daily,
        Screen.Petra,
        Screen.Files,
        Screen.Sleep,
        Screen.Tools
    )
    val icons = mapOf(
        Screen.Daily to Icons.Default.CalendarToday,
        Screen.Petra to Icons.AutoMirrored.Filled.Notes,
        Screen.Files to Icons.Default.Folder,
        Screen.Sleep to Icons.Default.NightsStay,
        Screen.Tools to Icons.Default.Build
    )

    Scaffold(
        bottomBar = {
            NavigationBar(
                containerColor = MaterialTheme.colorScheme.surface
            ) {
                val navBackStackEntry by navController.currentBackStackEntryAsState()
                val currentDestination = navBackStackEntry?.destination
                items.forEach { screen ->
                    NavigationBarItem(
                        icon = { androidx.compose.material3.Icon(icons.getValue(screen), screen.label) },
                        label = { Text(screen.label) },
                        selected = currentDestination?.hierarchy?.any { it.route == screen.route } == true,
                        onClick = {
                            navController.navigate(screen.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    )
                }
            }
        }
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(appBackgroundBrush())
        ) {
            NavHost(
                navController = navController,
                startDestination = Screen.Daily.route,
                modifier = Modifier
            ) {
                composable(Screen.Daily.route) { DailyScreen(Modifier, innerPadding) }
                composable(Screen.Petra.route) { PetraScreen(Modifier, innerPadding) }
                composable(Screen.Files.route) { FilesScreen(Modifier, innerPadding) }
                composable(Screen.Sleep.route) { SleepTimesScreen(Modifier, innerPadding) }
                composable(Screen.Tools.route) { ToolsScreen(Modifier, innerPadding) }
            }
        }
    }
}
