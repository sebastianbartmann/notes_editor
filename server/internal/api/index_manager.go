package api

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultQMDIndexCommand = "/home/dev/.bun/bin/qmd"
)

type IndexStatus struct {
	InProgress    bool       `json:"in_progress"`
	Pending       bool       `json:"pending"`
	LastReason    string     `json:"last_reason,omitempty"`
	LastStartedAt *time.Time `json:"last_started_at,omitempty"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastErrorAt   *time.Time `json:"last_error_at,omitempty"`
}

// IndexManager serializes qmd indexing jobs and coalesces bursts.
// It is intentionally strict: indexing errors are preserved and logged loudly.
type IndexManager struct {
	cond *sync.Cond

	mu         sync.Mutex
	started    bool
	stopping   bool
	inProgress bool
	pending    bool
	reason     string

	lastReason    string
	lastStartedAt time.Time
	lastSuccessAt time.Time
	lastError     string
	lastErrorAt   time.Time

	debounce time.Duration
	command  string
	rootPath string
	persons  []string
}

func NewIndexManager(rootPath string, persons []string) *IndexManager {
	uniq := make(map[string]struct{}, len(persons))
	filtered := make([]string, 0, len(persons))
	for _, p := range persons {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := uniq[p]; ok {
			continue
		}
		uniq[p] = struct{}{}
		filtered = append(filtered, p)
	}

	im := &IndexManager{
		debounce: 2 * time.Second,
		command:  defaultQMDIndexCommand,
		rootPath: rootPath,
		persons:  filtered,
	}
	im.cond = sync.NewCond(&im.mu)
	return im
}

func (m *IndexManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return
	}
	m.started = true
	go m.loop()
}

func (m *IndexManager) Stop() {
	m.mu.Lock()
	m.stopping = true
	m.cond.Broadcast()
	m.mu.Unlock()
}

func (m *IndexManager) TriggerReindex(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pending = true
	if strings.TrimSpace(reason) != "" {
		m.reason = strings.TrimSpace(reason)
	}
	m.cond.Signal()
}

func (m *IndexManager) Status() IndexStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	var startedAt *time.Time
	if !m.lastStartedAt.IsZero() {
		t := m.lastStartedAt
		startedAt = &t
	}
	var successAt *time.Time
	if !m.lastSuccessAt.IsZero() {
		t := m.lastSuccessAt
		successAt = &t
	}
	var errAt *time.Time
	if !m.lastErrorAt.IsZero() {
		t := m.lastErrorAt
		errAt = &t
	}

	return IndexStatus{
		InProgress:    m.inProgress,
		Pending:       m.pending,
		LastReason:    m.lastReason,
		LastStartedAt: startedAt,
		LastSuccessAt: successAt,
		LastError:     m.lastError,
		LastErrorAt:   errAt,
	}
}

func (m *IndexManager) loop() {
	for {
		m.mu.Lock()
		for !m.stopping && !m.pending {
			m.cond.Wait()
		}
		if m.stopping {
			m.mu.Unlock()
			return
		}
		debounce := m.debounce
		m.mu.Unlock()

		time.Sleep(debounce)

		m.mu.Lock()
		if m.stopping {
			m.mu.Unlock()
			return
		}
		reason := m.reason
		m.reason = ""
		m.pending = false
		m.inProgress = true
		m.lastReason = reason
		m.lastStartedAt = time.Now()
		m.mu.Unlock()

		if reason == "" {
			reason = "unspecified"
		}
		log.Printf("qmd index: start (reason=%s)", reason)
		err := m.runIndex()

		m.mu.Lock()
		if err != nil {
			m.lastError = err.Error()
			m.lastErrorAt = time.Now()
			log.Printf("qmd index: failed (reason=%s): %v", reason, err)
		} else {
			m.lastSuccessAt = time.Now()
			m.lastError = ""
			m.lastErrorAt = time.Time{}
			log.Printf("qmd index: success (reason=%s)", reason)
		}
		m.inProgress = false
		m.cond.Broadcast()
		m.mu.Unlock()
	}
}

func (m *IndexManager) runIndex() error {
	if err := m.ensureCollections(); err != nil {
		return err
	}

	// Update collection metadata/file set first, then compute embeddings.
	if err := m.runQMD("update", 15*time.Minute); err != nil {
		return err
	}
	if err := m.runQMD("embed", 45*time.Minute); err != nil {
		return err
	}
	return nil
}

func (m *IndexManager) ensureCollections() error {
	if strings.TrimSpace(m.rootPath) == "" {
		return fmt.Errorf("notes root path is empty")
	}

	existing, err := m.listCollections()
	if err != nil {
		return err
	}
	target, err := m.targetPersons()
	if err != nil {
		return err
	}

	for _, person := range target {
		if _, ok := existing[person]; ok {
			continue
		}
		personPath := filepath.Join(m.rootPath, person)
		log.Printf("qmd index: creating collection for person=%s path=%s", person, personPath)
		if err := m.runQMDWithArgs(20*time.Minute, "collection", "add", personPath, "--name", person); err != nil {
			return err
		}
	}
	return nil
}

func (m *IndexManager) targetPersons() ([]string, error) {
	if len(m.persons) > 0 {
		return append([]string(nil), m.persons...), nil
	}

	entries, err := os.ReadDir(m.rootPath)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") || !entry.IsDir() {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (m *IndexManager) listCollections() (map[string]struct{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.command, "collection", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = strings.TrimSpace(stdout.String())
		}
		return nil, fmt.Errorf("qmd collection list failed: %w: %s", err, errText)
	}

	collections := map[string]struct{}{}
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "(qmd://") {
			continue
		}
		name := strings.SplitN(line, " ", 2)[0]
		name = strings.TrimSpace(name)
		if name != "" {
			collections[name] = struct{}{}
		}
	}
	return collections, nil
}

func (m *IndexManager) runQMD(subcommand string, timeout time.Duration) error {
	return m.runQMDWithArgs(timeout, subcommand)
}

func (m *IndexManager) runQMDWithArgs(timeout time.Duration, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("no qmd args provided")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, m.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	subcommand := args[0]
	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		outText := strings.TrimSpace(stdout.String())
		switch {
		case errText != "":
			return &IndexCommandError{Subcommand: subcommand, Err: err, Output: errText}
		case outText != "":
			return &IndexCommandError{Subcommand: subcommand, Err: err, Output: outText}
		default:
			return &IndexCommandError{Subcommand: subcommand, Err: err}
		}
	}
	return nil
}

type IndexCommandError struct {
	Subcommand string
	Err        error
	Output     string
}

func (e *IndexCommandError) Error() string {
	if e == nil {
		return ""
	}
	if e.Output != "" {
		return "qmd " + e.Subcommand + " failed: " + e.Err.Error() + ": " + e.Output
	}
	return "qmd " + e.Subcommand + " failed: " + e.Err.Error()
}
