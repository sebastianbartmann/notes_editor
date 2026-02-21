package com.bartmann.noteseditor

import androidx.activity.compose.BackHandler
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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun ToolClaudeScreen(modifier: Modifier) {
    val person = UserSettings.person
    var inputText by remember(person) { mutableStateOf(ClaudeSessionStore.draftInput(person)) }
    var isLoading by remember { mutableStateOf(false) }
    var activeRemoteRunId by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf("") }
    var actions by remember { mutableStateOf<List<AgentAction>>(emptyList()) }
    var actionsError by remember { mutableStateOf("") }
    var pendingConfirmation by remember { mutableStateOf<AgentAction?>(null) }
    var showSessionsDialog by remember { mutableStateOf(false) }
    var sessions by remember { mutableStateOf<List<AgentSessionSummary>>(emptyList()) }
    var sessionsError by remember { mutableStateOf("") }
    var sessionsStatus by remember { mutableStateOf("") }
    var sessionsLoading by remember { mutableStateOf(false) }
    var sessionsBusy by remember { mutableStateOf(false) }
    var streamingAssistantText by remember { mutableStateOf("") }
    var lastPerson by remember { mutableStateOf<String?>(null) }
    val messages = ClaudeSessionStore.messages
    val visibleMessages = if (UserSettings.agentVerboseOutput) {
        messages
    } else {
        messages.filter {
            it.type != "tool_call" &&
            it.type != "tool_result" &&
            it.type != "status" &&
            it.type != "usage"
        }
    }
    val scope = rememberCoroutineScope()
    var refreshJob by remember { mutableStateOf<Job?>(null) }
    var postSwitchPollJob by remember { mutableStateOf<Job?>(null) }
    var activeStreamJob by remember { mutableStateOf<Job?>(null) }
    val listState = rememberLazyListState()
    var autoScrollEnabled by remember { mutableStateOf(true) }
    val autoScrollThresholdPx = 50
    val hasActiveRun = isLoading || activeRemoteRunId != null
    val latestUsage = messages.lastOrNull { it.type == "usage" && it.usage != null }?.usage
    val usageSummary = if (!UserSettings.agentVerboseOutput) {
        "Verbose output disabled"
    } else if (latestUsage?.remainingTokens != null && latestUsage.contextWindow != null && latestUsage.contextWindow > 0) {
        "Context: ${latestUsage.totalTokens ?: 0} used, ${latestUsage.remainingTokens} left of ${latestUsage.contextWindow}"
    } else if (latestUsage != null) {
        "Context: ${latestUsage.totalTokens ?: 0} tokens used"
    } else {
        "Context: not available yet"
    }

    fun flushStreamingAssistantText() {
        if (streamingAssistantText.isBlank()) return
        messages.add(AgentConversationItem(type = "message", role = "assistant", content = streamingAssistantText))
        streamingAssistantText = ""
    }

    fun sendMessage() {
        val text = inputText.trim()
        if (text.isEmpty() || hasActiveRun) return
        inputText = ""
        ClaudeSessionStore.updateDraftInput(person, "")
        activeRemoteRunId = null
        isLoading = true
        statusMessage = "Connecting..."
        streamingAssistantText = ""
        messages.add(AgentConversationItem(type = "message", role = "user", content = text))

        scope.launch {
            try {
                ApiClient.agentChatStream(text, ClaudeSessionStore.sessionId).collect { event ->
                    when (event.type) {
                        "start" -> {
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                        }
                        "text" -> {
                            streamingAssistantText += event.delta.orEmpty()
                        }
                        "tool_call" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "tool_call",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        tool = event.tool,
                                        args = event.args
                                    )
                                )
                            }
                        }
                        "tool_result" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "tool_result",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        tool = event.tool,
                                        ok = event.ok,
                                        summary = event.summary
                                    )
                                )
                            }
                        }
                        "status" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "status",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        message = event.message
                                    )
                                )
                            }
                        }
                        "done" -> {
                            flushStreamingAssistantText()
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                            statusMessage = ""
                        }
                        "error" -> {
                            flushStreamingAssistantText()
                            val errorText = event.message ?: "stream error"
                            messages.add(
                                AgentConversationItem(
                                    type = "error",
                                    runId = event.runId,
                                    seq = event.seq,
                                    ts = event.ts,
                                    message = errorText
                                )
                            )
                            statusMessage = "Error: $errorText"
                        }
                        "usage" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "usage",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        usage = event.usage
                                    )
                                )
                            }
                        }
                    }
                }
            } catch (_: CancellationException) {
                // Session switch/new session can cancel the active stream intentionally.
            } catch (exc: Exception) {
                flushStreamingAssistantText()
                statusMessage = "Error: ${exc.message}"
            } finally {
                flushStreamingAssistantText()
                isLoading = false
                activeStreamJob = null
                if (!statusMessage.startsWith("Error")) {
                    statusMessage = ""
                }
            }
        }.also { activeStreamJob = it }
    }

    fun runAction(action: AgentAction) {
        if (hasActiveRun) return
        if (action.metadata.requiresConfirmation && pendingConfirmation?.id != action.id) {
            pendingConfirmation = action
            statusMessage = "Confirm action: ${action.label}"
            return
        }
        pendingConfirmation = null
        activeRemoteRunId = null
        isLoading = true
        statusMessage = "Running ${action.label}..."
        streamingAssistantText = ""
        messages.add(AgentConversationItem(type = "message", role = "user", content = "Run action: ${action.label}"))

        scope.launch {
            try {
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
                            streamingAssistantText += event.delta.orEmpty()
                        }
                        "tool_call" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "tool_call",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        tool = event.tool,
                                        args = event.args
                                    )
                                )
                            }
                        }
                        "tool_result" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "tool_result",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        tool = event.tool,
                                        ok = event.ok,
                                        summary = event.summary
                                    )
                                )
                            }
                        }
                        "status" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "status",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        message = event.message
                                    )
                                )
                            }
                        }
                        "done" -> {
                            flushStreamingAssistantText()
                            if (!event.sessionId.isNullOrBlank()) {
                                ClaudeSessionStore.sessionId = event.sessionId
                            }
                            statusMessage = ""
                        }
                        "error" -> {
                            flushStreamingAssistantText()
                            val errorText = event.message ?: "stream error"
                            messages.add(
                                AgentConversationItem(
                                    type = "error",
                                    runId = event.runId,
                                    seq = event.seq,
                                    ts = event.ts,
                                    message = errorText
                                )
                            )
                            statusMessage = "Error: $errorText"
                        }
                        "usage" -> {
                            flushStreamingAssistantText()
                            if (UserSettings.agentVerboseOutput) {
                                messages.add(
                                    AgentConversationItem(
                                        type = "usage",
                                        runId = event.runId,
                                        seq = event.seq,
                                        ts = event.ts,
                                        usage = event.usage
                                    )
                                )
                            }
                        }
                    }
                }
            } catch (_: CancellationException) {
                // Session switch/new session can cancel the active stream intentionally.
            } catch (exc: Exception) {
                flushStreamingAssistantText()
                statusMessage = "Error: ${exc.message}"
            } finally {
                flushStreamingAssistantText()
                isLoading = false
                activeStreamJob = null
                if (!statusMessage.startsWith("Error")) {
                    statusMessage = ""
                }
            }
        }.also { activeStreamJob = it }
    }

    fun stopActiveStreamForSessionSwitch() {
        activeStreamJob?.cancel()
        activeStreamJob = null
        flushStreamingAssistantText()
        isLoading = false
    }

    fun startNewSession() {
        stopActiveStreamForSessionSwitch()
        refreshJob?.cancel()
        refreshJob = null
        postSwitchPollJob?.cancel()
        postSwitchPollJob = null
        ClaudeSessionStore.startNew()
        activeRemoteRunId = null
        pendingConfirmation = null
        statusMessage = ""
        streamingAssistantText = ""
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
        if (person == null) return
        ClaudeSessionStore.saveCurrentToCache()
        showSessionsDialog = true
        sessionsStatus = ""
        loadSessions()
    }

    fun startPostSwitchPolling(targetSessionId: String) {
        postSwitchPollJob?.cancel()
        val baseline = ClaudeSessionStore.messages.toList()
        postSwitchPollJob = scope.launch {
            var previousHistory = baseline
            var unchangedCount = 0
            while (unchangedCount < 2) {
                delay(3000)
                if (ClaudeSessionStore.sessionId != targetSessionId) break
                if (isLoading) continue
                val resp = try {
                    ApiClient.fetchAgentSessionHistory(targetSessionId)
                } catch (_: Exception) {
                    break
                }
                val history = resp.toItems()
                if (history == previousHistory) {
                    unchangedCount += 1
                } else {
                    unchangedCount = 0
                }
                previousHistory = history
                if (ClaudeSessionStore.sessionId != targetSessionId) break
                ClaudeSessionStore.loadSession(targetSessionId, history)
                activeRemoteRunId = resp.activeRun?.runId
                statusMessage = ""
                streamingAssistantText = ""
            }
        }.also { job ->
            job.invokeOnCompletion {
                if (postSwitchPollJob == job) {
                    postSwitchPollJob = null
                }
            }
        }
    }

    fun continueSession(targetSessionId: String) {
        if (person == null || sessionsBusy) return
        if (targetSessionId == ClaudeSessionStore.sessionId) {
            showSessionsDialog = false
            return
        }
        stopActiveStreamForSessionSwitch()
        refreshJob?.cancel()
        refreshJob = null
        postSwitchPollJob?.cancel()
        postSwitchPollJob = null
        if (ClaudeSessionStore.isInCache(targetSessionId)) {
            ClaudeSessionStore.switchTo(targetSessionId)
            statusMessage = ""
            pendingConfirmation = null
            streamingAssistantText = ""
            showSessionsDialog = false
            startPostSwitchPolling(targetSessionId)
            return
        }
        scope.launch {
            sessionsBusy = true
            sessionsError = ""
            sessionsStatus = ""
            try {
                val resp = ApiClient.fetchAgentSessionHistory(targetSessionId)
                val history = resp.toItems()
                ClaudeSessionStore.switchTo(targetSessionId, history)
                activeRemoteRunId = resp.activeRun?.runId
                statusMessage = ""
                pendingConfirmation = null
                streamingAssistantText = ""
                showSessionsDialog = false
                startPostSwitchPolling(targetSessionId)
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
            sessionsStatus = ""
            try {
                ApiClient.clearAllAgentSessions()
                ClaudeSessionStore.clear()
                ClaudeSessionStore.clearCache()
                sessions = emptyList()
                statusMessage = ""
                pendingConfirmation = null
                streamingAssistantText = ""
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
            sessionsStatus = ""
            try {
                ApiClient.clearAgentSession(targetSessionId)
                sessions = sessions.filter { it.sessionId != targetSessionId }
                if (ClaudeSessionStore.sessionId == targetSessionId) {
                    ClaudeSessionStore.clear()
                    statusMessage = ""
                    pendingConfirmation = null
                    streamingAssistantText = ""
                }
                ClaudeSessionStore.removeFromCache(targetSessionId)
            } catch (exc: Exception) {
                sessionsError = "Failed to delete session: ${exc.message}"
            } finally {
                sessionsBusy = false
            }
        }
    }

    fun exportSessionsMarkdown() {
        if (isLoading || person == null || sessionsBusy) return
        scope.launch {
            sessionsBusy = true
            sessionsError = ""
            sessionsStatus = ""
            try {
                val exported = ApiClient.exportAgentSessionsMarkdown()
                val sessionCount = (exported.files.size - 1).coerceAtLeast(0)
                sessionsStatus = "Exported $sessionCount session file(s) to ${exported.directory}"
            } catch (exc: Exception) {
                sessionsError = "Failed to export sessions: ${exc.message}"
            } finally {
                sessionsBusy = false
            }
        }
    }

    fun refreshCurrentSessionHistory() {
        val currentSessionId = ClaudeSessionStore.sessionId
        if (person == null || currentSessionId.isNullOrBlank() || isLoading) return
        refreshJob?.cancel()
        refreshJob = scope.launch {
            try {
                val resp = ApiClient.fetchAgentSessionHistory(currentSessionId)
                val history = resp.toItems()
                ClaudeSessionStore.loadSession(currentSessionId, history)
                activeRemoteRunId = resp.activeRun?.runId
                statusMessage = ""
                streamingAssistantText = ""
            } catch (_: Exception) {
                // Non-fatal refresh: keep currently rendered local state.
            }
        }
    }

    BackHandler(enabled = showSessionsDialog && !sessionsBusy) {
        showSessionsDialog = false
    }

    fun isNearBottom(): Boolean {
        val layoutInfo = listState.layoutInfo
        val totalItems = layoutInfo.totalItemsCount
        if (totalItems == 0) return true
        val lastVisible = layoutInfo.visibleItemsInfo.lastOrNull() ?: return true
        val lastIndex = totalItems - 1
        if (lastVisible.index < lastIndex) return false
        val distanceToBottom = layoutInfo.viewportEndOffset - (lastVisible.offset + lastVisible.size)
        return distanceToBottom >= -autoScrollThresholdPx
    }

    LaunchedEffect(listState) {
        snapshotFlow { listState.isScrollInProgress }.collect { scrolling ->
            if (scrolling) {
                autoScrollEnabled = isNearBottom()
            } else if (isNearBottom()) {
                autoScrollEnabled = true
            }
        }
    }

    LaunchedEffect(visibleMessages.size, streamingAssistantText, isLoading) {
        val totalItems = visibleMessages.size + if (streamingAssistantText.isNotBlank() || isLoading) 1 else 0
        if (totalItems > 0 && autoScrollEnabled) {
            listState.scrollToItem(totalItems - 1)
        }
    }

    LaunchedEffect(person) {
        if (lastPerson != null && person != lastPerson) {
            postSwitchPollJob?.cancel()
            postSwitchPollJob = null
            ClaudeSessionStore.clear()
            activeRemoteRunId = null
            pendingConfirmation = null
            statusMessage = ""
            streamingAssistantText = ""
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

    LaunchedEffect(person, ClaudeSessionStore.sessionId, isLoading) {
        if (person == null || isLoading) return@LaunchedEffect
        if (ClaudeSessionStore.sessionId.isNullOrBlank()) return@LaunchedEffect
        refreshCurrentSessionHistory()
    }

    LaunchedEffect(person, ClaudeSessionStore.sessionId, activeRemoteRunId) {
        val sid = ClaudeSessionStore.sessionId
        if (person == null || sid.isNullOrBlank() || activeRemoteRunId == null) return@LaunchedEffect
        while (true) {
            delay(3000)
            if (isLoading) continue
            if (ClaudeSessionStore.sessionId != sid) break
            try {
                val resp = ApiClient.fetchAgentSessionHistory(sid)
                val history = resp.toItems()
                ClaudeSessionStore.loadSession(sid, history)
                if (resp.activeRun == null) {
                    activeRemoteRunId = null
                    break
                }
                activeRemoteRunId = resp.activeRun.runId
            } catch (_: Exception) {
                break
            }
        }
    }

    ScreenLayout(modifier = modifier) {
        if (showSessionsDialog) {
            ScreenHeader(
                title = "Sessions",
                actionButton = {
                    CompactTextButton(text = "Close") {
                        if (!sessionsBusy) showSessionsDialog = false
                    }
                }
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                CompactButton(
                    text = if (sessionsBusy) "Working..." else "Export .md",
                    modifier = Modifier.weight(1f),
                    onClick = { exportSessionsMarkdown() }
                )
                CompactButton(
                    text = if (sessionsBusy) "Deleting..." else "Delete all",
                    modifier = Modifier.weight(1f),
                    background = AppTheme.colors.button,
                    border = AppTheme.colors.danger,
                    textColor = AppTheme.colors.danger,
                    onClick = { deleteAllSessions() }
                )
            }

            if (sessionsLoading) {
                AppText("Loading sessions...", AppTheme.typography.bodySmall, AppTheme.colors.muted)
            }
            if (sessionsError.isNotBlank()) {
                SelectableAppText(
                    text = sessionsError,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.danger
                )
            }
            if (sessionsStatus.isNotBlank()) {
                SelectableAppText(
                    text = sessionsStatus,
                    style = AppTheme.typography.bodySmall,
                    color = AppTheme.colors.muted
                )
            }

            AppText(
                "Saved sessions (${sessions.size})",
                AppTheme.typography.body,
                AppTheme.colors.text
            )
            Panel(modifier = Modifier.weight(1f)) {
                if (sessions.isEmpty() && sessionsError.isBlank() && !sessionsLoading) {
                    AppText("No sessions yet.", AppTheme.typography.bodySmall, AppTheme.colors.muted)
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        items(sessions) { session ->
                            SessionRow(
                                session = session,
                                active = session.sessionId == ClaudeSessionStore.sessionId,
                                onClick = { continueSession(session.sessionId) },
                                onDelete = { deleteSession(session.sessionId) }
                            )
                        }
                    }
                }
            }
        } else {
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

            StatusMessage(
                text = "Session: ${ClaudeSessionStore.sessionId ?: "new"}${if (activeRemoteRunId != null) "  |  Running..." else ""}  |  $usageSummary",
                showDivider = false
            )

            Panel(modifier = Modifier.weight(1f)) {
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(visibleMessages) { message ->
                        ChatBubble(message = message)
                    }
                    if (streamingAssistantText.isNotBlank()) {
                        item {
                            ChatBubble(message = AgentConversationItem(type = "message", role = "assistant", content = streamingAssistantText))
                        }
                    }
                    if (isLoading && streamingAssistantText.isBlank()) {
                        item {
                            ChatBubble(message = AgentConversationItem(type = "message", role = "assistant", content = "..."))
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
                        text = if (isLoading) "..." else if (activeRemoteRunId != null) "Running" else "Send",
                        modifier = Modifier
                            .fillMaxHeight()
                            .width(84.dp),
                        onClick = { if (!hasActiveRun && inputText.isNotBlank()) sendMessage() }
                    )
                }
                if (isLoading) {
                    StatusMessage(text = "Sending...")
                } else if (activeRemoteRunId != null) {
                    StatusMessage(text = "Session is running in background...")
                } else if (statusMessage.isNotEmpty()) {
                    StatusMessage(text = statusMessage)
                }
            }
        }
    }
}

@Composable
private fun ActiveRunRow(run: AgentActiveRun, onStop: () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(6.dp))
            .background(AppTheme.colors.button)
            .border(width = 1.dp, color = AppTheme.colors.panelBorder, shape = RoundedCornerShape(6.dp))
            .padding(horizontal = 10.dp, vertical = 8.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                AppText("Run: ${run.runId}", AppTheme.typography.bodySmall, AppTheme.colors.text)
                AppText(
                    "Session: ${run.sessionId ?: "new"}  |  updated ${formatSessionTimestamp(run.updatedAt)}",
                    AppTheme.typography.label,
                    AppTheme.colors.muted
                )
            }
            CompactButton(
                text = "Stop",
                modifier = Modifier.width(84.dp),
                background = AppTheme.colors.button,
                border = AppTheme.colors.danger,
                textColor = AppTheme.colors.danger,
                onClick = onStop
            )
        }
    }
}

@Composable
private fun ChatBubble(message: AgentConversationItem) {
    if (message.type != "message") {
        val (title, detail) = when (message.type) {
            "tool_call" -> {
                val argsText = formatAgentArgs(message.args)
                "Tool call: ${message.tool ?: "unknown"}" to argsText
            }
            "tool_result" -> {
                val status = if (message.ok == false) "failed" else "finished"
                "Tool ${message.tool ?: "unknown"} $status" to message.summary.orEmpty()
            }
            "status" -> (message.message ?: "Status update") to ""
            "usage" -> {
                val total = message.usage?.totalTokens ?: 0
                val remaining = message.usage?.remainingTokens
                val window = message.usage?.contextWindow
                if (remaining != null && window != null && window > 0) {
                    "Usage: $total tokens, $remaining left of $window" to ""
                } else {
                    "Usage: $total tokens" to ""
                }
            }
            else -> (message.message ?: "Error") to ""
        }
        val border = when (message.type) {
            "tool_call", "usage" -> AppTheme.colors.accent
            "error" -> AppTheme.colors.danger
            else -> AppTheme.colors.panelBorder
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(6.dp))
                .background(AppTheme.colors.button)
                .border(width = 1.dp, color = border, shape = RoundedCornerShape(6.dp))
                .padding(horizontal = 10.dp, vertical = 8.dp)
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                SelectionContainer {
                    AppText(
                        text = title,
                        style = AppTheme.typography.bodySmall,
                        color = if (message.type == "error") AppTheme.colors.danger else AppTheme.colors.text
                    )
                }
                if (detail.isNotBlank()) {
                    SelectionContainer {
                        AppText(
                            text = detail,
                            style = AppTheme.typography.label,
                            color = AppTheme.colors.muted
                        )
                    }
                }
            }
        }
        return
    }

    val isUser = message.role == "user"
    val bgColor = if (isUser) AppTheme.colors.accentDim else AppTheme.colors.input

    Column(
        modifier = Modifier
            .fillMaxWidth()
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
                text = message.content.orEmpty(),
                style = AppTheme.typography.bodySmall,
                color = AppTheme.colors.text
            )
        }
    }
}

private fun formatAgentArgs(args: kotlinx.serialization.json.JsonElement?): String {
    val raw = args?.toString().orEmpty()
    if (raw.isBlank()) return ""
    return if (raw.length > 320) raw.take(320) + "..." else raw
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
