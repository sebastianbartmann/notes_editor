package sleep

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ChildThomas = "Thomas"
	ChildFabian = "Fabian"

	StatusAsleep = "asleep"
	StatusAwake  = "awake"
)

var ErrNotFound = errors.New("sleep entry not found")

type Entry struct {
	ID         string
	Child      string
	Status     string
	OccurredAt *time.Time
	TimeText   string
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type NightSummary struct {
	NightDate      string
	Child          string
	DurationMinute int
	Bedtime        string
	WakeTime       string
}

type AverageSummary struct {
	Days            int
	Child           string
	AverageBedtime  string
	AverageWakeTime string
}

type Summary struct {
	Nights   []NightSummary
	Averages []AverageSummary
}

type Store struct {
	db       *sql.DB
	location *time.Location
}

func deriveTimeText(occurredAt *time.Time, loc *time.Location) string {
	if occurredAt == nil || loc == nil {
		return ""
	}
	return occurredAt.In(loc).Format("15:04")
}

func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errors.New("sleep db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{db: db, location: loc}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) init() error {
	stmts := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		`CREATE TABLE IF NOT EXISTS sleep_events (
			id TEXT PRIMARY KEY,
			child TEXT NOT NULL,
			status TEXT NOT NULL,
			occurred_at_utc TEXT,
			time_text TEXT NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			deleted_at_utc TEXT
		);`,
		"CREATE INDEX IF NOT EXISTS idx_sleep_events_active ON sleep_events(deleted_at_utc, child, occurred_at_utc);",
		"CREATE TABLE IF NOT EXISTS sleep_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);",
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func IsValidChild(child string) bool {
	return child == ChildThomas || child == ChildFabian
}

func NormalizeStatus(status string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case StatusAsleep, "eingeschlafen":
		return StatusAsleep, nil
	case StatusAwake, "aufgewacht":
		return StatusAwake, nil
	default:
		return "", fmt.Errorf("invalid status")
	}
}

func DisplayStatus(status string) string {
	if status == StatusAsleep {
		return "eingeschlafen"
	}
	return "aufgewacht"
}

func ParseOccurredAt(localDate, timeText string, loc *time.Location) (*time.Time, bool) {
	timeText = strings.TrimSpace(timeText)
	if timeText == "" {
		return nil, false
	}

	normalized := strings.ReplaceAll(timeText, ".", ":")
	for _, layout := range []string{"15:04", "15:04:05"} {
		t, err := time.ParseInLocation(layout, normalized, loc)
		if err != nil {
			continue
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", localDate, loc)
		if err != nil {
			return nil, false
		}
		combined := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
		utc := combined.UTC()
		return &utc, true
	}
	return nil, false
}

func ParseOccurredAtISO(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func (s *Store) ImportMarkdownIfNeeded(content string) (int, error) {
	var migrated string
	err := s.db.QueryRow("SELECT value FROM sleep_meta WHERE key='migrated_from_markdown'").Scan(&migrated)
	if err == nil && migrated == "1" {
		return 0, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	lines := strings.Split(content, "\n")
	now := time.Now().UTC().Format(time.RFC3339)
	count := 0

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " | ")
		if len(parts) < 4 {
			continue
		}
		date := strings.TrimSpace(parts[0])
		child := strings.TrimSpace(parts[1])
		timeText := strings.TrimSpace(parts[2])
		statusRaw := strings.TrimSpace(parts[3])

		if !IsValidChild(child) {
			continue
		}
		status, err := NormalizeStatus(statusRaw)
		if err != nil {
			continue
		}

		var occurredAt *time.Time
		notes := ""
		if parsed, ok := ParseOccurredAt(date, timeText, s.location); ok {
			occurredAt = parsed
		} else {
			notes = timeText
		}

		entryID := uuid.NewString()
		var occurredAny any
		if occurredAt != nil {
			occurredAny = occurredAt.Format(time.RFC3339)
		}

		if _, err := tx.Exec(
			`INSERT INTO sleep_events (id, child, status, occurred_at_utc, time_text, notes, created_at_utc, updated_at_utc, deleted_at_utc)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
			entryID, child, status, occurredAny, timeText, notes, now, now,
		); err != nil {
			return 0, err
		}
		count++
	}

	if _, err := tx.Exec(`INSERT INTO sleep_meta (key, value) VALUES ('migrated_from_markdown','1')
		ON CONFLICT(key) DO UPDATE SET value='1'`); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Store) ListEntries(limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := s.db.Query(`SELECT id, child, status, occurred_at_utc, time_text, notes, created_at_utc, updated_at_utc
		FROM sleep_events
		WHERE deleted_at_utc IS NULL
		ORDER BY COALESCE(occurred_at_utc, created_at_utc) DESC, created_at_utc DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		var e Entry
		var occurred sql.NullString
		var created string
		var updated string
		if err := rows.Scan(&e.ID, &e.Child, &e.Status, &occurred, &e.TimeText, &e.Notes, &created, &updated); err != nil {
			return nil, err
		}
		if occurred.Valid {
			parsed, err := time.Parse(time.RFC3339, occurred.String)
			if err == nil {
				e.OccurredAt = &parsed
			}
		}
		createdAt, _ := time.Parse(time.RFC3339, created)
		updatedAt, _ := time.Parse(time.RFC3339, updated)
		e.CreatedAt = createdAt
		e.UpdatedAt = updatedAt
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) CreateEntry(child, status string, occurredAt *time.Time, notes string) (Entry, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	var occurredAny any
	timeText := deriveTimeText(occurredAt, s.location)
	if occurredAt != nil {
		occurredAny = occurredAt.Format(time.RFC3339)
	}

	_, err := s.db.Exec(
		`INSERT INTO sleep_events (id, child, status, occurred_at_utc, time_text, notes, created_at_utc, updated_at_utc, deleted_at_utc)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		id, child, status, occurredAny, timeText, strings.TrimSpace(notes), now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return Entry{}, err
	}

	return Entry{
		ID:         id,
		Child:      child,
		Status:     status,
		OccurredAt: occurredAt,
		TimeText:   timeText,
		Notes:      strings.TrimSpace(notes),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (s *Store) UpdateEntry(id, child, status string, occurredAt *time.Time, notes string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var occurredAny any
	timeText := deriveTimeText(occurredAt, s.location)
	if occurredAt != nil {
		occurredAny = occurredAt.Format(time.RFC3339)
	}

	res, err := s.db.Exec(`UPDATE sleep_events
		SET child=?, status=?, occurred_at_utc=?, time_text=?, notes=?, updated_at_utc=?
		WHERE id=? AND deleted_at_utc IS NULL`,
		child, status, occurredAny, timeText, strings.TrimSpace(notes), now, id,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SoftDeleteEntry(id string) error {
	res, err := s.db.Exec(`UPDATE sleep_events SET deleted_at_utc=?, updated_at_utc=? WHERE id=? AND deleted_at_utc IS NULL`,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) BuildSummary() (Summary, error) {
	entries, err := s.ListEntries(2000)
	if err != nil {
		return Summary{}, err
	}

	type event struct {
		child      string
		status     string
		occurredAt time.Time
	}
	allEvents := make([]event, 0)
	for _, e := range entries {
		if e.OccurredAt == nil {
			continue
		}
		allEvents = append(allEvents, event{child: e.Child, status: e.Status, occurredAt: *e.OccurredAt})
	}
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].occurredAt.Before(allEvents[j].occurredAt)
	})

	type acc struct {
		nights []NightSummary
		bed7   []int
		wake7  []int
		bed30  []int
		wake30 []int
	}
	byChild := map[string]*acc{
		ChildThomas: {nights: make([]NightSummary, 0)},
		ChildFabian: {nights: make([]NightSummary, 0)},
	}

	lastAsleep := map[string]*time.Time{}
	nowLocal := time.Now().In(s.location)
	sevenCutoff := nowLocal.AddDate(0, 0, -7)
	thirtyCutoff := nowLocal.AddDate(0, 0, -30)

	for _, ev := range allEvents {
		if ev.status == StatusAsleep {
			t := ev.occurredAt
			lastAsleep[ev.child] = &t
			continue
		}
		start := lastAsleep[ev.child]
		if start == nil {
			continue
		}
		if ev.occurredAt.Before(*start) {
			continue
		}
		duration := ev.occurredAt.Sub(*start)
		if duration <= 0 || duration > 24*time.Hour {
			continue
		}
		startLocal := start.In(s.location)
		endLocal := ev.occurredAt.In(s.location)
		nightDate := startLocal.Format("2006-01-02")
		if startLocal.Hour() < 12 {
			nightDate = startLocal.AddDate(0, 0, -1).Format("2006-01-02")
		}

		minutes := int(duration / time.Minute)
		bedMin := startLocal.Hour()*60 + startLocal.Minute()
		wakeMin := endLocal.Hour()*60 + endLocal.Minute()
		row := NightSummary{
			NightDate:      nightDate,
			Child:          ev.child,
			DurationMinute: minutes,
			Bedtime:        startLocal.Format("15:04"),
			WakeTime:       endLocal.Format("15:04"),
		}
		bucket := byChild[ev.child]
		bucket.nights = append(bucket.nights, row)

		if !startLocal.Before(sevenCutoff) {
			bucket.bed7 = append(bucket.bed7, bedMin)
			bucket.wake7 = append(bucket.wake7, wakeMin)
		}
		if !startLocal.Before(thirtyCutoff) {
			bucket.bed30 = append(bucket.bed30, bedMin)
			bucket.wake30 = append(bucket.wake30, wakeMin)
		}
		lastAsleep[ev.child] = nil
	}

	nights := make([]NightSummary, 0)
	averages := make([]AverageSummary, 0)
	for _, child := range []string{ChildThomas, ChildFabian} {
		bucket := byChild[child]
		sort.Slice(bucket.nights, func(i, j int) bool {
			if bucket.nights[i].NightDate == bucket.nights[j].NightDate {
				return bucket.nights[i].Bedtime > bucket.nights[j].Bedtime
			}
			return bucket.nights[i].NightDate > bucket.nights[j].NightDate
		})
		nights = append(nights, bucket.nights...)

		if len(bucket.bed7) > 0 && len(bucket.wake7) > 0 {
			averages = append(averages, AverageSummary{
				Days:            7,
				Child:           child,
				AverageBedtime:  formatMeanMinutes(bucket.bed7),
				AverageWakeTime: formatMeanMinutes(bucket.wake7),
			})
		}
		if len(bucket.bed30) > 0 && len(bucket.wake30) > 0 {
			averages = append(averages, AverageSummary{
				Days:            30,
				Child:           child,
				AverageBedtime:  formatMeanMinutes(bucket.bed30),
				AverageWakeTime: formatMeanMinutes(bucket.wake30),
			})
		}
	}

	return Summary{Nights: nights, Averages: averages}, nil
}

func formatMeanMinutes(values []int) string {
	if len(values) == 0 {
		return ""
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	avg := int(float64(sum)/float64(len(values)) + 0.5)
	for avg < 0 {
		avg += 24 * 60
	}
	avg = avg % (24 * 60)
	h := avg / 60
	m := avg % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func (s *Store) ExportMarkdown() (string, error) {
	entries, err := s.ListEntries(5000)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# Sleep Times\n\n")
	b.WriteString("Generated from SQLite sleep database.\n\n")
	b.WriteString("| Date | Child | Time | Status | Notes |\n")
	b.WriteString("|---|---|---|---|---|\n")

	for _, e := range entries {
		date := ""
		timeText := e.TimeText
		if e.OccurredAt != nil {
			local := e.OccurredAt.In(s.location)
			date = local.Format("2006-01-02")
			if strings.TrimSpace(timeText) == "" {
				timeText = local.Format("15:04")
			}
		}
		status := DisplayStatus(e.Status)
		notes := strings.ReplaceAll(e.Notes, "|", "\\|")
		timeSafe := strings.ReplaceAll(timeText, "|", "\\|")
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", date, e.Child, timeSafe, status, notes))
	}

	return b.String(), nil
}
