package codex

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

const defaultSessionPollInterval = 500 * time.Millisecond
const maxActiveSessionFiles = 4
const sessionHeadBytes = 1024 * 1024
const sessionTailBytes = 8 * 1024 * 1024

type sessionEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionStore interface {
	SaveConversationRequest(context.Context, conversations.Request) error
	UpsertClientConnection(context.Context, conversations.ClientConnection) error
}

type SessionWatcherConfig struct {
	Root          string
	Store         sessionStore
	PollInterval  time.Duration
	RecentWindow  time.Duration
	Now           func() time.Time
	Log           io.Writer
	OnSaved       func(string)
	CapturePolicy func(context.Context) (enabled bool, capturePrompts bool)
}

type SessionWatcher struct {
	config         SessionWatcherConfig
	mu             sync.Mutex
	fileSizes      map[string]int64
	fileActive     map[string]bool
	signatures     map[string]string
	policySet      bool
	captureEnabled bool
	capturePrompts bool
	captureAfter   time.Time
}

func NewSessionWatcher(config SessionWatcherConfig) *SessionWatcher {
	if config.PollInterval <= 0 {
		config.PollInterval = defaultSessionPollInterval
	}
	if config.RecentWindow <= 0 {
		config.RecentWindow = 24 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Log == nil {
		config.Log = io.Discard
	}
	return &SessionWatcher{config: config, fileSizes: map[string]int64{}, fileActive: map[string]bool{}, signatures: map[string]string{}}
}

func (watcher *SessionWatcher) Run(ctx context.Context) error {
	if strings.TrimSpace(watcher.config.Root) == "" || watcher.config.Store == nil {
		return errors.New("Codex session root and store are required")
	}
	enabled, _ := watcher.capturePolicy(ctx)
	now := watcher.config.Now().UTC()
	detail := "已连接 Codex 本地会话监听，等待新消息（无需重启 Codex）"
	if !enabled {
		detail = "Codex 本地会话监听已连接；当前采集已暂停"
	}
	if err := watcher.config.Store.UpsertClientConnection(ctx, conversations.ClientConnection{
		Key: "codex", DisplayName: "Codex", Detected: true, State: "connected",
		Capability: "本地会话实时监听", Detail: detail,
		LastHeartbeatAt: &now, UpdatedAt: now,
	}); err != nil {
		return err
	}
	if err := watcher.scan(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(watcher.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := watcher.scan(ctx); err != nil && !errors.Is(err, context.Canceled) {
				fmt.Fprintln(watcher.config.Log, "Agent Doctor: Codex 会话监听暂时失败，将自动重试")
			}
		}
	}
}

func (watcher *SessionWatcher) scan(ctx context.Context) error {
	enabled, capturePrompts := watcher.capturePolicy(ctx)
	now := watcher.config.Now().UTC()
	watcher.mu.Lock()
	wasSet := watcher.policySet
	wasEnabled := watcher.captureEnabled
	wasCapturingPrompts := watcher.capturePrompts
	resumed := wasSet && enabled && !wasEnabled
	if wasSet && enabled && (!wasEnabled || (!wasCapturingPrompts && capturePrompts)) {
		watcher.captureAfter = now
	}
	watcher.captureEnabled = enabled
	watcher.capturePrompts = capturePrompts
	watcher.policySet = true
	watcher.mu.Unlock()
	if !enabled {
		if err := watcher.snapshotFileSizes(); err != nil {
			return err
		}
		if !wasSet || wasEnabled {
			_ = watcher.config.Store.UpsertClientConnection(ctx, conversations.ClientConnection{
				Key: "codex", DisplayName: "Codex", Detected: true, State: "connected",
				Capability: "本地会话实时监听", Detail: "Codex 本地会话监听已连接；当前采集已暂停",
				LastHeartbeatAt: &now, UpdatedAt: now,
			})
		}
		return nil
	}
	files, err := recentSessionFiles(watcher.config.Root, watcher.config.Now().Add(-watcher.config.RecentWindow))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(files) > maxActiveSessionFiles {
		files = files[:maxActiveSessionFiles]
	}
	totalSaved := 0
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		watcher.mu.Lock()
		previousSize, seen := watcher.fileSizes[path]
		watcher.mu.Unlock()
		if seen && previousSize == info.Size() {
			continue
		}
		saved, active, importErr := watcher.importFile(ctx, path)
		if importErr != nil {
			fmt.Fprintf(watcher.config.Log, "Agent Doctor: Codex 会话文件 %s 导入失败：%v\n", filepath.Base(path), importErr)
			continue
		}
		watcher.mu.Lock()
		watcher.fileSizes[path] = info.Size()
		watcher.fileActive[path] = active
		watcher.mu.Unlock()
		totalSaved += saved
	}
	if totalSaved > 0 || resumed {
		watcher.mu.Lock()
		anyActive := false
		for _, path := range files {
			if watcher.fileActive[path] {
				anyActive = true
				break
			}
		}
		watcher.mu.Unlock()
		now := watcher.config.Now().UTC()
		state := "connected"
		detail := "Codex 本地会话监听已连接，等待下一次调用（无需重启）"
		if anyActive {
			state = "active"
			detail = "正在实时采集 Codex 本地会话（无需重启）"
		}
		_ = watcher.config.Store.UpsertClientConnection(ctx, conversations.ClientConnection{
			Key: "codex", DisplayName: "Codex", Detected: true, State: state,
			Capability: "本地会话实时监听", Detail: detail,
			LastHeartbeatAt: &now, UpdatedAt: now,
		})
		fmt.Fprintf(watcher.config.Log, "Agent Doctor: Codex 已采集 %d 条会话更新\n", totalSaved)
	}
	return nil
}

func (watcher *SessionWatcher) snapshotFileSizes() error {
	files, err := recentSessionFiles(watcher.config.Root, watcher.config.Now().Add(-watcher.config.RecentWindow))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	for _, path := range files {
		if info, statErr := os.Stat(path); statErr == nil {
			watcher.fileSizes[path] = info.Size()
			watcher.fileActive[path] = false
		}
	}
	return nil
}

func (watcher *SessionWatcher) importFile(ctx context.Context, path string) (int, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()
	watcher.mu.Lock()
	captureAfter := watcher.captureAfter
	watcher.mu.Unlock()
	requests, err := parseCurrentSessionFileAfter(file, captureAfter)
	if err != nil {
		return 0, false, err
	}
	saved := 0
	_, capturePrompts := watcher.capturePolicy(ctx)
	active := false
	for _, record := range requests {
		if record.CompletedAt == nil {
			active = true
		}
		if !capturePrompts {
			record.Messages = nil
		}
		signature := requestSignature(record)
		watcher.mu.Lock()
		unchanged := watcher.signatures[record.ID] == signature
		watcher.mu.Unlock()
		if unchanged {
			continue
		}
		if err := watcher.config.Store.SaveConversationRequest(ctx, record); err != nil {
			return saved, active, err
		}
		watcher.mu.Lock()
		watcher.signatures[record.ID] = signature
		watcher.mu.Unlock()
		saved++
		if watcher.config.OnSaved != nil {
			watcher.config.OnSaved(record.SessionID)
		}
	}
	return saved, active, nil
}

func (watcher *SessionWatcher) capturePolicy(ctx context.Context) (bool, bool) {
	if watcher.config.CapturePolicy == nil {
		return true, true
	}
	return watcher.config.CapturePolicy(ctx)
}

func recentSessionFiles(root string, cutoff time.Time) ([]string, error) {
	type candidate struct {
		path string
		mod  time.Time
	}
	candidates := []candidate{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jsonl") {
			return nil
		}
		info, err := entry.Info()
		if err == nil && !info.ModTime().Before(cutoff) {
			candidates = append(candidates, candidate{path: path, mod: info.ModTime()})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(candidates, func(left, right int) bool { return candidates[left].mod.After(candidates[right].mod) })
	paths := make([]string, 0, len(candidates))
	for _, item := range candidates {
		paths = append(paths, item.path)
	}
	return paths, nil
}

func ParseSessionLog(input io.Reader) ([]conversations.Request, error) {
	return parseSessionLogWithParser(input, newSessionParser())
}

func parseSessionLogWithParser(input io.Reader, parser *sessionParser) ([]conversations.Request, error) {
	reader := bufio.NewReaderSize(input, 64*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(readErr, io.EOF) {
			break
		}
		var envelope sessionEnvelope
		if err := json.Unmarshal(bytes.TrimSpace(line), &envelope); err == nil {
			parser.consume(envelope)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break // tolerate a partially written final line
			}
			return nil, fmt.Errorf("read Codex session log: %w", readErr)
		}
	}
	return parser.requests(), nil
}

// ParseCurrentSessionLog finds the newest Codex turn without normalizing every
// historical message in a long-lived task. The watcher re-runs this function
// after appends, so prior turns are already persisted before a new turn starts.
func ParseCurrentSessionLog(input io.Reader) ([]conversations.Request, error) {
	return parseCurrentSessionLogAfter(input, time.Time{})
}

func parseCurrentSessionLogAfter(input io.Reader, captureAfter time.Time) ([]conversations.Request, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read current Codex session: %w", err)
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	latestMeta := -1
	metaForTurn := -1
	latestTurn := -1
	for index, line := range lines {
		var envelope sessionEnvelope
		if json.Unmarshal(bytes.TrimSpace(line), &envelope) != nil {
			continue
		}
		if envelope.Type == "session_meta" {
			latestMeta = index
			continue
		}
		if envelope.Type != "event_msg" {
			continue
		}
		var payload struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(envelope.Payload, &payload) == nil && payload.Type == "task_started" {
			latestTurn = index
			metaForTurn = latestMeta
		}
	}
	if latestTurn < 0 {
		parser := newSessionParserAfter(captureAfter)
		return parseSessionLogWithParser(bytes.NewReader(data), parser)
	}
	selected := make([]byte, 0, len(data)/4)
	if metaForTurn >= 0 && metaForTurn != latestTurn {
		selected = append(selected, lines[metaForTurn]...)
	}
	for _, line := range lines[latestTurn:] {
		selected = append(selected, line...)
	}
	parser := newSessionParserAfter(captureAfter)
	return parseSessionLogWithParser(bytes.NewReader(selected), parser)
}

// ParseCurrentSessionFile avoids loading an arbitrarily large long-running
// Codex task. Session identity comes from complete lines at the file head while
// the newest turn comes from a bounded tail window.
func ParseCurrentSessionFile(file *os.File) ([]conversations.Request, error) {
	return parseCurrentSessionFileAfter(file, time.Time{})
}

func parseCurrentSessionFileAfter(file *os.File, captureAfter time.Time) ([]conversations.Request, error) {
	if file == nil {
		return nil, errors.New("Codex session file is required")
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect Codex session file: %w", err)
	}
	if info.Size() <= sessionTailBytes {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		return parseCurrentSessionLogAfter(file, captureAfter)
	}
	head, err := readFileRange(file, 0, minInt64(info.Size(), sessionHeadBytes))
	if err != nil {
		return nil, err
	}
	if newline := bytes.LastIndexByte(head, '\n'); newline >= 0 {
		head = head[:newline+1]
	}
	turnOffset, found, err := findLatestTaskStartOffset(file, info.Size())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("latest Codex turn was not found")
	}
	tail, err := readFileRange(file, turnOffset, info.Size()-turnOffset)
	if err != nil {
		return nil, err
	}
	combined := make([]byte, 0, len(head)+len(tail))
	combined = append(combined, head...)
	combined = append(combined, tail...)
	return parseCurrentSessionLogAfter(bytes.NewReader(combined), captureAfter)
}

func findLatestTaskStartOffset(file *os.File, size int64) (int64, bool, error) {
	end := size
	carry := []byte{}
	for end > 0 {
		start := end - sessionTailBytes
		if start < 0 {
			start = 0
		}
		chunk, err := readFileRange(file, start, end-start)
		if err != nil {
			return 0, false, err
		}
		combined := make([]byte, 0, len(chunk)+len(carry))
		combined = append(combined, chunk...)
		combined = append(combined, carry...)
		lineStart := 0
		starts := []int{}
		lines := [][]byte{}
		for index, value := range combined {
			if value == '\n' {
				starts = append(starts, lineStart)
				lines = append(lines, combined[lineStart:index+1])
				lineStart = index + 1
			}
		}
		if start == 0 && lineStart < len(combined) {
			starts = append(starts, lineStart)
			lines = append(lines, combined[lineStart:])
		}
		firstComplete := 0
		if start > 0 {
			firstComplete = 1
		}
		for index := len(lines) - 1; index >= firstComplete; index-- {
			if tailHasTaskStarted(lines[index]) {
				return start + int64(starts[index]), true, nil
			}
		}
		if start > 0 {
			if len(lines) > 0 {
				carry = append([]byte(nil), lines[0]...)
			} else {
				carry = combined
			}
		}
		end = start
	}
	return 0, false, nil
}

func tailHasTaskStarted(data []byte) bool {
	reader := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			return false
		}
		var envelope sessionEnvelope
		if json.Unmarshal(bytes.TrimSpace(line), &envelope) == nil && envelope.Type == "event_msg" {
			var payload struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(envelope.Payload, &payload) == nil && payload.Type == "task_started" {
				return true
			}
		}
		if err != nil {
			return false
		}
	}
}

func readFileRange(file *os.File, offset, length int64) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}
	buffer := make([]byte, int(length))
	read, err := file.ReadAt(buffer, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read Codex session file range: %w", err)
	}
	return buffer[:read], nil
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

type sessionParser struct {
	sessionID      string
	projectID      string
	clientVersion  string
	currentTurn    string
	turns          map[string]*conversations.Request
	order          []string
	tokenSnapshots map[string]string
	captureAfter   time.Time
}

func newSessionParser() *sessionParser {
	return newSessionParserAfter(time.Time{})
}

func newSessionParserAfter(captureAfter time.Time) *sessionParser {
	return &sessionParser{clientVersion: "not-reported", projectID: "codex-local", turns: map[string]*conversations.Request{}, tokenSnapshots: map[string]string{}, captureAfter: captureAfter}
}

func (parser *sessionParser) consume(envelope sessionEnvelope) {
	timestamp := parseTimestamp(envelope.Timestamp)
	switch envelope.Type {
	case "session_meta":
		var payload struct {
			ID         string `json:"id"`
			SessionID  string `json:"session_id"`
			WorkingDir string `json:"cwd"`
			Version    string `json:"cli_version"`
		}
		if json.Unmarshal(envelope.Payload, &payload) == nil {
			parser.sessionID = firstNonempty(payload.SessionID, payload.ID, parser.sessionID)
			parser.projectID = firstNonempty(payload.WorkingDir, parser.projectID)
			parser.clientVersion = firstNonempty(payload.Version, parser.clientVersion)
		}
	case "turn_context":
		var payload struct {
			TurnID     string `json:"turn_id"`
			Model      string `json:"model"`
			WorkingDir string `json:"cwd"`
		}
		if json.Unmarshal(envelope.Payload, &payload) == nil {
			turn := parser.ensureTurn(firstNonempty(payload.TurnID, parser.currentTurn), timestamp)
			if turn != nil {
				turn.Model.DisplayName = firstNonempty(payload.Model, turn.Model.DisplayName)
				turn.ProjectID = firstNonempty(payload.WorkingDir, turn.ProjectID)
			}
		}
	case "event_msg":
		parser.consumeEvent(envelope.Payload, timestamp)
	case "response_item":
		parser.consumeResponse(envelope.Payload, timestamp)
	}
}

func (parser *sessionParser) consumeEvent(raw json.RawMessage, timestamp time.Time) {
	var event struct {
		Type        string          `json:"type"`
		TurnID      string          `json:"turn_id"`
		StartedAt   json.RawMessage `json:"started_at"`
		CompletedAt json.RawMessage `json:"completed_at"`
		DurationMS  int64           `json:"duration_ms"`
		FirstByteMS int64           `json:"time_to_first_token_ms"`
		Info        struct {
			Last struct {
				Input     int64 `json:"input_tokens"`
				Output    int64 `json:"output_tokens"`
				Cached    int64 `json:"cached_input_tokens"`
				Reasoning int64 `json:"reasoning_output_tokens"`
			} `json:"last_token_usage"`
			Total struct {
				Input     int64 `json:"input_tokens"`
				Output    int64 `json:"output_tokens"`
				Cached    int64 `json:"cached_input_tokens"`
				Reasoning int64 `json:"reasoning_output_tokens"`
			} `json:"total_token_usage"`
		} `json:"info"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return
	}
	tokenSnapshot, hasTokenTotal := tokenUsageSnapshot(event.Info.Total.Input, event.Info.Total.Output, event.Info.Total.Cached, event.Info.Total.Reasoning)
	if event.Type == "token_count" && !parser.captureAfter.IsZero() && !timestamp.After(parser.captureAfter) {
		if turn := parser.ensureTurn(parser.currentTurn, timestamp); turn != nil && hasTokenTotal {
			parser.tokenSnapshots[turn.ID] = tokenSnapshot
		}
		return
	}
	if event.Type != "task_started" && !parser.captureAfter.IsZero() && !timestamp.After(parser.captureAfter) {
		return
	}
	switch event.Type {
	case "task_started":
		parser.currentTurn = event.TurnID
		turn := parser.ensureTurn(event.TurnID, flexibleEventTime(event.StartedAt, timestamp))
		if turn != nil {
			turn.StartedAt = flexibleEventTime(event.StartedAt, timestamp)
		}
	case "token_count":
		if turn := parser.ensureTurn(parser.currentTurn, timestamp); turn != nil {
			if hasTokenTotal && parser.tokenSnapshots[turn.ID] == tokenSnapshot {
				return
			}
			if hasTokenTotal {
				parser.tokenSnapshots[turn.ID] = tokenSnapshot
			}
			turn.Usage.InputTokens = addTokenCount(turn.Usage.InputTokens, event.Info.Last.Input)
			turn.Usage.OutputTokens = addTokenCount(turn.Usage.OutputTokens, event.Info.Last.Output)
			turn.Usage.CachedTokens = addTokenCount(turn.Usage.CachedTokens, event.Info.Last.Cached)
			turn.Usage.ReasoningTokens = addTokenCount(turn.Usage.ReasoningTokens, event.Info.Last.Reasoning)
			turn.Usage.Precision = string(events.PrecisionExact)
			turn.Usage.Provenance = "codex-session-log"
		}
	case "task_complete", "turn_aborted":
		turn := parser.ensureTurn(firstNonempty(event.TurnID, parser.currentTurn), timestamp)
		if turn != nil {
			completed := flexibleEventTime(event.CompletedAt, timestamp)
			turn.CompletedAt = &completed
			turn.DurationMS = event.DurationMS
			if turn.DurationMS <= 0 && completed.After(turn.StartedAt) {
				turn.DurationMS = completed.Sub(turn.StartedAt).Milliseconds()
			}
			if event.FirstByteMS > 0 {
				turn.FirstByteMS = event.FirstByteMS
			}
			turn.StatusCode = 200
		}
	}
}

func tokenUsageSnapshot(input, output, cached, reasoning int64) (string, bool) {
	return fmt.Sprintf("%d/%d/%d/%d", input, output, cached, reasoning), input != 0 || output != 0 || cached != 0 || reasoning != 0
}

func (parser *sessionParser) consumeResponse(raw json.RawMessage, timestamp time.Time) {
	if !parser.captureAfter.IsZero() && !timestamp.After(parser.captureAfter) {
		return
	}
	turn := parser.ensureTurn(parser.currentTurn, timestamp)
	if turn == nil {
		return
	}
	var item struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		CallID    string          `json:"call_id"`
		Role      string          `json:"role"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Input     json.RawMessage `json:"input"`
		Output    json.RawMessage `json:"output"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &item) != nil {
		return
	}
	message := conversations.Message{ID: firstNonempty(item.ID, item.CallID, messageID(turn.ID, len(turn.Messages))), RequestID: turn.ID, Sequence: len(turn.Messages), CreatedAt: timestamp}
	switch item.Type {
	case "message":
		if item.Role != "user" && item.Role != "assistant" {
			return
		}
		message.Role = item.Role
		parts := []string{}
		for _, part := range item.Content {
			if part.Type == "input_text" || part.Type == "output_text" {
				parts = append(parts, part.Text)
			}
		}
		message.Content = privacy.FilterText(strings.TrimSpace(strings.Join(parts, "\n")))
		if message.Content == "" {
			return
		}
		if item.Role == "assistant" && turn.FirstByteMS <= 0 && timestamp.After(turn.StartedAt) {
			turn.FirstByteMS = timestamp.Sub(turn.StartedAt).Milliseconds()
		}
	case "function_call", "custom_tool_call":
		message.Role = "tool"
		message.ToolName = boundedLabel(item.Name, "tool")
		message.ToolPayloadJSON = filterToolArguments(firstRawMessage(item.Arguments, item.Input))
	case "function_call_output", "custom_tool_call_output":
		message.ID = firstNonempty(item.ID, messageID(turn.ID, len(turn.Messages)))
		message.Role = "tool"
		message.ToolName = "tool_result"
		message.ToolPayloadJSON = filterToolArguments(item.Output)
	default:
		return
	}
	turn.Messages = append(turn.Messages, message)
}

func firstRawMessage(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 && string(value) != "null" && string(value) != `""` {
			return value
		}
	}
	return nil
}

func (parser *sessionParser) ensureTurn(turnID string, startedAt time.Time) *conversations.Request {
	if strings.TrimSpace(turnID) == "" {
		return nil
	}
	if existing := parser.turns[turnID]; existing != nil {
		return existing
	}
	requestID := "codex-" + turnID
	record := &conversations.Request{
		ID: requestID, SessionID: firstNonempty(parser.sessionID, "codex-session-unknown"), ProjectID: parser.projectID,
		Client: events.ClientRef{Name: "codex", Version: parser.clientVersion}, Model: events.ModelRef{DisplayName: "not-reported"},
		Protocol: "codex-session-log", Method: "LOCAL", Path: "codex://session", StartedAt: startedAt,
		Usage:    conversations.Usage{Precision: string(events.PrecisionUnavailable), Provenance: "codex-session-log"},
		Cost:     conversations.Cost{Currency: "USD", Precision: string(events.PrecisionUnavailable), Provenance: "price-not-observed"},
		Messages: []conversations.Message{},
	}
	parser.turns[turnID] = record
	parser.order = append(parser.order, turnID)
	return record
}

func (parser *sessionParser) requests() []conversations.Request {
	result := make([]conversations.Request, 0, len(parser.order))
	for _, turnID := range parser.order {
		record := parser.turns[turnID]
		if record != nil && len(record.Messages) > 0 {
			result = append(result, *record)
		}
	}
	return result
}

func requestSignature(record conversations.Request) string {
	encoded, _ := json.Marshal(record)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func filterToolArguments(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var encoded string
	if json.Unmarshal(raw, &encoded) != nil {
		encoded = string(raw)
	}
	if filtered, err := privacy.FilterJSON([]byte(encoded)); err == nil {
		return string(filtered)
	}
	return privacy.FilterText(encoded)
}

func int64Pointer(value int64) *int64 { return &value }

func addTokenCount(current *int64, value int64) *int64 {
	if current != nil {
		value += *current
	}
	return int64Pointer(value)
}

func parseTimestamp(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	return time.Unix(0, 0).UTC()
}

func firstValidTime(value string, fallback time.Time) time.Time {
	if parsed := parseTimestamp(value); !parsed.Equal(time.Unix(0, 0).UTC()) {
		return parsed
	}
	return fallback
}

func flexibleEventTime(raw json.RawMessage, fallback time.Time) time.Time {
	if len(raw) == 0 || string(raw) == "null" {
		return fallback
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return firstValidTime(text, fallback)
	}
	// Desktop builds may encode this field as a monotonic/epoch number. The
	// envelope timestamp is the portable wall-clock source used for storage.
	return fallback
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func messageID(requestID string, sequence int) string {
	return fmt.Sprintf("%s-message-%d", requestID, sequence)
}
