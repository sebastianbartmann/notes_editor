package linkedin

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"notes-editor/internal/config"
)

func newTestService(vaultRoot string) *Service {
	return &Service{
		config:    &config.LinkedInConfig{},
		vaultRoot: vaultRoot,
	}
}

func TestLogPost(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	err := svc.LogPost("sebastian", "urn:li:share:123", "Test post text", `{"id":"123"}`)
	if err != nil {
		t.Fatalf("LogPost failed: %v", err)
	}

	// Verify file was created
	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("CSV log file was not created")
	}

	// Read and verify content
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records (header + data), got %d", len(records))
	}

	// Check header
	expectedHeader := []string{"timestamp", "action", "post_urn", "comment_urn", "text", "response"}
	for i, h := range expectedHeader {
		if records[0][i] != h {
			t.Errorf("Header[%d]: expected %q, got %q", i, h, records[0][i])
		}
	}

	// Check data row
	data := records[1]
	if data[1] != "post" {
		t.Errorf("Expected action 'post', got %q", data[1])
	}
	if data[2] != "urn:li:share:123" {
		t.Errorf("Expected post_urn 'urn:li:share:123', got %q", data[2])
	}
	if data[3] != "" {
		t.Errorf("Expected empty comment_urn, got %q", data[3])
	}
	if data[4] != "Test post text" {
		t.Errorf("Expected text 'Test post text', got %q", data[4])
	}
	if data[5] != `{"id":"123"}` {
		t.Errorf("Expected response '{\"id\":\"123\"}', got %q", data[5])
	}
}

func TestLogComment(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	err := svc.LogComment("petra", "comment", "urn:li:share:456", "urn:li:comment:789", "My comment", `{"$URN":"urn:li:comment:789"}`)
	if err != nil {
		t.Fatalf("LogComment failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "petra", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	data := records[1]
	if data[1] != "comment" {
		t.Errorf("Expected action 'comment', got %q", data[1])
	}
	if data[2] != "urn:li:share:456" {
		t.Errorf("Expected post_urn 'urn:li:share:456', got %q", data[2])
	}
	if data[3] != "urn:li:comment:789" {
		t.Errorf("Expected comment_urn, got %q", data[3])
	}
}

func TestLogComment_Reply(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	err := svc.LogComment("sebastian", "reply", "urn:li:share:111", "urn:li:comment:222", "My reply", `{}`)
	if err != nil {
		t.Fatalf("LogComment reply failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	data := records[1]
	if data[1] != "reply" {
		t.Errorf("Expected action 'reply', got %q", data[1])
	}
}

func TestLogActivity_AppendMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	// Log multiple activities
	if err := svc.LogPost("sebastian", "urn:1", "Post 1", `{}`); err != nil {
		t.Fatalf("First LogPost failed: %v", err)
	}
	if err := svc.LogPost("sebastian", "urn:2", "Post 2", `{}`); err != nil {
		t.Fatalf("Second LogPost failed: %v", err)
	}
	if err := svc.LogComment("sebastian", "comment", "urn:1", "urn:c1", "Comment 1", `{}`); err != nil {
		t.Fatalf("LogComment failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// 1 header + 3 data rows
	if len(records) != 4 {
		t.Fatalf("Expected 4 records, got %d", len(records))
	}

	// Verify header appears only once (first row)
	if records[0][0] != "timestamp" {
		t.Errorf("First row should be header, got %q", records[0][0])
	}
	// Data rows should not start with "timestamp"
	for i := 1; i < len(records); i++ {
		if records[i][0] == "timestamp" {
			t.Errorf("Row %d should not be header", i)
		}
	}
}

func TestLogActivity_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	// Directory does not exist initially
	linkedinDir := filepath.Join(tmpDir, "newperson", "linkedin")
	if _, err := os.Stat(linkedinDir); !os.IsNotExist(err) {
		t.Fatal("linkedin directory should not exist initially")
	}

	// Log should create the directory
	if err := svc.LogPost("newperson", "urn:x", "text", `{}`); err != nil {
		t.Fatalf("LogPost failed: %v", err)
	}

	if _, err := os.Stat(linkedinDir); os.IsNotExist(err) {
		t.Fatal("linkedin directory should be created")
	}
}

func TestLogActivity_EscapesNewlines(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	textWithNewlines := "Line 1\nLine 2\nLine 3"
	if err := svc.LogPost("sebastian", "urn:x", textWithNewlines, `{}`); err != nil {
		t.Fatalf("LogPost failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// Text should have escaped newlines
	escapedText := records[1][4]
	if strings.Contains(escapedText, "\n") {
		t.Errorf("Text should not contain literal newlines: %q", escapedText)
	}
	if escapedText != `Line 1\nLine 2\nLine 3` {
		t.Errorf("Text should have escaped newlines: %q", escapedText)
	}
}

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already compact",
			input:    `{"id":"123"}`,
			expected: `{"id":"123"}`,
		},
		{
			name:     "with whitespace",
			input:    `{ "id" : "123" }`,
			expected: `{"id":"123"}`,
		},
		{
			name:     "multiline",
			input:    "{\n  \"id\": \"123\",\n  \"name\": \"test\"\n}",
			expected: `{"id":"123","name":"test"}`,
		},
		{
			name:     "invalid JSON returns original",
			input:    `not json`,
			expected: `not json`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "nested object",
			input:    "{\n  \"outer\": {\n    \"inner\": 42\n  }\n}",
			expected: `{"outer":{"inner":42}}`,
		},
		{
			name:     "array",
			input:    "[ 1, 2, 3 ]",
			expected: `[1,2,3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compactJSON(tt.input)
			if result != tt.expected {
				t.Errorf("compactJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogActivity_TimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	if err := svc.LogPost("sebastian", "urn:x", "text", `{}`); err != nil {
		t.Fatalf("LogPost failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	timestamp := records[1][0]
	// RFC3339 format: 2006-01-02T15:04:05Z07:00
	if !strings.Contains(timestamp, "T") {
		t.Errorf("Timestamp should be RFC3339 format, got %q", timestamp)
	}
	if len(timestamp) < 20 {
		t.Errorf("Timestamp too short for RFC3339, got %q", timestamp)
	}
}

func TestLogActivity_IsolatesPersons(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	// Log for different persons
	if err := svc.LogPost("sebastian", "urn:seb", "Sebastian's post", `{}`); err != nil {
		t.Fatalf("LogPost for sebastian failed: %v", err)
	}
	if err := svc.LogPost("petra", "urn:pet", "Petra's post", `{}`); err != nil {
		t.Fatalf("LogPost for petra failed: %v", err)
	}

	// Verify separate files
	sebPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	petPath := filepath.Join(tmpDir, "petra", "linkedin", "posts.csv")

	for _, path := range []string{sebPath, petPath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Log file should exist: %s", path)
		}
	}

	// Verify content isolation
	sebFile, _ := os.Open(sebPath)
	defer sebFile.Close()
	sebRecords, _ := csv.NewReader(sebFile).ReadAll()

	petFile, _ := os.Open(petPath)
	defer petFile.Close()
	petRecords, _ := csv.NewReader(petFile).ReadAll()

	if len(sebRecords) != 2 || len(petRecords) != 2 {
		t.Error("Each person should have exactly 1 data row")
	}

	if sebRecords[1][4] != "Sebastian's post" {
		t.Errorf("Sebastian's file has wrong content: %q", sebRecords[1][4])
	}
	if petRecords[1][4] != "Petra's post" {
		t.Errorf("Petra's file has wrong content: %q", petRecords[1][4])
	}
}

func TestLogActivity_HandlesSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)

	// Text with quotes, commas, and other CSV-problematic characters
	specialText := `He said "Hello, World!" and it's great`
	if err := svc.LogPost("sebastian", "urn:x", specialText, `{"msg":"quoted \"value\""}`); err != nil {
		t.Fatalf("LogPost failed: %v", err)
	}

	logPath := filepath.Join(tmpDir, "sebastian", "linkedin", "posts.csv")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV with special characters: %v", err)
	}

	// CSV reader should properly handle quoted fields
	if records[1][4] != specialText {
		t.Errorf("Special characters not preserved: got %q, want %q", records[1][4], specialText)
	}
}
