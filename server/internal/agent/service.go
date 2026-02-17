package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"notes-editor/internal/claude"
	"notes-editor/internal/linkedin"
	"notes-editor/internal/vault"
)

const (
	defaultMaxRunDuration   = 2 * time.Minute
	defaultMaxToolCalls     = 40
	piFallbackStatus        = "Gateway runtime unavailable; using Anthropic API key runtime for this run"
	emptyStreamStatus       = "No assistant output received (upstream closed without text/error)"
	maxStepsLimitStatusFmt  = "Action max_steps=%d applied for this run"
	toolCallLimitStatusFmt  = "Run exceeded max tool calls (%d)"
	defaultActionStepsLimit = 0
)

var ErrSessionBusy = errors.New("session already has an active run")

// ServiceOptions controls runtime behavior limits.
type ServiceOptions struct {
	MaxRunDuration  time.Duration
	MaxToolCalls    int
	AllowPiFallback *bool
}

// ChatRequest is the request body for agent chat endpoints.
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	ActionID  string `json:"action_id,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
}

// ChatResponse is the non-streaming response body for agent chat endpoint.
type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

// StreamRun contains run metadata and the event stream for one request.
type StreamRun struct {
	RunID  string
	Events <-chan StreamEvent
}

type runControl struct {
	person    string
	sessionID string
	startedAt time.Time
	updatedAt time.Time
	cancel    chan struct{}
	once      sync.Once
}

type resolvedMessage struct {
	Text           string
	ActionMaxSteps int
}

// ActiveRunSummary describes one currently running person-scoped stream run.
type ActiveRunSummary struct {
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id,omitempty"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Service orchestrates agent requests.
type Service struct {
	store           *vault.Store
	maxRunDuration  time.Duration
	maxToolCalls    int
	allowPiFallback bool

	mu                      sync.Mutex
	activeRuns              map[string]*runControl
	activeSessionRun        map[string]string
	sessionRecordsByPerson  map[string]map[string]*sessionRecord
	sessionSequenceByPerson map[string]int
	conversationsByPerson   map[string]map[string][]ConversationItem
	runtimes                map[string]Runtime
}

// NewService creates an agent service.
func NewService(claudeSvc *claude.Service, store *vault.Store) *Service {
	piRuntime := NewPiGatewayRuntime("").WithDependencies(store, nil)
	return NewServiceWithRuntimesAndOptions(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     NewAnthropicRuntime(claudeSvc),
		RuntimeModeGatewaySubscription: piRuntime,
	}, ServiceOptions{})
}

// NewServiceWithOptions creates a runtime-wired service.
func NewServiceWithOptions(claudeSvc *claude.Service, store *vault.Store, linkedinSvc *linkedin.Service, piGatewayURL string, options ServiceOptions) *Service {
	piRuntime := NewPiGatewayRuntime(piGatewayURL).WithDependencies(store, linkedinSvc)
	return NewServiceWithRuntimesAndOptions(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     NewAnthropicRuntime(claudeSvc),
		RuntimeModeGatewaySubscription: piRuntime,
	}, options)
}

// NewServiceWithRuntimes creates an agent service with explicit runtime map.
func NewServiceWithRuntimes(store *vault.Store, runtimes map[string]Runtime) *Service {
	return NewServiceWithRuntimesAndOptions(store, runtimes, ServiceOptions{})
}

// NewServiceWithRuntimesAndOptions creates an agent service with explicit runtime map and options.
func NewServiceWithRuntimesAndOptions(store *vault.Store, runtimes map[string]Runtime, options ServiceOptions) *Service {
	maxDuration := options.MaxRunDuration
	if maxDuration <= 0 {
		maxDuration = defaultMaxRunDuration
	}
	maxToolCalls := options.MaxToolCalls
	if maxToolCalls <= 0 {
		maxToolCalls = defaultMaxToolCalls
	}
	allowFallback := true
	if options.AllowPiFallback != nil {
		allowFallback = *options.AllowPiFallback
	}

	return &Service{
		store:                   store,
		maxRunDuration:          maxDuration,
		maxToolCalls:            maxToolCalls,
		allowPiFallback:         allowFallback,
		activeRuns:              make(map[string]*runControl),
		activeSessionRun:        make(map[string]string),
		sessionRecordsByPerson:  make(map[string]map[string]*sessionRecord),
		sessionSequenceByPerson: make(map[string]int),
		conversationsByPerson:   make(map[string]map[string][]ConversationItem),
		runtimes:                runtimes,
	}
}

// IsSessionBusy reports whether err indicates session concurrency conflict.
func IsSessionBusy(err error) bool {
	return errors.Is(err, ErrSessionBusy)
}

// Chat executes a non-streaming request.
func (s *Service) Chat(person string, req ChatRequest) (*ChatResponse, error) {
	resolved, err := s.resolveMessage(person, req)
	if err != nil {
		return nil, err
	}
	toolLimit := s.effectiveToolLimit(resolved.ActionMaxSteps)

	runtime, _, selectedMode, err := s.selectRuntime(person)
	if err != nil {
		return nil, err
	}
	usedRuntime := runtime

	runID := uuid.New().String()
	if err := s.tryBeginSessionRun(person, req.SessionID, runID); err != nil {
		return nil, err
	}
	defer s.endSessionRun(person, req.SessionID, runID)

	resp, err := runtime.Chat(person, RuntimeChatRequest{
		SessionID:    req.SessionID,
		Message:      resolved.Text,
		MaxToolCalls: toolLimit,
	})
	if err != nil {
		if !s.shouldAttemptPiFallback(selectedMode, err) {
			return nil, err
		}
		fallbackRuntime := s.runtimes[RuntimeModeAnthropicAPIKey]
		resp, err = fallbackRuntime.Chat(person, RuntimeChatRequest{
			SessionID:    req.SessionID,
			Message:      resolved.Text,
			MaxToolCalls: toolLimit,
		})
		if err != nil {
			return nil, err
		}
		usedRuntime = fallbackRuntime
	}
	s.touchSession(person, resp.SessionID, req.Message, usedRuntime.Mode())

	return &ChatResponse{
		Response:  resp.Response,
		SessionID: resp.SessionID,
		RunID:     runID,
	}, nil
}

// ChatStream executes a streaming request with v2 event schema.
func (s *Service) ChatStream(ctx context.Context, person string, req ChatRequest) (*StreamRun, error) {
	_ = ctx
	resolved, err := s.resolveMessage(person, req)
	if err != nil {
		return nil, err
	}
	toolLimit := s.effectiveToolLimit(resolved.ActionMaxSteps)

	runtime, fallbackMessage, selectedMode, err := s.selectRuntime(person)
	if err != nil {
		return nil, err
	}
	usedRuntime := runtime

	runID := uuid.New().String()
	if err := s.tryBeginSessionRun(person, req.SessionID, runID); err != nil {
		return nil, err
	}

	// Run lifecycle is intentionally detached from request context so runs can continue
	// when a client disconnects or switches views.
	streamCtx := context.Background()
	upstream, err := runtime.ChatStream(streamCtx, person, RuntimeChatRequest{
		SessionID:    req.SessionID,
		Message:      resolved.Text,
		MaxToolCalls: toolLimit,
	})
	if err != nil {
		if s.shouldAttemptPiFallback(selectedMode, err) {
			anthropic := s.runtimes[RuntimeModeAnthropicAPIKey]
			upstream, err = anthropic.ChatStream(streamCtx, person, RuntimeChatRequest{
				SessionID:    req.SessionID,
				Message:      resolved.Text,
				MaxToolCalls: toolLimit,
			})
			if err == nil {
				usedRuntime = anthropic
				fallbackMessage = piFallbackStatus
			}
		}
		if err != nil {
			s.endSessionRun(person, req.SessionID, runID)
			return nil, err
		}
	}

	run := s.registerRun(runID, person, req.SessionID)
	out := make(chan StreamEvent, 100)

	go func() {
		finalSessionID := req.SessionID
		runItems := make([]ConversationItem, 0, 24)
		var assistantText strings.Builder
		userText := strings.TrimSpace(req.Message)
		if userText == "" && strings.TrimSpace(req.ActionID) != "" {
			userText = "Run action: " + strings.TrimSpace(req.ActionID)
		}
		if userText != "" {
			runItems = append(runItems, ConversationItem{
				Type:    ConversationItemMessage,
				Role:    "user",
				Content: userText,
				RunID:   runID,
				TS:      time.Now().UTC(),
			})
		}

		seq := 0
		emit := func(event StreamEvent) {
			if event.RunID == "" {
				event.RunID = runID
			}
			if event.TS.IsZero() {
				event.TS = time.Now().UTC()
			}
			seq++
			if event.Seq == 0 {
				event.Seq = seq
			}
			// Persist assistant text in-order before non-text timeline items.
			if event.Type != "text" {
				if _, ok := conversationItemFromStreamEvent(event); ok && assistantText.Len() > 0 {
					runItems = append(runItems, ConversationItem{
						Type:      ConversationItemMessage,
						Role:      "assistant",
						Content:   assistantText.String(),
						SessionID: finalSessionID,
						RunID:     runID,
						TS:        event.TS,
					})
					assistantText.Reset()
				}
			}
			s.touchRunEvent(runID)
			out <- event
			if item, ok := conversationItemFromStreamEvent(event); ok {
				runItems = append(runItems, item)
			}
		}

		emitTerminal := func(message string) {
			emit(StreamEvent{
				Type:    "error",
				Message: message,
			})
			emit(StreamEvent{
				Type:      "done",
				SessionID: finalSessionID,
			})
		}

		defer close(out)
		defer s.unregisterRun(runID)
		defer s.endSessionRun(person, req.SessionID, runID)
		defer func() {
			if assistantText.Len() > 0 {
				runItems = append(runItems, ConversationItem{
					Type:      ConversationItemMessage,
					Role:      "assistant",
					Content:   assistantText.String(),
					SessionID: finalSessionID,
					RunID:     runID,
					TS:        time.Now().UTC(),
				})
			}
			if finalSessionID != "" && len(runItems) > 0 {
				for i := range runItems {
					if runItems[i].SessionID == "" {
						runItems[i].SessionID = finalSessionID
					}
				}
				s.appendStoredConversation(person, finalSessionID, runItems)
			}
			s.touchSession(person, finalSessionID, req.Message, usedRuntime.Mode())
		}()

		emit(StreamEvent{
			Type:      "start",
			SessionID: req.SessionID,
		})
		if fallbackMessage != "" {
			emit(StreamEvent{
				Type:    "status",
				Message: fallbackMessage,
			})
		}
		if resolved.ActionMaxSteps > defaultActionStepsLimit {
			emit(StreamEvent{
				Type:    "status",
				Message: fmt.Sprintf(maxStepsLimitStatusFmt, resolved.ActionMaxSteps),
			})
		}

		timer := time.NewTimer(s.maxRunDuration)
		defer timer.Stop()

		sawDone := false
		sawText := false
		sawError := false
		toolCallsSeen := 0

		for {
			select {
			case <-run.cancel:
				emitTerminal("Run cancelled")
				s.drainStream(upstream.Events)
				return
			case <-timer.C:
				emitTerminal("Run timed out")
				s.drainStream(upstream.Events)
				return
			case event, ok := <-upstream.Events:
				if !ok {
					if !sawDone {
						if !sawText && !sawError {
							emitTerminal(emptyStreamStatus)
						} else {
							emit(StreamEvent{
								Type:      "done",
								SessionID: finalSessionID,
							})
						}
					}
					return
				}
				if event.SessionID != "" {
					finalSessionID = event.SessionID
					s.updateRunSessionID(runID, finalSessionID)
				}
				if event.RunID == "" {
					event.RunID = runID
				}
				if event.Type == "tool_call" {
					toolCallsSeen++
					if toolLimit > 0 && toolCallsSeen > toolLimit {
						emitTerminal(fmt.Sprintf(toolCallLimitStatusFmt, toolLimit))
						s.drainStream(upstream.Events)
						return
					}
				}
				if event.Type == "text" && event.Delta != "" {
					sawText = true
					assistantText.WriteString(event.Delta)
				}
				if event.Type == "error" {
					sawError = true
				}
				if event.Type == "done" {
					if !sawText && !sawError {
						emitTerminal(emptyStreamStatus)
						sawDone = true
						continue
					}
					sawDone = true
				}
				if shouldSuppressStatusEvent(event) {
					continue
				}
				emit(event)
			}
		}
	}()

	return &StreamRun{
		RunID:  runID,
		Events: out,
	}, nil
}

// GetConfig returns per-person agent config.
func (s *Service) GetConfig(person string) (*Config, error) {
	return s.getConfig(person)
}

// SaveConfig updates per-person agent config.
func (s *Service) SaveConfig(person string, update ConfigUpdate) (*Config, error) {
	return s.saveConfig(person, update)
}

// ListActions returns available per-person actions.
func (s *Service) ListActions(person string) ([]Action, error) {
	return s.listActions(person)
}

func (s *Service) resolveMessage(person string, req ChatRequest) (*resolvedMessage, error) {
	if req.ActionID == "" {
		msg := strings.TrimSpace(req.Message)
		if msg == "" {
			return nil, fmt.Errorf("message is required")
		}
		return &resolvedMessage{Text: msg}, nil
	}

	action, err := s.resolveAction(person, req.ActionID)
	if err != nil {
		return nil, err
	}
	if action.Metadata.RequiresConfirmation && !req.Confirm {
		return nil, fmt.Errorf("action requires confirmation")
	}

	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		if strings.TrimSpace(action.Prompt) == "" {
			return nil, fmt.Errorf("action prompt is empty")
		}
		return &resolvedMessage{
			Text:           action.Prompt,
			ActionMaxSteps: action.Metadata.MaxSteps,
		}, nil
	}
	return &resolvedMessage{
		Text:           action.Prompt + "\n\nAdditional context:\n" + msg,
		ActionMaxSteps: action.Metadata.MaxSteps,
	}, nil
}

// ClearSession clears app-level session state for the selected runtime mode.
func (s *Service) ClearSession(person, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	runtime, err := s.runtimeForSession(person, sessionID)
	if err != nil {
		return err
	}
	if piRuntime, ok := runtime.(*PiGatewayRuntime); ok {
		if err := piRuntime.ClearSessionForPerson(person, sessionID); err != nil {
			return err
		}
	} else {
		if err := runtime.ClearSession(sessionID); err != nil {
			return err
		}
	}
	s.removeSessionRecord(person, sessionID)
	s.removeStoredConversation(person, sessionID)
	return nil
}

// GetConversationHistory returns unified persisted conversation items for a session.
func (s *Service) GetConversationHistory(person, sessionID string) ([]ConversationItem, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	if history := s.getStoredConversation(person, sessionID); len(history) > 0 {
		return history, nil
	}

	runtime, err := s.runtimeForSession(person, sessionID)
	if err != nil {
		return nil, err
	}
	var history []claude.ChatMessage
	if piRuntime, ok := runtime.(*PiGatewayRuntime); ok {
		history, err = piRuntime.GetHistoryForPerson(person, sessionID)
	} else {
		history, err = runtime.GetHistory(sessionID)
	}
	if err != nil {
		return nil, err
	}

	items := chatMessagesToItems(history)
	if len(items) > 0 {
		s.replaceStoredConversation(person, sessionID, items)
	}
	s.touchSession(person, sessionID, "", runtime.Mode())
	return items, nil
}

// GetHistory returns chat-only history for legacy Claude endpoints.
func (s *Service) GetHistory(person, sessionID string) ([]claude.ChatMessage, error) {
	items, err := s.GetConversationHistory(person, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]claude.ChatMessage, 0, len(items))
	for _, item := range items {
		if item.Type != ConversationItemMessage {
			continue
		}
		if item.Role != "user" && item.Role != "assistant" {
			continue
		}
		out = append(out, claude.ChatMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}
	return out, nil
}

// StopRun stops a currently active streaming run.
func (s *Service) StopRun(person, runID string) bool {
	s.mu.Lock()
	run, ok := s.activeRuns[runID]
	s.mu.Unlock()
	if !ok || run.person != person {
		return false
	}
	run.once.Do(func() {
		close(run.cancel)
	})
	return true
}

// ListActiveRuns returns currently running streams for one person.
func (s *Service) ListActiveRuns(person string) []ActiveRunSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ActiveRunSummary, 0, len(s.activeRuns))
	for runID, run := range s.activeRuns {
		if run.person != person {
			continue
		}
		out = append(out, ActiveRunSummary{
			RunID:     runID,
			SessionID: run.sessionID,
			StartedAt: run.startedAt,
			UpdatedAt: run.updatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}

func (s *Service) getStoredConversation(person, sessionID string) []ConversationItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.conversationsByPerson[person]
	if personSessions == nil {
		return nil
	}
	items := personSessions[sessionID]
	if len(items) == 0 {
		return nil
	}
	out := make([]ConversationItem, len(items))
	copy(out, items)
	return out
}

func (s *Service) replaceStoredConversation(person, sessionID string, items []ConversationItem) {
	if len(items) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.conversationsByPerson[person]
	if personSessions == nil {
		personSessions = make(map[string][]ConversationItem)
		s.conversationsByPerson[person] = personSessions
	}
	copied := make([]ConversationItem, len(items))
	copy(copied, items)
	personSessions[sessionID] = copied
}

func (s *Service) appendStoredConversation(person, sessionID string, items []ConversationItem) {
	if len(items) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.conversationsByPerson[person]
	if personSessions == nil {
		personSessions = make(map[string][]ConversationItem)
		s.conversationsByPerson[person] = personSessions
	}
	personSessions[sessionID] = append(personSessions[sessionID], items...)
}

func (s *Service) removeStoredConversation(person, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.conversationsByPerson[person]
	if personSessions == nil {
		return
	}
	delete(personSessions, sessionID)
	if len(personSessions) == 0 {
		delete(s.conversationsByPerson, person)
	}
}

func (s *Service) registerRun(runID, person, sessionID string) *runControl {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	run := &runControl{
		person:    person,
		sessionID: strings.TrimSpace(sessionID),
		startedAt: now,
		updatedAt: now,
		cancel:    make(chan struct{}),
	}
	s.activeRuns[runID] = run
	return run
}

func (s *Service) unregisterRun(runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeRuns, runID)
}

func (s *Service) touchRunEvent(runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.activeRuns[runID]
	if !ok {
		return
	}
	run.updatedAt = time.Now().UTC()
}

func (s *Service) updateRunSessionID(runID, sessionID string) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.activeRuns[runID]
	if !ok {
		return
	}
	run.sessionID = sid
	run.updatedAt = time.Now().UTC()
}

func (s *Service) tryBeginSessionRun(person, sessionID, runID string) error {
	if sessionID == "" {
		return nil
	}

	key := sessionRunKey(person, sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if activeRunID, ok := s.activeSessionRun[key]; ok {
		return fmt.Errorf("%w: session_id %q is busy (run_id %s)", ErrSessionBusy, sessionID, activeRunID)
	}
	s.activeSessionRun[key] = runID
	return nil
}

func (s *Service) endSessionRun(person, sessionID, runID string) {
	if sessionID == "" {
		return
	}

	key := sessionRunKey(person, sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if activeRunID, ok := s.activeSessionRun[key]; ok && activeRunID == runID {
		delete(s.activeSessionRun, key)
	}
}

func sessionRunKey(person, sessionID string) string {
	return person + "::" + sessionID
}

func (s *Service) emitTerminal(events chan<- StreamEvent, sessionID, runID, message string) {
	events <- StreamEvent{
		Type:    "error",
		Message: message,
		RunID:   runID,
	}
	events <- StreamEvent{
		Type:      "done",
		SessionID: sessionID,
		RunID:     runID,
	}
}

func (s *Service) selectRuntime(person string) (Runtime, string, string, error) {
	cfg, err := s.getConfig(person)
	if err != nil {
		return nil, "", "", err
	}
	runtime, status, err := s.selectRuntimeForMode(cfg.RuntimeMode)
	return runtime, status, cfg.RuntimeMode, err
}

func (s *Service) selectRuntimeForMode(mode string) (Runtime, string, error) {
	switch mode {
	case "", RuntimeModeAnthropicAPIKey:
		runtime := s.runtimes[RuntimeModeAnthropicAPIKey]
		if runtime == nil || !runtime.Available() {
			return nil, "", &RuntimeUnavailableError{
				Mode:   RuntimeModeAnthropicAPIKey,
				Reason: "Anthropic runtime not configured",
			}
		}
		return runtime, "", nil
	case RuntimeModeGatewaySubscription:
		piRuntime := s.runtimes[RuntimeModeGatewaySubscription]
		if piRuntime != nil && piRuntime.Available() {
			return piRuntime, "", nil
		}

		if s.allowPiFallback {
			anthropicRuntime := s.runtimes[RuntimeModeAnthropicAPIKey]
			if anthropicRuntime != nil && anthropicRuntime.Available() {
				return anthropicRuntime, piFallbackStatus, nil
			}
		}

		return nil, "", &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: "Gateway runtime unavailable and Anthropic fallback not available",
		}
	default:
		return nil, "", fmt.Errorf("unsupported runtime mode: %q", mode)
	}
}

func (s *Service) runtimeForSession(person, sessionID string) (Runtime, error) {
	if mode, ok := s.runtimeModeForSession(person, sessionID); ok {
		runtime := s.runtimes[mode]
		if runtime != nil && runtime.Available() {
			return runtime, nil
		}
		return nil, &RuntimeUnavailableError{
			Mode:   mode,
			Reason: "session runtime unavailable",
		}
	}

	runtime, _, _, err := s.selectRuntime(person)
	if err != nil {
		return nil, err
	}
	return runtime, nil
}

func (s *Service) shouldAttemptPiFallback(selectedMode string, err error) bool {
	if selectedMode != RuntimeModeGatewaySubscription || !s.allowPiFallback {
		return false
	}
	anthropic := s.runtimes[RuntimeModeAnthropicAPIKey]
	if anthropic == nil || !anthropic.Available() {
		return false
	}
	return IsRuntimeUnavailable(err)
}

func (s *Service) drainStream(events <-chan StreamEvent) {
	go func() {
		for range events {
		}
	}()
}

func (s *Service) effectiveToolLimit(actionMaxSteps int) int {
	limit := s.maxToolCalls
	if actionMaxSteps > defaultActionStepsLimit && (limit == 0 || actionMaxSteps < limit) {
		limit = actionMaxSteps
	}
	return limit
}

func shouldSuppressStatusEvent(event StreamEvent) bool {
	if event.Type != "status" {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(event.Message))
	return strings.Contains(msg, "gateway mode==")
}

func conversationItemFromStreamEvent(event StreamEvent) (ConversationItem, bool) {
	base := ConversationItem{
		SessionID: event.SessionID,
		RunID:     event.RunID,
		Seq:       event.Seq,
		TS:        event.TS,
	}
	switch event.Type {
	case "tool_call":
		base.Type = ConversationItemToolCall
		base.Tool = event.Tool
		base.Args = event.Args
		return base, true
	case "tool_result":
		base.Type = ConversationItemToolResult
		base.Tool = event.Tool
		base.OK = event.OK
		base.Summary = event.Summary
		return base, true
	case "status":
		base.Type = ConversationItemStatus
		base.Message = event.Message
		return base, true
	case "error":
		base.Type = ConversationItemError
		base.Message = event.Message
		return base, true
	case "usage":
		base.Type = ConversationItemUsage
		base.Usage = event.Usage
		return base, true
	default:
		return ConversationItem{}, false
	}
}

func mapClaudeEvent(event claude.StreamEvent) []StreamEvent {
	switch event.Type {
	case "text":
		return []StreamEvent{{
			Type:  "text",
			Delta: event.Delta,
		}}
	case "tool_use":
		args := map[string]any{}
		if input, ok := event.Input.(map[string]any); ok {
			args = input
		}
		return []StreamEvent{{
			Type: "tool_call",
			Tool: event.Name,
			Args: args,
		}}
	case "status":
		// Current Claude runtime emits "Tool <name> executed" after each tool call.
		if strings.HasPrefix(event.Message, "Tool ") && strings.HasSuffix(event.Message, " executed") {
			toolName := strings.TrimSuffix(strings.TrimPrefix(event.Message, "Tool "), " executed")
			return []StreamEvent{{
				Type:    "tool_result",
				Tool:    toolName,
				OK:      true,
				Summary: event.Message,
			}}
		}
		return []StreamEvent{{
			Type:    "status",
			Message: event.Message,
		}}
	case "error":
		return []StreamEvent{{
			Type:    "error",
			Message: event.Message,
		}}
	case "usage":
		if event.Usage == nil {
			return nil
		}
		usage := &UsageSnapshot{
			InputTokens:      event.Usage.InputTokens,
			OutputTokens:     event.Usage.OutputTokens,
			CacheReadTokens:  event.Usage.CacheReadTokens,
			CacheWriteTokens: event.Usage.CacheWriteTokens,
			TotalTokens:      event.Usage.TotalTokens,
			ContextWindow:    event.Usage.ContextWindow,
			RemainingTokens:  event.Usage.RemainingTokens,
		}
		return []StreamEvent{{
			Type:  "usage",
			Usage: usage,
		}}
	case "done":
		return []StreamEvent{{
			Type:      "done",
			SessionID: event.SessionID,
		}}
	default:
		return nil
	}
}
