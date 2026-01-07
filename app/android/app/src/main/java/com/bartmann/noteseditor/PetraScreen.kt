package com.bartmann.noteseditor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun PetraScreen(modifier: Modifier, padding: androidx.compose.foundation.layout.PaddingValues) {
    var content by remember { mutableStateOf("") }
    var appendText by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun refresh() {
        scope.launch {
            try {
                val petra = ApiClient.fetchPetra()
                content = petra.content
                message = "Loaded."
            } catch (exc: Exception) {
                message = "Failed to load: ${exc.message}"
            }
        }
    }

    LaunchedEffect(Unit) {
        refresh()
    }

    Column(
        modifier = modifier
            .padding(padding)
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        ScreenTitle(text = "Petra Notes")
        Panel {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                CompactButton(text = "Refresh") { refresh() }
                CompactButton(text = "Save") {
                    scope.launch {
                        try {
                            val response = ApiClient.savePetra(content)
                            message = response.message
                        } catch (exc: Exception) {
                            message = "Save failed: ${exc.message}"
                        }
                    }
                }
            }
            CompactDivider()
            SectionTitle(text = "Current note")
            CompactOutlinedTextField(
                value = content,
                onValueChange = { content = it },
                label = "Petra note",
                modifier = Modifier.fillMaxWidth(),
                minLines = 7
            )
            CompactDivider()
            SectionTitle(text = "Quick append")
            CompactOutlinedTextField(
                value = appendText,
                onValueChange = { appendText = it },
                label = "Append text",
                modifier = Modifier.fillMaxWidth(),
                minLines = 3
            )
            CompactButton(text = "Append") {
                scope.launch {
                    try {
                        val response = ApiClient.appendPetra(appendText)
                        message = response.message
                        appendText = ""
                        refresh()
                    } catch (exc: Exception) {
                        message = "Append failed: ${exc.message}"
                    }
                }
            }
            if (message.isNotBlank()) {
                CompactDivider()
                Text(text = message, color = MaterialTheme.colorScheme.secondary)
            }
        }
    }
}
