package com.bartmann.noteseditor

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

private val PanelColor = Color(0xFF15171A)
private val PanelBorder = Color(0xFF2A2D33)
private val MutedText = Color(0xFF9AA0A6)
private val Accent = Color(0xFFD9832B)
private val InputBg = Color(0xFF0F1114)

@Composable
fun ScreenTitle(text: String) {
    Text(text = text, fontSize = 16.sp, color = MaterialTheme.colorScheme.onBackground)
}

@Composable
fun SectionTitle(text: String) {
    Text(text = text, fontSize = 12.sp, color = MutedText)
}

@Composable
fun CompactDivider() {
    HorizontalDivider(thickness = 1.dp, color = PanelBorder)
}

@Composable
fun CompactButton(text: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        contentPadding = PaddingValues(horizontal = 10.dp, vertical = 4.dp),
        shape = RoundedCornerShape(6.dp),
        border = BorderStroke(1.dp, PanelBorder),
        colors = ButtonDefaults.buttonColors(
            containerColor = Color(0xFF1E2227),
            contentColor = Color(0xFFE6E6E6)
        )
    ) {
        Text(text = text, fontSize = 11.sp)
    }
}

@Composable
fun CompactTextButton(text: String, onClick: () -> Unit) {
    TextButton(
        onClick = onClick,
        contentPadding = PaddingValues(horizontal = 8.dp, vertical = 2.dp),
        colors = ButtonDefaults.textButtonColors(contentColor = MutedText)
    ) {
        Text(text = text, fontSize = 11.sp)
    }
}

@Composable
fun CompactOutlinedTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier,
    minLines: Int = 1
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label, fontSize = 11.sp, color = MutedText) },
        textStyle = TextStyle(fontSize = 12.sp, color = Color(0xFFE6E6E6)),
        modifier = modifier,
        minLines = minLines,
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = Accent,
            unfocusedBorderColor = PanelBorder,
            focusedTextColor = Color(0xFFE6E6E6),
            unfocusedTextColor = Color(0xFFE6E6E6),
            focusedContainerColor = InputBg,
            unfocusedContainerColor = InputBg,
            focusedLabelColor = MutedText,
            unfocusedLabelColor = MutedText,
            cursorColor = Accent
        ),
        shape = RoundedCornerShape(6.dp)
    )
}

@Composable
fun Panel(content: @Composable () -> Unit) {
    Surface(
        color = PanelColor,
        shape = RoundedCornerShape(6.dp),
        border = BorderStroke(1.dp, PanelBorder),
        shadowElevation = 6.dp,
        modifier = Modifier.padding(2.dp)
    ) {
        androidx.compose.foundation.layout.Column(
            modifier = Modifier.padding(10.dp),
            content = { content() }
        )
    }
}

fun appBackgroundBrush(): Brush =
    Brush.radialGradient(
        colors = listOf(Color(0xFF1A1C20), Color(0xFF0F1012)),
        radius = 900f
    )
