package com.bartmann.noteseditor

import java.io.IOException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.FormBody
import okhttp3.OkHttpClient
import okhttp3.Request

object ApiClient {
    private val json = Json { ignoreUnknownKeys = true }
    private val client = OkHttpClient.Builder().build()

    private val baseUrls = AppConfig.BASE_URLS
    private val authHeader = "Bearer ${AppConfig.AUTH_TOKEN}"
    private const val PERSON_HEADER = "X-Notes-Person"

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
                        throw IOException("HTTP ${response.code}: $body")
                    }
                    return@withContext parse(body)
                }
            } catch (exc: IOException) {
                lastError = exc
            }
        }
        throw (lastError ?: IOException("No reachable servers"))
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

    private suspend inline fun <reified T> postForm(path: String, params: Map<String, String>): T =
        executeRequest(
            buildRequest = { baseUrl ->
                val formBuilder = FormBody.Builder()
                for ((key, value) in params) {
                    formBuilder.add(key, value)
                }
                val builder = Request.Builder()
                    .url("$baseUrl$path")
                    .header("Authorization", authHeader)
                    .header("Accept", "application/json")
                    .post(formBuilder.build())
                val person = UserSettings.person
                if (person != null) {
                    builder.header(PERSON_HEADER, person)
                }
                builder.build()
            },
            parse = { body -> decode(body) }
        )

    suspend fun fetchDaily(): DailyNote = getJson("/api/daily")
    suspend fun saveDaily(content: String): ApiMessage =
        postForm("/api/save", mapOf("content" to content))

    suspend fun appendDaily(content: String, pinned: Boolean): ApiMessage =
        postForm(
            "/api/append",
            mapOf("content" to content, "pinned" to if (pinned) "on" else "")
        )

    suspend fun addTodo(category: String): ApiMessage =
        postForm("/api/todos/add", mapOf("category" to category))

    suspend fun toggleTodo(path: String, line: Int): ApiMessage =
        postForm("/api/todos/toggle", mapOf("path" to path, "line" to line.toString()))

    suspend fun clearPinned(): ApiMessage =
        postForm("/api/clear-pinned", emptyMap())


    suspend fun fetchSleepTimes(): SleepTimesResponse = getJson("/api/sleep-times")

    suspend fun appendSleepTimes(
        child: String,
        entry: String,
        asleep: Boolean,
        woke: Boolean
    ): ApiMessage {
        val params = mutableMapOf(
            "child" to child,
            "entry" to entry
        )
        if (asleep) params["asleep"] = "on"
        if (woke) params["woke"] = "on"
        return postForm("/api/sleep-times/append", params)
    }

    suspend fun deleteSleepEntry(line: Int): ApiMessage =
        postForm("/api/sleep-times/delete", mapOf("line" to line.toString()))

    suspend fun listFiles(path: String): FilesResponse =
        getJson("/api/files/list?path=$path")

    suspend fun readFile(path: String): FileReadResponse =
        getJson("/api/files/read?path=$path")

    suspend fun createFile(path: String): ApiMessage =
        postForm("/api/files/create", mapOf("path" to path))

    suspend fun saveFile(path: String, content: String): ApiMessage =
        postForm("/api/files/save-json", mapOf("path" to path, "content" to content))

    suspend fun deleteFile(path: String): ApiMessage =
        postForm("/api/files/delete-json", mapOf("path" to path))

    suspend fun unpinEntry(path: String, line: Int): ApiMessage =
        postForm("/api/files/unpin", mapOf("path" to path, "line" to line.toString()))

    suspend fun runClaude(prompt: String): ClaudeResponse =
        postForm("/api/tools/claude-json", mapOf("prompt" to prompt))
}
