package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Daily handles daily note operations.
type Daily struct {
	store *Store
}

// NewDaily creates a new Daily instance.
func NewDaily(store *Store) *Daily {
	return &Daily{store: store}
}

// GetOrCreateDaily returns today's daily note, creating it if it doesn't exist.
// The note inherits incomplete todos and pinned entries from the previous note.
func (d *Daily) GetOrCreateDaily(person string, date time.Time) (string, string, error) {
	filename := date.Format("2006-01-02") + ".md"
	path := filepath.Join("daily", filename)

	// Check if today's note exists
	content, err := d.store.ReadFile(person, path)
	if err == nil {
		return content, path, nil
	}

	if !os.IsNotExist(err) {
		return "", "", err
	}

	// Create new daily note with inheritance
	content, err = d.generateDailyNote(person, date)
	if err != nil {
		return "", "", err
	}

	if err := d.store.WriteFile(person, path, content); err != nil {
		return "", "", err
	}

	return content, path, nil
}

// generateDailyNote creates a new daily note with inherited todos and pinned entries.
func (d *Daily) generateDailyNote(person string, date time.Time) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString(fmt.Sprintf("# %s\n\n", date.Format("2006-01-02")))

	// Todos section with inheritance
	builder.WriteString("## todos\n\n")

	prevContent, err := d.findPreviousNote(person, date)
	if err == nil && prevContent != "" {
		todos := d.extractIncompleteTodos(prevContent)
		if todos != "" {
			builder.WriteString(todos)
			builder.WriteString("\n")
		}
	}

	// Custom notes section with inherited pinned entries
	builder.WriteString("## custom notes\n\n")

	if prevContent != "" {
		pinned := d.extractPinnedNotes(prevContent)
		if pinned != "" {
			builder.WriteString(pinned)
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// findPreviousNote finds the most recent daily note before the given date.
func (d *Daily) findPreviousNote(person string, date time.Time) (string, error) {
	dailyPath := filepath.Join(d.store.rootPath, person, "daily")

	entries, err := os.ReadDir(dailyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	// Collect and sort date filenames
	var dates []string
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)
	targetDate := date.Format("2006-01-02")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if datePattern.MatchString(name) {
			dateStr := strings.TrimSuffix(name, ".md")
			if dateStr < targetDate {
				dates = append(dates, dateStr)
			}
		}
	}

	if len(dates) == 0 {
		return "", nil
	}

	// Sort descending to get most recent first
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Read the most recent note
	prevPath := filepath.Join("daily", dates[0]+".md")
	return d.store.ReadFile(person, prevPath)
}

// extractIncompleteTodos extracts incomplete todos from a daily note's ## todos section.
func (d *Daily) extractIncompleteTodos(content string) string {
	// Find the ## todos section
	todosStart := strings.Index(content, "## todos")
	if todosStart == -1 {
		return ""
	}

	// Find where the section ends (next ## or end of content)
	sectionContent := content[todosStart+len("## todos"):]
	nextSection := strings.Index(sectionContent, "\n## ")
	if nextSection != -1 {
		sectionContent = sectionContent[:nextSection]
	}

	// Extract lines that are incomplete todos (- [ ]) or category headers (###)
	var result []string
	completedPattern := regexp.MustCompile(`^\s*-\s*\[[xX]\]`)
	todoPattern := regexp.MustCompile(`^\s*-\s*\[ \]`)
	headerPattern := regexp.MustCompile(`^###\s+`)

	lines := strings.Split(sectionContent, "\n")
	inCategory := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if headerPattern.MatchString(line) {
			// Category header - include it
			result = append(result, line)
			inCategory = true
			continue
		}

		if completedPattern.MatchString(line) {
			// Skip completed todos
			continue
		}

		if todoPattern.MatchString(line) {
			// Include incomplete todos
			result = append(result, line)
			inCategory = true
			continue
		}

		// Include non-todo lines if we're in a category context
		if inCategory && strings.HasPrefix(trimmed, "-") {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, "\n")
}

// extractPinnedNotes extracts pinned entries from a daily note's ## custom notes section.
func (d *Daily) extractPinnedNotes(content string) string {
	// Find the ## custom notes section
	customStart := strings.Index(content, "## custom notes")
	if customStart == -1 {
		return ""
	}

	sectionContent := content[customStart+len("## custom notes"):]
	nextSection := strings.Index(sectionContent, "\n## ")
	if nextSection != -1 {
		sectionContent = sectionContent[:nextSection]
	}

	// Find pinned entries (### HH:MM <pinned> and their content until next ###)
	pinnedPattern := regexp.MustCompile(`^###\s+\d{2}:\d{2}\s*<pinned>`)
	headerPattern := regexp.MustCompile(`^###\s+`)

	var result []string
	var currentEntry []string
	inPinned := false

	lines := strings.Split(sectionContent, "\n")
	for _, line := range lines {
		if headerPattern.MatchString(line) {
			// Save previous pinned entry if any
			if inPinned && len(currentEntry) > 0 {
				result = append(result, strings.Join(currentEntry, "\n"))
			}

			// Check if this is a pinned entry
			if pinnedPattern.MatchString(line) {
				currentEntry = []string{line}
				inPinned = true
			} else {
				currentEntry = nil
				inPinned = false
			}
			continue
		}

		if inPinned {
			currentEntry = append(currentEntry, line)
		}
	}

	// Don't forget the last entry
	if inPinned && len(currentEntry) > 0 {
		result = append(result, strings.Join(currentEntry, "\n"))
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, "\n")
}

// AddTask adds a task to a specific category in the daily note.
func (d *Daily) AddTask(person, path, category, task string) error {
	content, err := d.store.ReadFile(person, path)
	if err != nil {
		return err
	}

	// Find the category header (### work or ### priv)
	categoryHeader := "### " + category
	idx := strings.Index(content, categoryHeader)

	var newContent string
	if idx == -1 {
		// Category doesn't exist, add it under ## todos
		todosIdx := strings.Index(content, "## todos")
		if todosIdx == -1 {
			return fmt.Errorf("todos section not found")
		}

		// Find end of todos section header
		insertIdx := todosIdx + len("## todos")
		newContent = content[:insertIdx] + "\n\n" + categoryHeader + "\n- [ ] " + task + content[insertIdx:]
	} else {
		// Find end of category header line
		lineEnd := strings.Index(content[idx:], "\n")
		if lineEnd == -1 {
			lineEnd = len(content) - idx
		}
		insertIdx := idx + lineEnd

		newContent = content[:insertIdx] + "\n- [ ] " + task + content[insertIdx:]
	}

	return d.store.WriteFile(person, path, newContent)
}

// ToggleTask toggles a task's completion status at the given line number (1-indexed).
func (d *Daily) ToggleTask(person, path string, lineNum int) error {
	content, err := d.store.ReadFile(person, path)
	if err != nil {
		return err
	}

	lines := strings.Split(content, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return fmt.Errorf("line number %d out of range", lineNum)
	}

	line := lines[lineNum-1]
	uncheckedPattern := regexp.MustCompile(`^(\s*-\s*)\[ \](.*)$`)
	checkedPattern := regexp.MustCompile(`^(\s*-\s*)\[[xX]\](.*)$`)

	if uncheckedPattern.MatchString(line) {
		lines[lineNum-1] = uncheckedPattern.ReplaceAllString(line, "${1}[x]${2}")
	} else if checkedPattern.MatchString(line) {
		lines[lineNum-1] = checkedPattern.ReplaceAllString(line, "${1}[ ]${2}")
	} else {
		return fmt.Errorf("line %d is not a task", lineNum)
	}

	return d.store.WriteFile(person, path, strings.Join(lines, "\n"))
}

// ClearAllPinned removes all <pinned> markers from a note.
func (d *Daily) ClearAllPinned(person, path string) error {
	content, err := d.store.ReadFile(person, path)
	if err != nil {
		return err
	}

	// Remove <pinned> markers from ### HH:MM <pinned> lines
	// Use (?m) multiline flag so ^ matches start of each line
	pattern := regexp.MustCompile(`(?m)(^###\s+\d{2}:\d{2})\s*<pinned>`)
	newContent := pattern.ReplaceAllString(content, "$1")

	return d.store.WriteFile(person, path, newContent)
}

// UnpinEntry removes the <pinned> marker from a specific line (1-indexed).
func (d *Daily) UnpinEntry(person, path string, lineNum int) error {
	content, err := d.store.ReadFile(person, path)
	if err != nil {
		return err
	}

	lines := strings.Split(content, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return fmt.Errorf("line number %d out of range", lineNum)
	}

	line := lines[lineNum-1]
	pattern := regexp.MustCompile(`(^###\s+\d{2}:\d{2})\s*<pinned>`)
	if !pattern.MatchString(line) {
		return fmt.Errorf("line %d is not a pinned entry", lineNum)
	}

	lines[lineNum-1] = pattern.ReplaceAllString(line, "$1")
	return d.store.WriteFile(person, path, strings.Join(lines, "\n"))
}

// AppendEntry appends a timestamped entry to the custom notes section.
func (d *Daily) AppendEntry(person, path, text string, pinned bool) error {
	content, err := d.store.ReadFile(person, path)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("15:04")
	header := "### " + timestamp
	if pinned {
		header += " <pinned>"
	}

	entry := "\n" + header + "\n" + text + "\n"

	// Append to end of file
	newContent := strings.TrimRight(content, "\n") + entry

	return d.store.WriteFile(person, path, newContent)
}
