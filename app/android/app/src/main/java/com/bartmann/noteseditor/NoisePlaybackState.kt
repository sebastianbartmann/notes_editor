package com.bartmann.noteseditor

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

object NoisePlaybackState {
    var isPlaying by mutableStateOf(false)
        internal set
}
