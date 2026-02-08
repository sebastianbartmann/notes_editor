package com.bartmann.noteseditor

import java.io.IOException
import java.net.URLEncoder
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.channelFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okio.BufferedSource

object ApiClient {
    private val json = Json { ignoreUnknownKeys = true }
    private val client = OkHttpClient.Builder().build()
    private val streamClient = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private val baseUrls = AppConfig.BASE_URLS
    private val authHeader = "Bearer ${AppConfig.AUTH_TOKEN}"
    private const val PERSON_HEADER = "X-Notes-Person"

    class ApiHttpException(message: String) : Exception(message)

    private suspend fun <T> executeRequest(
        buildRequest: (String) -> Request,
        parse: (String) -> T
    ): T = withContext(Dispatchers.IO) {
        var lastError: IOException? = null
        for (baseUrl in baseUrls) {
            val request = buildRequest(baseUrl)
            try {
                client.newCall(request).execute().use { response ->
                    val body = response.body?.string().orEmpty()
                    if (!response.isSuccessful) {
                        throw ApiHttpException("HTTP ${response.code}: $body")
                    }
                    return@withContext parse(body)
                }
            } catch (exc: IOException) {
                lastError = exc
            }
        }
        throw (lastError ?: IOException("No reachable servers"))
    }

    private fun <T> executeStream(
        buildRequest: (String) -> Request,
        parseLine: (String) -> T
    ): Flow<T> = channelFlow {
        val job = launch(Dispatchers.IO) {
            var lastError: IOException? = null
            for (baseUrl in baseUrls) {
                val request = buildRequest(baseUrl)
                val call = streamClient.newCall(request)
                val response = try {
                    call.execute()
                } catch (exc: IOException) {
                    lastError = exc
                    continue
                }

                if (!response.isSuccessful) {
                    val body = response.body?.string().orEmpty()
                    response.close()
                    close(ApiHttpException("HTTP ${response.code}: $body"))
                    return@launch
                }

                val source: BufferedSource = response.body?.source()
                    ?: run {
                        response.close()
                        close(IOException("Empty response body"))
                        return@launch
                    }

                try {
                    while (!source.exhausted()) {
                        val line = source.readUtf8Line() ?: break
                        if (line.isBlank()) continue
                        val event = parseLine(line)
                        send(event)
                    }
                } catch (exc: Exception) {
                    close(exc)
                    response.close()
                    return@launch
                }

                response.close()
                close()
                return@launch
            }
            close(lastError ?: IOException("No reachable servers"))
        }

        awaitClose {
            job.cancel()
        }
    }

    private inline fun <reified T> decode(body: String): T =
        json.decodeFromString(body)

    private suspend inline fun <reified T> getJson(path: String): T =
        executeRequest(
            buildRequest = { baseUrl ->
                val builder = Request.Builder()
                    .url("$baseUrl$path")
                    .header("Authorization", authHeader)
                    .header("Accept", "application/json")
                val person = UserSettings.person
                if (person != null) {
                    builder.header(PERSON_HEADER, person)
                }
                builder.get().build()
            },
            parse = { body -> decode(body) }
        )

    private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()

    private suspend inline fun <reified T, reified R> postJson(path: String, payload: R): T =
        executeRequest(
            buildRequest = { baseUrl ->
                val jsonBody = json.encodeToString(payload).toRequestBody(JSON_MEDIA_TYPE)
                val builder = Request.Builder()
                    .url("$baseUrl$path")
                    .header("Authorization", authHeader)
                    .header("Accept", "application/json")
                    .header("Content-Type", "application/json")
                    .post(jsonBody)
                val person = UserSettings.person
                if (person != null) {
                    builder.header(PERSON_HEADER, person)
                }
                builder.build()
            },
            parse = { raw -> decode(raw) }
        )

    suspend fun fetchDaily(): DailyNote = getJson("/api/daily")

    suspend fun saveDaily(path: String, content: String): ApiMessage =
        postJson("/api/save", SaveDailyRequest(path = path, content = content))

    suspend fun appendDaily(path: String, text: String, pinned: Boolean): ApiMessage =
        postJson("/api/append", AppendDailyRequest(path = path, text = text, pinned = pinned))

    suspend fun addTodo(category: String, text: String = ""): ApiMessage =
        postJson("/api/todos/add", AddTodoRequest(category = category, text = text))

    suspend fun toggleTodo(path: String, line: Int): ApiMessage =
        postJson("/api/todos/toggle", ToggleTodoRequest(path = path, line = line))

    suspend fun clearPinned(path: String): ApiMessage =
        postJson("/api/clear-pinned", ClearPinnedRequest(path = path))

    suspend fun fetchSleepTimes(): SleepTimesResponse = getJson("/api/sleep-times")

    suspend fun appendSleepTimes(
        child: String,
        time: String,
        status: String
    ): ApiMessage =
        postJson("/api/sleep-times/append", AppendSleepRequest(child = child, time = time, status = status))

    suspend fun deleteSleepEntry(line: Int): ApiMessage =
        postJson("/api/sleep-times/delete", DeleteSleepRequest(line = line))

    suspend fun listFiles(path: String): FilesResponse =
        getJson("/api/files/list?path=${URLEncoder.encode(path, "UTF-8")}")

    suspend fun readFile(path: String): FileReadResponse =
        getJson("/api/files/read?path=${URLEncoder.encode(path, "UTF-8")}")

    suspend fun createFile(path: String): ApiMessage =
        postJson("/api/files/create", CreateFileRequest(path = path))

    suspend fun saveFile(path: String, content: String): ApiMessage =
        postJson("/api/files/save", SaveFileRequest(path = path, content = content))

    suspend fun deleteFile(path: String): ApiMessage =
        postJson("/api/files/delete", DeleteFileRequest(path = path))

    suspend fun unpinEntry(path: String, line: Int): ApiMessage =
        postJson("/api/files/unpin", UnpinEntryRequest(path = path, line = line))

    suspend fun claudeChat(message: String, sessionId: String?): ClaudeChatResponse =
        postJson("/api/claude/chat", ClaudeChatRequest(message = message, sessionId = sessionId))

    fun claudeChatStream(message: String, sessionId: String?): Flow<ClaudeStreamEvent> {
        val request = ClaudeChatRequest(message = message, sessionId = sessionId)
        return executeStream(
            buildRequest = { baseUrl ->
            val jsonBody = json.encodeToString(request).toRequestBody(JSON_MEDIA_TYPE)
            val builder = Request.Builder()
                .url("$baseUrl/api/claude/chat-stream")
                .header("Authorization", authHeader)
                .header("Accept", "application/x-ndjson")
                .header("Content-Type", "application/json")
                .post(jsonBody)
            val person = UserSettings.person
            if (person != null) {
                builder.header(PERSON_HEADER, person)
            }
            builder.build()
            },
            parseLine = { line -> json.decodeFromString<ClaudeStreamEvent>(line) }
        )
    }

    suspend fun claudeClear(sessionId: String): ApiMessage =
        postJson("/api/claude/clear", ClaudeClearRequest(sessionId = sessionId))

    fun agentChatStream(
        message: String,
        sessionId: String?,
        actionId: String? = null,
        confirm: Boolean = false
    ): Flow<AgentStreamEvent> {
        val request = AgentChatRequest(
            message = message,
            sessionId = sessionId,
            actionId = actionId,
            confirm = if (confirm) true else null
        )
        return executeStream(
            buildRequest = { baseUrl ->
                val jsonBody = json.encodeToString(request).toRequestBody(JSON_MEDIA_TYPE)
                val builder = Request.Builder()
                    .url("$baseUrl/api/agent/chat-stream")
                    .header("Authorization", authHeader)
                    .header("Accept", "application/x-ndjson")
                    .header("Content-Type", "application/json")
                    .post(jsonBody)
                val person = UserSettings.person
                if (person != null) {
                    builder.header(PERSON_HEADER, person)
                }
                builder.build()
            },
            parseLine = { line -> json.decodeFromString<AgentStreamEvent>(line) }
        )
    }

    suspend fun clearAgentSession(sessionId: String): ApiMessage =
        postJson("/api/agent/session/clear", ClaudeClearRequest(sessionId = sessionId))

    suspend fun fetchAgentConfig(): AgentConfig = getJson("/api/agent/config")

    suspend fun saveAgentConfig(runtimeMode: String, prompt: String): AgentConfig =
        postJson(
            "/api/agent/config",
            AgentConfigUpdateRequest(
                runtimeMode = runtimeMode,
                prompt = prompt
            )
        )

    suspend fun listAgentActions(): List<AgentAction> =
        getJson<AgentActionsResponse>("/api/agent/actions").actions

    suspend fun fetchEnv(): EnvResponse = getJson("/api/settings/env")

    suspend fun saveEnv(content: String): ApiMessage =
        postJson("/api/settings/env", SaveEnvRequest(content = content))
}
