package com.bartmann.noteseditor

import android.view.KeyCharacterMap
import android.view.KeyEvent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.unit.dp

@Composable
fun KeyboardAccessoryBar(
    modifier: Modifier = Modifier
) {
    val view = LocalView.current
    val density = LocalDensity.current
    val imeBottom = WindowInsets.ime.getBottom(density)
    val isKeyboardVisible = imeBottom > 0

    AnimatedVisibility(
        visible = isKeyboardVisible,
        enter = slideInVertically { it },
        exit = slideOutVertically { it }
    ) {
        Row(
        modifier = modifier
            .fillMaxWidth()
            .background(AppTheme.colors.panel)
            .border(1.dp, AppTheme.colors.panelBorder)
            .padding(horizontal = AppTheme.spacing.sm, vertical = AppTheme.spacing.xs),
        horizontalArrangement = Arrangement.SpaceEvenly
    ) {
        AccessoryButton("↑") { sendKeyEvent(view, KeyEvent.KEYCODE_DPAD_UP) }
        AccessoryButton("↓") { sendKeyEvent(view, KeyEvent.KEYCODE_DPAD_DOWN) }
        AccessoryButton("←") { sendKeyEvent(view, KeyEvent.KEYCODE_DPAD_LEFT) }
        AccessoryButton("→") { sendKeyEvent(view, KeyEvent.KEYCODE_DPAD_RIGHT) }
        AccessoryButton("/") { commitText(view, "/") }
        AccessoryButton("[") { commitText(view, "[") }
        AccessoryButton("]") { commitText(view, "]") }
        }
    }
}

@Composable
private fun AccessoryButton(
    text: String,
    onClick: () -> Unit
) {
    val shape = RoundedCornerShape(6.dp)
    Box(
        modifier = Modifier
            .border(1.dp, AppTheme.colors.panelBorder, shape)
            .background(AppTheme.colors.button, shape)
            .clickable(onClick = onClick)
            .padding(horizontal = AppTheme.spacing.md, vertical = AppTheme.spacing.xs),
        contentAlignment = Alignment.Center
    ) {
        AppText(text = text, style = AppTheme.typography.body, color = AppTheme.colors.buttonText)
    }
}

private fun sendKeyEvent(view: android.view.View, keyCode: Int) {
    val eventTime = android.os.SystemClock.uptimeMillis()
    val downEvent = KeyEvent(eventTime, eventTime, KeyEvent.ACTION_DOWN, keyCode, 0)
    val upEvent = KeyEvent(eventTime, eventTime, KeyEvent.ACTION_UP, keyCode, 0)
    view.dispatchKeyEvent(downEvent)
    view.dispatchKeyEvent(upEvent)
}

private fun commitText(view: android.view.View, text: String) {
    val charMap = KeyCharacterMap.load(KeyCharacterMap.VIRTUAL_KEYBOARD)
    val events = charMap.getEvents(text.toCharArray())
    events?.forEach { event ->
        view.dispatchKeyEvent(event)
    }
}
