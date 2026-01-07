package com.bartmann.noteseditor

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.sp

private val DarkColors = darkColorScheme(
    primary = Color(0xFFD9832B),
    onPrimary = Color(0xFF0F1012),
    secondary = Color(0xFF9AA0A6),
    onSecondary = Color(0xFF0F1012),
    surface = Color(0xFF15171A),
    onSurface = Color(0xFFE6E6E6),
    background = Color(0xFF0F1012),
    onBackground = Color(0xFFE6E6E6)
)

@Composable
fun NotesEditorTheme(content: @Composable () -> Unit) {
    val typography = Typography(
        bodyLarge = TextStyle(fontSize = 13.sp, lineHeight = 19.sp, fontFamily = FontFamily.Monospace),
        bodyMedium = TextStyle(fontSize = 12.sp, lineHeight = 17.sp, fontFamily = FontFamily.Monospace),
        bodySmall = TextStyle(fontSize = 11.sp, lineHeight = 15.sp, fontFamily = FontFamily.Monospace),
        titleLarge = TextStyle(fontSize = 16.sp, lineHeight = 20.sp, fontFamily = FontFamily.Monospace),
        titleMedium = TextStyle(fontSize = 14.sp, lineHeight = 18.sp, fontFamily = FontFamily.Monospace),
        titleSmall = TextStyle(fontSize = 13.sp, lineHeight = 16.sp, fontFamily = FontFamily.Monospace),
        labelLarge = TextStyle(fontSize = 11.sp, fontFamily = FontFamily.Monospace),
        labelMedium = TextStyle(fontSize = 10.sp, fontFamily = FontFamily.Monospace),
        labelSmall = TextStyle(fontSize = 9.sp, fontFamily = FontFamily.Monospace)
    )
    MaterialTheme(
        colorScheme = DarkColors,
        typography = typography,
        content = content
    )
}
