package com.bartmann.noteseditor

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import kotlinx.coroutines.launch

@Composable
fun ToolClaudeScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var prompt by remember { mutableStateOf("") }
    var responseText by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    ScreenLayout(
        modifier = modifier,
        padding = padding
    ) {
        ScreenTitle(text = "Claude")
        Panel {
            SectionTitle(text = "Prompt")
            CompactTextField(
                value = prompt,
                onValueChange = { prompt = it },
                placeholder = "Ask Claude...",
                modifier = Modifier.fillMaxWidth(),
                minLines = 3
            )
            CompactButton(text = "Run") {
                scope.launch {
                    try {
                        val response = ApiClient.runClaude(prompt)
                        responseText = response.response
                        message = response.message
                    } catch (exc: Exception) {
                        message = "Claude failed: ${exc.message}"
                    }
                }
            }
            if (responseText.isNotBlank()) {
                CompactDivider()
                AppText(
                    text = responseText,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.text
                )
            }
            StatusMessage(text = message)
        }
    }
}
