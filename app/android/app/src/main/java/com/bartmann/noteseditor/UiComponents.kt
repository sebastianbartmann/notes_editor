package com.bartmann.noteseditor

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicText
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp

@Composable
fun AppText(text: String, style: TextStyle, color: androidx.compose.ui.graphics.Color, modifier: Modifier = Modifier) {
    BasicText(text = text, style = style.copy(color = color), modifier = modifier)
}

@Composable
fun ScreenTitle(text: String) {
    AppText(text = text, style = AppTheme.typography.title, color = AppTheme.colors.text)
}

@Composable
fun ScreenHeader(
    title: String,
    actionButton: @Composable (() -> Unit)? = null
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(32.dp)
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        AppText(
            text = title,
            style = AppTheme.typography.title,
            color = AppTheme.colors.text
        )
        if (actionButton != null) {
            actionButton()
        }
    }
}

@Composable
fun SectionTitle(text: String) {
    AppText(text = text.uppercase(), style = AppTheme.typography.section, color = AppTheme.colors.muted)
}

@Composable
fun ScreenLayout(
    modifier: Modifier = Modifier,
    scrollable: Boolean = true,
    content: @Composable ColumnScope.() -> Unit
) {
    val baseModifier = modifier
        .fillMaxSize()
        .padding(AppTheme.spacing.sm)
    val layoutModifier = if (scrollable) {
        baseModifier.verticalScroll(rememberScrollState())
    } else {
        baseModifier
    }
    Column(
        modifier = layoutModifier,
        verticalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm),
        content = content
    )
}

@Composable
fun CompactDivider() {
    Column {
        Spacer(modifier = Modifier.height(AppTheme.spacing.sm))
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .background(AppTheme.colors.panelBorder)
                .height(1.dp)
        )
        Spacer(modifier = Modifier.height(AppTheme.spacing.sm))
    }
}

@Composable
fun CompactButton(
    text: String,
    modifier: Modifier = Modifier,
    background: Color = AppTheme.colors.button,
    border: Color = AppTheme.colors.panelBorder,
    textColor: Color = AppTheme.colors.buttonText,
    onClick: () -> Unit
) {
    val shape = RoundedCornerShape(6.dp)
    Box(
        modifier = modifier
            .border(1.dp, border, shape)
            .background(background, shape)
            .clickable(onClick = onClick)
            .padding(horizontal = AppTheme.spacing.sm, vertical = 3.dp)
        , contentAlignment = androidx.compose.ui.Alignment.Center
    ) {
        AppText(text = text, style = AppTheme.typography.body, color = textColor)
    }
}

@Composable
fun CompactTextButton(text: String, onClick: () -> Unit) {
    CompactButton(text = text, onClick = onClick)
}

@Composable
fun CompactTextField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier,
    minLines: Int = 1,
    readOnly: Boolean = false,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    keyboardActions: KeyboardActions = KeyboardActions.Default
) {
    var isFocused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(6.dp)
    val borderColor = if (isFocused) AppTheme.colors.accent else AppTheme.colors.panelBorder

    BasicTextField(
        value = value,
        onValueChange = onValueChange,
        textStyle = AppTheme.typography.body.copy(color = AppTheme.colors.text),
        cursorBrush = SolidColor(AppTheme.colors.accent),
        readOnly = readOnly,
        keyboardOptions = keyboardOptions,
        keyboardActions = keyboardActions,
        modifier = modifier
            .border(1.dp, borderColor, shape)
            .background(AppTheme.colors.input, shape)
            .padding(AppTheme.spacing.sm)
            .onFocusChanged { isFocused = it.isFocused },
        minLines = minLines,
        decorationBox = { innerTextField ->
            Box {
                if (value.isBlank()) {
                    AppText(
                        text = placeholder,
                        style = AppTheme.typography.bodySmall,
                        color = AppTheme.colors.muted
                    )
                }
                innerTextField()
            }
        }
    )
}

@Composable
fun AppCheckbox(
    checked: Boolean,
    modifier: Modifier = Modifier,
    size: Int = 14
) {
    val shape = RoundedCornerShape(3.dp)
    val fill = if (checked) AppTheme.colors.accent else AppTheme.colors.checkboxFill
    val border = if (checked) AppTheme.colors.accent else AppTheme.colors.panelBorder
    Box(
        modifier = modifier
            .size(size.dp)
            .border(1.dp, border, shape)
            .background(fill, shape),
        contentAlignment = androidx.compose.ui.Alignment.Center
    ) {
    }
}

@Composable
fun Panel(
    modifier: Modifier = Modifier,
    fill: Boolean = true,
    content: @Composable ColumnScope.() -> Unit
) {
    val shape = RoundedCornerShape(6.dp)
    Box(
        modifier = modifier
            .shadow(6.dp, shape)
            .background(AppTheme.colors.panel, shape)
            .border(1.dp, AppTheme.colors.panelBorder, shape)
            .padding(AppTheme.spacing.sm)
    ) {
        val columnModifier = if (fill) {
            Modifier.fillMaxSize()
        } else {
            Modifier.fillMaxWidth()
        }
        Column(
            modifier = columnModifier,
            verticalArrangement = Arrangement.spacedBy(AppTheme.spacing.sm)
        ) {
            content()
        }
    }
}

@Composable
fun appBackgroundBrush(): Brush =
    Brush.verticalGradient(
        colors = listOf(AppTheme.colors.background, AppTheme.colors.background)
    )

@Composable
fun MessageBadge(text: String) {
    val shape = RoundedCornerShape(6.dp)
    Box(
        modifier = Modifier
            .background(AppTheme.colors.panel, shape)
            .border(1.dp, AppTheme.colors.panelBorder, shape)
            .padding(horizontal = AppTheme.spacing.sm, vertical = 4.dp)
    ) {
        AppText(text = text, style = AppTheme.typography.label, color = AppTheme.colors.text)
    }
}

@Composable
fun StatusMessage(text: String, showDivider: Boolean = true) {
    if (text.isBlank()) return
    if (showDivider) {
        CompactDivider()
    }
    MessageBadge(text = text)
}

@Composable
fun NoteSurface(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    val shape = RoundedCornerShape(6.dp)
    Box(
        modifier = modifier
            .background(AppTheme.colors.note, shape)
            .border(1.dp, AppTheme.colors.panelBorder, shape)
            .padding(AppTheme.spacing.xs)
    ) {
        content()
    }
}
