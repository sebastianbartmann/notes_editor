package api

import (
	"sync"
	"time"

	"notes-editor/internal/vault"
)

type SyncStatus struct {
	InProgress  bool       `json:"in_progress"`
	PendingPull bool       `json:"pending_pull"`
	PendingPush bool       `json:"pending_push"`
	LastPullAt  *time.Time `json:"last_pull_at,omitempty"`
	LastPushAt  *time.Time `json:"last_push_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	LastErrorAt *time.Time `json:"last_error_at,omitempty"`
}

// SyncManager serializes git operations and coalesces frequent triggers.
// It also uses a shared vault lock to avoid concurrent file operations while git mutates the working tree.
type SyncManager struct {
	vaultMu *sync.RWMutex
	git     *vault.Git

	cond *sync.Cond

	mu          sync.Mutex
	started     bool
	stopping    bool
	inProgress  bool
	pendingPull bool
	pendingPush bool
	pushMessage string

	lastPullAt  time.Time
	lastPushAt  time.Time
	lastError   string
	lastErrorAt time.Time

	// Tunables (hardcoded for now; can be promoted to config/env later).
	debounce time.Duration
	minPull  time.Duration
}

func NewSyncManager(vaultMu *sync.RWMutex, git *vault.Git) *SyncManager {
	sm := &SyncManager{
		vaultMu:  vaultMu,
		git:      git,
		debounce: 500 * time.Millisecond,
		minPull:  30 * time.Second,
	}
	sm.cond = sync.NewCond(&sm.mu)
	return sm
}

func (s *SyncManager) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return
	}
	s.started = true
	go s.loop()
}

func (s *SyncManager) Stop() {
	s.mu.Lock()
	s.stopping = true
	s.cond.Broadcast()
	s.mu.Unlock()
}

func (s *SyncManager) Status() SyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastPullAt *time.Time
	if !s.lastPullAt.IsZero() {
		t := s.lastPullAt
		lastPullAt = &t
	}
	var lastPushAt *time.Time
	if !s.lastPushAt.IsZero() {
		t := s.lastPushAt
		lastPushAt = &t
	}
	var lastErrorAt *time.Time
	if !s.lastErrorAt.IsZero() {
		t := s.lastErrorAt
		lastErrorAt = &t
	}

	return SyncStatus{
		InProgress:  s.inProgress,
		PendingPull: s.pendingPull,
		PendingPush: s.pendingPush,
		LastPullAt:  lastPullAt,
		LastPushAt:  lastPushAt,
		LastError:   s.lastError,
		LastErrorAt: lastErrorAt,
	}
}

// TriggerPull requests a pull. If recently pulled, this becomes a no-op.
func (s *SyncManager) TriggerPull() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.shouldPullLocked() {
		return
	}
	s.pendingPull = true
	s.cond.Signal()
}

// TriggerPullIfStale requests a pull only if last pull is older than maxAge.
func (s *SyncManager) TriggerPullIfStale(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inProgress || s.pendingPull {
		return
	}
	if !s.lastPullAt.IsZero() && time.Since(s.lastPullAt) < maxAge {
		return
	}
	s.pendingPull = true
	s.cond.Signal()
}

// TriggerPush requests a commit+push run. The message is best-effort; if multiple writes
// happen quickly, the last message wins.
func (s *SyncManager) TriggerPush(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingPush = true
	if message != "" {
		s.pushMessage = message
	}
	s.cond.Signal()
}

// SyncNow triggers a pull and optionally waits for completion (up to timeout).
func (s *SyncManager) SyncNow(wait bool, timeout time.Duration) SyncStatus {
	if wait {
		done := make(chan struct{})
		s.mu.Lock()
		if s.shouldPullLocked() {
			s.pendingPull = true
		}
		s.cond.Signal()
		s.mu.Unlock()

		go func() {
			s.waitIdle()
			close(done)
		}()

		if timeout <= 0 {
			<-done
			return s.Status()
		}
		select {
		case <-done:
			return s.Status()
		case <-time.After(timeout):
			return s.Status()
		}
	}

	s.TriggerPull()
	return s.Status()
}

func (s *SyncManager) waitIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.inProgress || s.pendingPull || s.pendingPush {
		s.cond.Wait()
	}
}

func (s *SyncManager) shouldPullLocked() bool {
	if s.inProgress || s.pendingPull {
		return false
	}
	// Rate-limit pulls to avoid paying the network tax on every page open.
	if !s.lastPullAt.IsZero() && time.Since(s.lastPullAt) < s.minPull {
		return false
	}
	return true
}

func (s *SyncManager) loop() {
	for {
		// Wait for work.
		s.mu.Lock()
		for !s.stopping && !s.pendingPull && !s.pendingPush {
			s.cond.Wait()
		}
		if s.stopping {
			s.mu.Unlock()
			return
		}

		// Debounce to coalesce bursts (e.g., multiple API calls from one UI action).
		debounce := s.debounce
		s.mu.Unlock()
		time.Sleep(debounce)

		s.mu.Lock()
		if s.stopping {
			s.mu.Unlock()
			return
		}
		doPull := s.pendingPull
		doPush := s.pendingPush
		msg := s.pushMessage
		s.pendingPull = false
		s.pendingPush = false
		s.pushMessage = ""
		s.inProgress = true
		s.mu.Unlock()

		// Serialize git work vs. file operations.
		s.vaultMu.Lock()
		var err error
		if doPull {
			if pullErr := s.git.Pull(); pullErr != nil {
				err = pullErr
			} else {
				s.mu.Lock()
				s.lastPullAt = time.Now()
				s.mu.Unlock()
			}
		}
		if doPush {
			if msg == "" {
				msg = "Sync changes"
			}
			if pushErr := s.git.CommitAndPush(msg); pushErr != nil {
				err = pushErr
			} else {
				s.mu.Lock()
				s.lastPushAt = time.Now()
				s.mu.Unlock()
			}
		}
		s.vaultMu.Unlock()

		// Update status and notify waiters.
		s.mu.Lock()
		if err != nil {
			s.lastError = err.Error()
			s.lastErrorAt = time.Now()
		}
		s.inProgress = false
		s.cond.Broadcast()
		s.mu.Unlock()
	}
}
