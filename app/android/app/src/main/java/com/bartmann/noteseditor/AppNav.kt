package com.bartmann.noteseditor

object AppNav {
    @Volatile
    var openSync: (() -> Unit)? = null
}

