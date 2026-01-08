package com.bartmann.noteseditor

import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

data class AppColors(
    val background: Color,
    val panel: Color,
    val panelBorder: Color,
    val text: Color,
    val muted: Color,
    val accent: Color,
    val accentDim: Color,
    val danger: Color,
    val input: Color,
    val note: Color,
    val button: Color,
    val buttonText: Color,
    val checkboxFill: Color,
)

data class AppSpacing(
    val xs: Dp = 6.dp,
    val sm: Dp = 10.dp,
    val md: Dp = 14.dp,
    val lg: Dp = 18.dp,
    val xl: Dp = 24.dp,
)

data class AppTypography(
    val body: TextStyle,
    val bodySmall: TextStyle,
    val title: TextStyle,
    val section: TextStyle,
    val label: TextStyle,
)

private val LocalAppColors = staticCompositionLocalOf {
    AppColors(
        background = Color(0xFF0F1012),
        panel = Color(0xFF15171A),
        panelBorder = Color(0xFF2A2D33),
        text = Color(0xFFE6E6E6),
        muted = Color(0xFF9AA0A6),
        accent = Color(0xFFD9832B),
        accentDim = Color(0xFF7A4A1D),
        danger = Color(0xFFD66B6B),
        input = Color(0xFF0F1114),
        note = Color(0xFF101317),
        button = Color(0xFF1E2227),
        buttonText = Color(0xFFE6E6E6),
        checkboxFill = Color(0xFFE6E6E6),
    )
}

private val LocalAppSpacing = staticCompositionLocalOf { AppSpacing() }

private val LocalAppTypography = staticCompositionLocalOf {
    val appFont = FontFamily(
        Font(R.font.jetbrains_mono_nerd_regular, weight = FontWeight.Normal),
        Font(R.font.jetbrains_mono_nerd_medium, weight = FontWeight.Medium),
        Font(R.font.jetbrains_mono_nerd_bold, weight = FontWeight.Bold),
    )
    AppTypography(
        body = TextStyle(fontSize = 12.sp, lineHeight = 17.sp, fontFamily = appFont),
        bodySmall = TextStyle(fontSize = 11.sp, lineHeight = 15.sp, fontFamily = appFont),
        title = TextStyle(fontSize = 16.sp, lineHeight = 20.sp, fontFamily = appFont, fontWeight = FontWeight.Medium),
        section = TextStyle(fontSize = 12.sp, lineHeight = 16.sp, fontFamily = appFont, letterSpacing = 0.4.sp),
        label = TextStyle(fontSize = 11.sp, lineHeight = 14.sp, fontFamily = appFont),
    )
}

object AppTheme {
    val colors: AppColors
        @Composable get() = LocalAppColors.current
    val spacing: AppSpacing
        @Composable get() = LocalAppSpacing.current
    val typography: AppTypography
        @Composable get() = LocalAppTypography.current
}

@Composable
fun NotesEditorTheme(content: @Composable () -> Unit) {
    val colors = if (UserSettings.theme == "light") {
        AppColors(
            background = Color(0xFFE9F7F7),
            panel = Color(0xFFF6FBFF),
            panelBorder = Color(0xFFC7E3E6),
            text = Color(0xFF1A2A2F),
            muted = Color(0xFF4F6F78),
            accent = Color(0xFF3AA7A3),
            accentDim = Color(0xFFC9F1EF),
            danger = Color(0xFFD76A6A),
            input = Color(0xFFF2FAFB),
            note = Color(0xFFF9FDFF),
            button = Color(0xFFEEF6F8),
            buttonText = Color(0xFF1A2A2F),
            checkboxFill = Color(0xFFF2FAFB),
        )
    } else {
        AppColors(
            background = Color(0xFF0F1012),
            panel = Color(0xFF15171A),
            panelBorder = Color(0xFF2A2D33),
            text = Color(0xFFE6E6E6),
            muted = Color(0xFF9AA0A6),
            accent = Color(0xFFD9832B),
            accentDim = Color(0xFF7A4A1D),
            danger = Color(0xFFD66B6B),
            input = Color(0xFF0F1114),
            note = Color(0xFF101317),
            button = Color(0xFF20252B),
            buttonText = Color(0xFFE6E6E6),
            checkboxFill = Color(0xFFE6E6E6),
        )
    }
    val spacing = AppSpacing()
    val typography = LocalAppTypography.current
    CompositionLocalProvider(
        LocalAppColors provides colors,
        LocalAppSpacing provides spacing,
        LocalAppTypography provides typography,
    ) {
        content()
    }
}
