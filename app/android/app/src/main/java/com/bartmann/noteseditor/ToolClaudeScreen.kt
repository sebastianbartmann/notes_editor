package com.bartmann.noteseditor

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.TextButton
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
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun ToolClaudeScreen(modifier: Modifier) {
    val person = UserSettings.person
    var inputText by remember(person) { mutableStateOf(ClaudeSessionStore.draftInput(person)) }
    var isLoading by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf("") }
    var actions by remember { mutableStateOf<List<AgentAction>>(emptyList()) }
    var actionsError by remember { mutableStateOf("") }
    var pendingConfirmation by remember { mutableStateOf<AgentAction?>(null) }
    var showSessionsDialog by remember { mutableStateOf(false) }
    var sessions by remember { mutableStateOf<List<AgentSessionSummary>>(emptyList()) }
    var sessionsError by remember { mutableStateOf("") }
    var sessionsLoading by remember { mutableStateOf(false) }
    var sessionsBusy by remember { mutableStateOf(false) }
    var lastPerson by remember { mutableStateOf<String?>(null) }
    val messages = ClaudeSessionStore.messages
    val scope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    fun sendMessage() {
        val text = inputText.trim()
        if (text.isEmpty() || isLoading) return
        inputText = ""
        ClaudeSessionStore.updateDraftInput(person, "")
        isLoading = true
        statusMessage = "Connecting..."
        messages.add(ChatMessage(role = "user", content = text))

        scope.launch {
            try {
                val assistantIndex = messages.size
                messages.add(ChatMessage(role = "assistant", content = ""))
                var assistantText = ""
                ApiClient.agentChatStream(text, ClaudeSessionStore.sessionId).collect { event ->
                    when (event.type) {
                        "start" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                        }
                        "text" -> {
                            assistantText += event.delta.orEmpty()
                            messages[assistantIndex] = ChatMessage(role = "assistant", content = assistantText)
                        }
                        "tool_call" -> {
                            val url = event.args
                                ?.jsonObject
                                ?.get("url")
                                ?.jsonPrimitive
                                ?.contentOrNull
                            statusMessage = if (url != null) {
                                "Tool: ${event.tool} $url"
                            } else {
                                "Tool: ${event.tool ?: "working"}"
                            }
                        }
                        "tool_result" -> {
                            statusMessage = event.summary
                                ?: "Tool ${event.tool ?: "unknown"} finished"
                        }
                        "status" -> {
                            statusMessage = event.message ?: "Working..."
                        }
                        "done" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                            statusMessage = ""
                        }
                        "error" -> {
                            val errorText = event.message ?: "stream error"
                            statusMessage = "Error: $errorText"
                            if (assistantText.isBlank()) {
                                messages[assistantIndex] = ChatMessage(
                                    role = "assistant",
                                    content = "Error: $errorText"
                                )
                            }
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

    fun runAction(action: AgentAction) {
        if (isLoading) return
        if (action.metadata.requiresConfirmation && pendingConfirmation?.id != action.id) {
            pendingConfirmation = action
            statusMessage = "Confirm action: ${action.label}"
            return
        }
        pendingConfirmation = null
        isLoading = true
        statusMessage = "Running ${action.label}..."
        messages.add(ChatMessage(role = "user", content = "Run action: ${action.label}"))

        scope.launch {
            try {
                val assistantIndex = messages.size
                messages.add(ChatMessage(role = "assistant", content = ""))
                var assistantText = ""
                ApiClient.agentChatStream(
                    message = "",
                    sessionId = ClaudeSessionStore.sessionId,
                    actionId = action.id,
                    confirm = action.metadata.requiresConfirmation
                ).collect { event ->
                    when (event.type) {
                        "start" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                        }
                        "text" -> {
                            assistantText += event.delta.orEmpty()
                            messages[assistantIndex] = ChatMessage(role = "assistant", content = assistantText)
                        }
                        "tool_call" -> {
                            statusMessage = "Tool: ${event.tool ?: "working"}"
                        }
                        "tool_result" -> {
                            statusMessage = event.summary ?: "Tool finished"
                        }
                        "status" -> {
                            statusMessage = event.message ?: "Working..."
                        }
                        "done" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                            statusMessage = ""
                        }
                        "error" -> {
                            val errorText = event.message ?: "stream error"
                            statusMessage = "Error: $errorText"
                            if (assistantText.isBlank()) {
                                messages[assistantIndex] = ChatMessage(
                                    role = "assistant",
                                    content = "Error: $errorText"
                                )
                            }
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

    fun startNewSession() {
        if (isLoading) return
        ClaudeSessionStore.clear()
        pendingConfirmation = null
        statusMessage = ""
    }

    fun loadSessions() {
        if (person == null) return
        scope.launch {
            sessionsLoading = true
            sessionsError = ""
            try {
                sessions = ApiClient.fetchAgentSessions()
            } catch (exc: Exception) {
                sessions = emptyList()
                sessionsError = "Failed to load sessions: ${exc.message}"
            } finally {
                sessionsLoading = false
            }
        }
    }

    fun openSessions() {
        if (isLoading || person == null) return
        showSessionsDialog = true
        loadSessions()
    }

    fun continueSession(targetSessionId: String) {
        if (isLoading || person == null || sessionsBusy) return
        scope.launch {
            sessionsBusy = true
            sessionsError = ""
            try {
                val history = ApiClient.fetchAgentSessionHistory(targetSessionId)
                ClaudeSessionStore.loadSession(targetSessionId, history)
                statusMessage = ""
                pendingConfirmation = null
                showSessionsDialog = false
            } catch (exc: Exception) {
                sessionsError = "Failed to open session: ${exc.message}"
            } finally {
                sessionsBusy = false
            }
        }
    }

    fun deleteAllSessions() {
        if (isLoading || person == null || sessionsBusy) return
        scope.launch {
            sessionsBusy = true
            sessionsError = ""
            try {
                ApiClient.clearAllAgentSessions()
                ClaudeSessionStore.clear()
                sessions = emptyList()
                showSessionsDialog = false
                statusMessage = ""
                pendingConfirmation = null
            } catch (exc: Exception) {
                sessionsError = "Failed to delete sessions: ${exc.message}"
            } finally {
                sessionsBusy = false
            }
        }
    }

    fun deleteSession(targetSessionId: String) {
        if (isLoading || person == null || sessionsBusy) return
        scope.launch {
            sessionsBusy = true
            sessionsError = ""
            try {
                ApiClient.clearAgentSession(targetSessionId)
                sessions = sessions.filter { it.sessionId != targetSessionId }
                if (ClaudeSessionStore.sessionId == targetSessionId) {
                    ClaudeSessionStore.clear()
                    statusMessage = ""
                    pendingConfirmation = null
                }
            } catch (exc: Exception) {
                sessionsError = "Failed to delete session: ${exc.message}"
            } finally {
                sessionsBusy = false
            }
        }
    }

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.size - 1)
        }
    }

    LaunchedEffect(person) {
        if (lastPerson != null && person != lastPerson) {
            ClaudeSessionStore.clear()
            pendingConfirmation = null
            statusMessage = ""
        }
        lastPerson = person
        if (person == null) return@LaunchedEffect
        try {
            actions = ApiClient.listAgentActions()
            actionsError = ""
        } catch (exc: Exception) {
            actions = emptyList()
            actionsError = "Failed to load actions: ${exc.message}"
        }
    }

    ScreenLayout(modifier = modifier) {
        ScreenHeader(
            title = "Agent",
            actionButton = {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    CompactTextButton(text = "Sessions") {
                        openSessions()
                    }
                    CompactTextButton(text = "New") {
                        startNewSession()
                    }
                }
            }
        )

        if (actions.isNotEmpty()) {
            LazyRow(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(actions) { action ->
                    CompactButton(
                        text = if (action.metadata.requiresConfirmation) "${action.label} *" else action.label,
                        onClick = { runAction(action) }
                    )
                }
            }
        } else if (actionsError.isNotBlank()) {
            StatusMessage(text = actionsError, showDivider = false)
        } else if (person != null) {
            StatusMessage(
                text = "No actions for this person.",
                showDivider = false
            )
        }
        if (pendingConfirmation != null) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CompactButton(
                    text = "Confirm: ${pendingConfirmation?.label}",
                    modifier = Modifier.weight(1f),
                    background = AppTheme.colors.accentDim,
                    border = AppTheme.colors.accent,
                    onClick = { pendingConfirmation?.let { runAction(it) } }
                )
                CompactButton(
                    text = "Cancel",
                    modifier = Modifier.width(96.dp),
                    onClick = {
                        pendingConfirmation = null
                        statusMessage = ""
                    }
                )
            }
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
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.Bottom
            ) {
                CompactTextField(
                    value = inputText,
                    onValueChange = {
                        inputText = it
                        ClaudeSessionStore.updateDraftInput(person, it)
                    },
                    placeholder = "Ask Agent...",
                    modifier = Modifier.weight(1f),
                    minLines = 2
                )
                CompactButton(
                    text = if (isLoading) "..." else "Send",
                    modifier = Modifier
                        .fillMaxHeight()
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

        if (showSessionsDialog) {
            AlertDialog(
                onDismissRequest = {
                    if (!sessionsBusy) {
                        showSessionsDialog = false
                    }
                },
                containerColor = AppTheme.colors.panel,
                iconContentColor = AppTheme.colors.text,
                titleContentColor = AppTheme.colors.text,
                textContentColor = AppTheme.colors.text,
                title = {
                    AppText("Sessions", AppTheme.typography.title, AppTheme.colors.text)
                },
                text = {
                    Column(
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.verticalScroll(rememberScrollState())
                    ) {
                        if (sessionsLoading) {
                            AppText("Loading sessions...", AppTheme.typography.bodySmall, AppTheme.colors.muted)
                        } else if (sessions.isEmpty() && sessionsError.isBlank()) {
                            AppText("No sessions yet.", AppTheme.typography.bodySmall, AppTheme.colors.muted)
                        }

                        if (sessionsError.isNotBlank()) {
                            SelectableAppText(
                                text = sessionsError,
                                style = AppTheme.typography.bodySmall,
                                color = AppTheme.colors.danger
                            )
                        }

                        sessions.forEach { session ->
                            SessionRow(
                                session = session,
                                active = session.sessionId == ClaudeSessionStore.sessionId,
                                onClick = { continueSession(session.sessionId) },
                                onDelete = { deleteSession(session.sessionId) }
                            )
                        }
                    }
                },
                confirmButton = {
                    TextButton(onClick = {
                        if (!sessionsBusy) {
                            showSessionsDialog = false
                        }
                    }) {
                        AppText("Close", AppTheme.typography.label, AppTheme.colors.muted)
                    }
                },
                dismissButton = {
                    TextButton(onClick = {
                        if (!sessionsBusy) {
                            deleteAllSessions()
                        }
                    }) {
                        AppText(
                            if (sessionsBusy) "Deleting..." else "Delete all",
                            AppTheme.typography.label,
                            AppTheme.colors.danger
                        )
                    }
                }
            )
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
                text = if (isUser) "You" else "Agent",
                style = AppTheme.typography.label,
                color = AppTheme.colors.muted
            )
            SelectionContainer {
                AppText(
                    text = message.content,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.text
                )
            }
        }
    }
}

@Composable
private fun SessionRow(
    session: AgentSessionSummary,
    active: Boolean,
    onClick: () -> Unit,
    onDelete: () -> Unit
) {
    val border = if (active) AppTheme.colors.accent else AppTheme.colors.panelBorder
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(6.dp))
            .background(AppTheme.colors.input)
            .border(width = 1.dp, color = border, shape = RoundedCornerShape(6.dp)),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(
            modifier = Modifier
                .weight(1f)
                .clickable(onClick = onClick)
                .padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp)
        ) {
            AppText(session.name, AppTheme.typography.body, AppTheme.colors.text)
            AppText(
                "${session.messageCount} msgs - ${formatSessionTimestamp(session.lastUsedAt)}",
                AppTheme.typography.label,
                AppTheme.colors.muted
            )
            if (!session.lastPreview.isNullOrBlank()) {
                AppText(session.lastPreview, AppTheme.typography.bodySmall, AppTheme.colors.muted)
            }
        }

        IconButton(onClick = onDelete) {
            Icon(
                imageVector = Icons.Default.Delete,
                contentDescription = "Delete session",
                tint = AppTheme.colors.danger
            )
        }
    }
}

private fun formatSessionTimestamp(raw: String): String {
    return try {
        val instant = Instant.parse(raw)
        DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")
            .withZone(ZoneId.systemDefault())
            .format(instant)
    } catch (_: Exception) {
        raw
    }
}
