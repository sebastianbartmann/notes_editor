package linkedin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogPost logs a LinkedIn post to the person's activity log.
func (s *Service) LogPost(person, postURN, text, response string) error {
	return s.logActivity(person, "post", postURN, "", text, response)
}

// LogComment logs a LinkedIn comment to the person's activity log.
func (s *Service) LogComment(person, action, postURN, commentURN, text, response string) error {
	return s.logActivity(person, action, postURN, commentURN, text, response)
}

// logActivity appends an activity record to the CSV log file.
// CSV format: timestamp, action, post_urn, comment_urn, text, response
func (s *Service) logActivity(person, action, postURN, commentURN, text, response string) error {
	// Build log file path
	logDir := filepath.Join(s.vaultRoot, person, "linkedin")
	logPath := filepath.Join(logDir, "posts.csv")

	// Create directory if needed
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Check if file exists to determine if we need to write header
	writeHeader := false
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		writeHeader = true
	}

	// Open file for append
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header if new file
	if writeHeader {
		if err := writer.Write([]string{"timestamp", "action", "post_urn", "comment_urn", "text", "response"}); err != nil {
			return err
		}
	}

	// Compact the response JSON
	compactResponse := compactJSON(response)

	// Escape newlines in text
	escapedText := strings.ReplaceAll(text, "\n", "\\n")

	// Write record
	record := []string{
		time.Now().Format(time.RFC3339),
		action,
		postURN,
		commentURN,
		escapedText,
		compactResponse,
	}

	return writer.Write(record)
}

// compactJSON compacts a JSON string by removing unnecessary whitespace.
func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return s // Return original if compaction fails
	}
	return buf.String()
}
