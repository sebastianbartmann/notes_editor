package com.bartmann.noteseditor

private val dailyFilePattern = Regex("""^(\d{4}-\d{2}-\d{2})\.md$""")

object DailyNavigation {
    fun collectAvailableDailyPaths(entries: List<FileEntry>, todayDate: String): List<String> =
        entries
            .asSequence()
            .filter { !it.isDir }
            .mapNotNull { entry ->
                val match = dailyFilePattern.matchEntire(entry.name) ?: return@mapNotNull null
                val date = match.groupValues[1]
                if (date > todayDate) return@mapNotNull null
                "daily/${entry.name}"
            }
            .distinct()
            .sorted()
            .toList()

    fun dateFromPath(path: String): String? {
        val fileName = path.substringAfterLast('/')
        return dailyFilePattern.matchEntire(fileName)?.groupValues?.get(1)
    }

    fun previousPath(paths: List<String>, currentPath: String): String? {
        val index = paths.indexOf(currentPath)
        if (index <= 0) {
            return null
        }
        return paths[index - 1]
    }

    fun nextPath(paths: List<String>, currentPath: String): String? {
        val index = paths.indexOf(currentPath)
        if (index == -1 || index >= paths.lastIndex) {
            return null
        }
        return paths[index + 1]
    }
}
