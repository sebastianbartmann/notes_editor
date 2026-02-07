package com.bartmann.noteseditor

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DailyNavigationTest {
    @Test
    fun collectAvailableDailyPaths_filtersNonDailyAndFutureDates() {
        val entries = listOf(
            FileEntry(name = "2026-02-04.md", path = "daily/2026-02-04.md", isDir = false),
            FileEntry(name = "2026-02-06.md", path = "daily/2026-02-06.md", isDir = false),
            FileEntry(name = "2026-02-08.md", path = "daily/2026-02-08.md", isDir = false),
            FileEntry(name = "notes.md", path = "daily/notes.md", isDir = false),
            FileEntry(name = "archive", path = "daily/archive", isDir = true),
        )

        val paths = DailyNavigation.collectAvailableDailyPaths(entries, todayDate = "2026-02-07")

        assertEquals(
            listOf("daily/2026-02-04.md", "daily/2026-02-06.md"),
            paths
        )
    }

    @Test
    fun previousAndNextPath_followSortedListAndStopAtEdges() {
        val paths = listOf(
            "daily/2026-02-01.md",
            "daily/2026-02-03.md",
            "daily/2026-02-07.md",
        )

        assertEquals("daily/2026-02-01.md", DailyNavigation.previousPath(paths, "daily/2026-02-03.md"))
        assertEquals("daily/2026-02-07.md", DailyNavigation.nextPath(paths, "daily/2026-02-03.md"))
        assertNull(DailyNavigation.previousPath(paths, "daily/2026-02-01.md"))
        assertNull(DailyNavigation.nextPath(paths, "daily/2026-02-07.md"))
    }

    @Test
    fun dateFromPath_extractsDailyDate() {
        assertEquals("2026-02-07", DailyNavigation.dateFromPath("daily/2026-02-07.md"))
        assertNull(DailyNavigation.dateFromPath("daily/notes.md"))
        assertNull(DailyNavigation.dateFromPath("2026-02-07.txt"))
    }
}
