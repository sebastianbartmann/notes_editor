package com.bartmann.noteseditor

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.contentOrNull

@Composable
fun ToolClaudeScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var inputText by remember { mutableStateOf("") }
    var isLoading by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf("") }
    val messages = ClaudeSessionStore.messages
    val scope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    fun sendMessage() {
        val text = inputText.trim()
        if (text.isEmpty() || isLoading) return
        inputText = ""
        isLoading = true
        statusMessage = "Connecting..."
        messages.add(ChatMessage(role = "user", content = text))

        scope.launch {
            try {
                val assistantIndex = messages.size
                messages.add(ChatMessage(role = "assistant", content = ""))
                var assistantText = ""
                ApiClient.claudeChatStream(text, ClaudeSessionStore.sessionId).collect { event ->
                    when (event.type) {
                        "text" -> {
                            assistantText += event.delta.orEmpty()
                            messages[assistantIndex] = ChatMessage(role = "assistant", content = assistantText)
                        }
                        "status" -> {
                            statusMessage = event.message ?: "Working..."
                        }
                        "tool" -> {
                            val url = event.input
                                ?.jsonObject
                                ?.get("url")
                                ?.jsonPrimitive
                                ?.contentOrNull
                            statusMessage = if (url != null) {
                                "Tool: ${event.name} $url"
                            } else {
                                "Tool: ${event.name ?: "working"}"
                            }
                        }
                        "ping" -> {
                            // Keep-alive event; no UI update.
                        }
                        "done" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                            statusMessage = ""
                        }
                        "error" -> {
                            statusMessage = "Error: ${event.message}"
                        }
                    }
                }
            } catch (exc: Exception) {
                statusMessage = "Error: ${exc.message}"
            } finally {
                isLoading = false
                if (!statusMessage.startsWith("Error")) {
                    statusMessage = ""
                }
            }
        }
    }

    fun clearChat() {
        scope.launch {
            val currentSessionId = ClaudeSessionStore.sessionId
            if (currentSessionId != null) {
                ApiClient.claudeClear(currentSessionId)
            }
            ClaudeSessionStore.clear()
            statusMessage = ""
        }
    }

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.size - 1)
        }
    }

    ScreenLayout(modifier = modifier, padding = padding) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            ScreenTitle(text = "Claude")
            CompactButton(text = "Clear", onClick = ::clearChat)
        }

        Panel(modifier = Modifier.weight(1f)) {
            LazyColumn(
                state = listState,
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(messages) { message ->
                    ChatBubble(message = message)
                }
                if (isLoading) {
                    item {
                        ChatBubble(message = ChatMessage(role = "assistant", content = "..."))
                    }
                }
            }
        }

        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.Bottom
            ) {
                CompactTextField(
                    value = inputText,
                    onValueChange = { inputText = it },
                    placeholder = "Ask Claude...",
                    modifier = Modifier.weight(1f),
                    minLines = 2
                )
                CompactButton(
                    text = if (isLoading) "..." else "Send",
                    modifier = Modifier
                        .height(48.dp)
                        .width(84.dp),
                    onClick = { if (!isLoading && inputText.isNotBlank()) sendMessage() }
                )
            }
            if (isLoading) {
                StatusMessage(text = "Sending...")
            } else if (statusMessage.isNotEmpty()) {
                StatusMessage(text = statusMessage)
            }
        }
    }
}

@Composable
private fun ChatBubble(message: ChatMessage) {
    val isUser = message.role == "user"
    val alignment = if (isUser) Alignment.CenterEnd else Alignment.CenterStart
    val bgColor = if (isUser) AppTheme.colors.accentDim else AppTheme.colors.input

    Box(
        modifier = Modifier.fillMaxWidth(),
        contentAlignment = alignment
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth(0.85f)
                .clip(RoundedCornerShape(8.dp))
                .background(bgColor)
                .padding(10.dp)
        ) {
            AppText(
                text = if (isUser) "You" else "Claude",
                style = AppTheme.typography.label,
                color = AppTheme.colors.muted
            )
            AppText(
                text = message.content,
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.text
            )
        }
    }
}
