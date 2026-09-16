package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct taiji_session_stats_t {
	int32_t context_tokens;
	int32_t context_window;
	double context_used_pct;
	int32_t input_tokens;
	int32_t output_tokens;
	double estimated_cost_usd;
} taiji_session_stats_t;

typedef struct taiji_turn_metrics_t {
	int32_t total_duration_ms;
	int32_t thinking_duration_ms;
	int32_t output_tokens;
	int32_t generation_duration_ms;
	double tokens_per_sec;
	int32_t had_thinking;
} taiji_turn_metrics_t;

typedef void (*taiji_delta_cb)(const char* delta, void* user_data);
typedef void (*taiji_done_cb)(int ok, const char* error, const char* answer, void* user_data);
typedef void (*taiji_event_cb)(const char* event_json, void* user_data);

static void call_delta(taiji_delta_cb cb, const char* text, void* user) {
	if (cb != NULL) {
		cb(text, user);
	}
}

static void call_done(taiji_done_cb cb, int ok, const char* err, const char* answer, void* user) {
	if (cb != NULL) {
		cb(ok, err, answer, user);
	}
}

static void call_event(taiji_event_cb cb, const char* event_json, void* user) {
	if (cb != NULL) {
		cb(event_json, user);
	}
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unsafe"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
)

type runtimeHandle struct {
	rt      *runtime.Runtime
	cfg     *taijicfg.Config
	session *session.Session
	workDir string

	mu              sync.Mutex
	cancel          context.CancelFunc
	lastTurnMetrics agent.RunMetrics
	onEvent         C.taiji_event_cb
	onEventUser     unsafe.Pointer
}

var (
	handlesMu  sync.Mutex
	handles           = map[uint64]*runtimeHandle{}
	nextHandle uint64 = 1

	initErrMu sync.Mutex
	initErr   string
)

func setInitError(msg string) {
	initErrMu.Lock()
	initErr = msg
	initErrMu.Unlock()
}

func main() {}

//export taiji_init
func taiji_init(workDir *C.char, localMode C.int, errorOut **C.char) C.uint64_t {
	setInitError("")

	dir := strings.TrimSpace(C.GoString(workDir))
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			setInitError(err.Error())
			if errorOut != nil {
				*errorOut = C.CString(err.Error())
			}
			return 0
		}
		dir = wd
	}
	if err := os.Chdir(dir); err != nil {
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}

	cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: localMode != 0})
	if err != nil {
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}

	rt, err := runtime.New(context.Background(), cfg, runtime.Options{Local: localMode != 0})
	if err != nil {
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}

	providerID := strings.TrimSpace(cfg.DefaultProvider)
	sess, err := session.New(cfg.Root, "", providerID, "tui-chat")
	if err != nil {
		rt.Close()
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	if err := session.SaveDirBinding(cfg.Root, dir, sess.State.ID); err != nil {
		rt.Close()
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}

	h := &runtimeHandle{
		rt:      rt,
		cfg:     cfg,
		session: sess,
		workDir: dir,
	}

	handlesMu.Lock()
	id := nextHandle
	nextHandle++
	handles[id] = h
	handlesMu.Unlock()
	return C.uint64_t(id)
}

//export taiji_close
func taiji_close(handle C.uint64_t) {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	if ok {
		delete(handles, uint64(handle))
	}
	handlesMu.Unlock()
	if !ok || h == nil {
		return
	}
	h.mu.Lock()
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
	h.mu.Unlock()
	if h.rt != nil {
		h.rt.Close()
	}
}

//export taiji_set_event_callback
func taiji_set_event_callback(handle C.uint64_t, onEvent C.taiji_event_cb, userData unsafe.Pointer) {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return
	}
	h.mu.Lock()
	h.onEvent = onEvent
	h.onEventUser = userData
	h.mu.Unlock()
}

//export taiji_cancel
func taiji_cancel(handle C.uint64_t) {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return
	}
	h.mu.Lock()
	if h.cancel != nil {
		h.cancel()
	}
	h.mu.Unlock()
}

//export taiji_chat
func taiji_chat(handle C.uint64_t, goal *C.char, provider *C.char, onDelta C.taiji_delta_cb, onDone C.taiji_done_cb, userData unsafe.Pointer) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		cErr := C.CString("invalid taiji handle")
		C.call_done(onDone, 0, cErr, nil, userData)
		C.free(unsafe.Pointer(cErr))
		return 0
	}

	goalText := strings.TrimSpace(C.GoString(goal))
	if goalText == "" {
		cErr := C.CString("goal is required")
		C.call_done(onDone, 0, cErr, nil, userData)
		C.free(unsafe.Pointer(cErr))
		return 0
	}

	providerID := strings.TrimSpace(C.GoString(provider))
	if providerID == "" {
		providerID = strings.TrimSpace(h.cfg.DefaultProvider)
	}
	if providerID == "" && h.session != nil {
		providerID = h.session.State.Provider
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancel = cancel
	h.mu.Unlock()

	answerShown := false
	streamed := false
	sendDelta := func(delta string) error {
		// Keep whitespace-only deltas (especially "\n") — models often stream
		// newlines as their own tokens and Markdown needs them for block structure.
		if delta == "" {
			return nil
		}
		answerShown = true
		streamed = true
		cDelta := C.CString(delta)
		C.call_delta(onDelta, cDelta, userData)
		C.free(unsafe.Pointer(cDelta))
		return nil
	}

	h.mu.Lock()
	onEvent := h.onEvent
	onEventUser := h.onEventUser
	h.mu.Unlock()

	sendEvent := func(ev events.Event) {
		if onEvent == nil {
			return
		}
		raw, err := json.Marshal(ev)
		if err != nil {
			return
		}
		cJSON := C.CString(string(raw))
		C.call_event(onEvent, cJSON, onEventUser)
		C.free(unsafe.Pointer(cJSON))
	}

	result, err := agent.Run(ctx, agent.RunRequest{
		Goal:       goalText,
		ProviderID: providerID,
		Session:    h.session,
		Providers:  h.rt.Providers,
		Tools:      h.rt.Tools,
		Skills:     h.rt.Skills,
		Config:     h.cfg,
		RAG:        h.rt.RAG,
		MCP:        h.rt.MCP,
		WorkDir:    h.workDir,
		EventHandler: func(ev events.Event) {
			sendEvent(ev)
		},
		StreamDelta: sendDelta,
		StreamThinkDelta: func(delta string) error {
			if delta == "" {
				return nil
			}
			sendEvent(events.Event{Type: events.TypeThinkDelta, Text: delta})
			return nil
		},
		StreamThoughtStart: func(thoughtNum int) error {
			sendEvent(events.Event{Type: events.TypeThoughtStart, Step: thoughtNum})
			return nil
		},
	})

	h.mu.Lock()
	h.cancel = nil
	h.mu.Unlock()

	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "cancel") {
			cErr := C.CString("cancelled")
			C.call_done(onDone, 0, cErr, nil, userData)
			C.free(unsafe.Pointer(cErr))
			return 0
		}
		cErr := C.CString(err.Error())
		C.call_done(onDone, 0, cErr, nil, userData)
		C.free(unsafe.Pointer(cErr))
		return 0
	}

	answer := ""
	if result != nil {
		answer = strings.TrimSpace(result.Answer)
		h.mu.Lock()
		h.lastTurnMetrics = result.Metrics
		h.mu.Unlock()
	}
	if !answerShown && answer != "" {
		cAnswer := C.CString(answer)
		C.call_delta(onDelta, cAnswer, userData)
		C.free(unsafe.Pointer(cAnswer))
	} else if answer != "" && !streamed {
		cAnswer := C.CString(answer)
		C.call_delta(onDelta, cAnswer, userData)
		C.free(unsafe.Pointer(cAnswer))
	}

	cAnswer := C.CString(answer)
	C.call_done(onDone, 1, nil, cAnswer, userData)
	C.free(unsafe.Pointer(cAnswer))

	if err := h.session.Save(); err != nil {
		return 1
	}
	maybeAutoNameSession(context.Background(), h, goalText, providerID)
	return 1
}

func maybeAutoNameSession(ctx context.Context, h *runtimeHandle, userMessage, providerID string) {
	if h == nil || h.cfg == nil || !h.cfg.Agent.AutoNameSessionsEnabled() {
		return
	}
	if h.session == nil || !h.session.NeedsAutoName() {
		return
	}
	_, _ = session.AutoName(ctx, h.session, h.rt.Providers, providerID, userMessage)
}

//export taiji_session_label
func taiji_session_label(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return C.CString("")
	}
	return C.CString(h.session.Label())
}

//export taiji_session_name
func taiji_session_name(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return C.CString("")
	}
	return C.CString(strings.TrimSpace(h.session.State.Name))
}

//export taiji_session_id
func taiji_session_id(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return C.CString("")
	}
	return C.CString(h.session.State.ID)
}

func activeProviderID(h *runtimeHandle) string {
	if h == nil {
		return ""
	}
	providerID := ""
	if h.session != nil {
		providerID = strings.TrimSpace(h.session.State.Provider)
	}
	if providerID == "" && h.cfg != nil {
		providerID = strings.TrimSpace(h.cfg.DefaultProvider)
	}
	return providerID
}

func formatProviderLabel(h *runtimeHandle) string {
	if h == nil || h.cfg == nil {
		return ""
	}
	providerID := activeProviderID(h)
	pc, ok := h.cfg.Providers[providerID]
	if !ok {
		if providerID == "" {
			return "Local"
		}
		return capitalizeLabel(providerID) + " Local"
	}
	driverID := strings.TrimSpace(pc.ProviderID)
	if driverID == "" {
		driverID = providerID
	}
	if driverID == "" {
		return "Local"
	}
	return capitalizeLabel(driverID) + " Local"
}

func formatModelName(h *runtimeHandle) string {
	if h == nil || h.cfg == nil {
		return ""
	}
	providerID := activeProviderID(h)
	pc, ok := h.cfg.Providers[providerID]
	if !ok {
		return ""
	}
	return strings.TrimSpace(pc.Model)
}

func capitalizeLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

//export taiji_work_dir
func taiji_work_dir(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("")
	}
	return C.CString(strings.TrimSpace(h.workDir))
}

//export taiji_model_name
func taiji_model_name(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("")
	}
	return C.CString(formatModelName(h))
}

//export taiji_provider_label
func taiji_provider_label(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("")
	}
	return C.CString(formatProviderLabel(h))
}

func formatMCPStatus(h *runtimeHandle) string {
	if h == nil || h.rt == nil || h.rt.MCP == nil {
		return "(nicht verbunden)"
	}
	connected := h.rt.MCP.ConnectedServers()
	if len(connected) == 0 {
		if h.cfg != nil && len(h.cfg.MCP.Servers) > 0 {
			return "(nicht verbunden)"
		}
		return "(nicht konfiguriert)"
	}
	sort.Strings(connected)
	lines := make([]string, 0, len(connected))
	for _, name := range connected {
		lines = append(lines, name+" Connected")
	}
	return strings.Join(lines, "\n")
}

//export taiji_mcp_status
func taiji_mcp_status(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("(nicht verbunden)")
	}
	return C.CString(formatMCPStatus(h))
}

// taiji_mcp_servers returns the names of connected MCP servers, one per line
// (empty string when none). Machine-readable counterpart to taiji_mcp_status.
//
//export taiji_mcp_servers
func taiji_mcp_servers(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.rt == nil || h.rt.MCP == nil {
		return C.CString("")
	}
	connected := h.rt.MCP.ConnectedServers()
	sort.Strings(connected)
	return C.CString(strings.Join(connected, "\n"))
}

//export taiji_last_turn_metrics
func taiji_last_turn_metrics(handle C.uint64_t, out *C.taiji_turn_metrics_t) C.int {
	if out == nil {
		return 0
	}
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return 0
	}

	h.mu.Lock()
	m := h.lastTurnMetrics
	h.mu.Unlock()

	out.total_duration_ms = C.int32_t(m.TotalDurationMs)
	out.thinking_duration_ms = C.int32_t(m.ThinkingDurationMs)
	out.output_tokens = C.int32_t(m.OutputTokens)
	out.generation_duration_ms = C.int32_t(m.GenerationDurationMs)
	out.tokens_per_sec = C.double(m.TokensPerSec)
	out.had_thinking = C.int32_t(boolToInt(m.HadThinking))
	return 1
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

//export taiji_session_stats
func taiji_session_stats(handle C.uint64_t, out *C.taiji_session_stats_t) C.int {
	if out == nil {
		return 0
	}
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return 0
	}

	providerID := strings.TrimSpace(h.session.State.Provider)
	if providerID == "" && h.cfg != nil {
		providerID = strings.TrimSpace(h.cfg.DefaultProvider)
	}
	snap := h.session.ContextSnapshot(h.cfg, providerID)

	out.context_tokens = C.int32_t(snap.ContextTokens)
	out.context_window = C.int32_t(snap.ContextWindow)
	out.context_used_pct = C.double(snap.ContextUsedPct)
	out.input_tokens = C.int32_t(snap.InputTokens)
	out.output_tokens = C.int32_t(snap.OutputTokens)
	out.estimated_cost_usd = C.double(snap.EstimatedCostUSD)
	return 1
}

//export taiji_init_error
func taiji_init_error() *C.char {
	initErrMu.Lock()
	msg := initErr
	initErrMu.Unlock()
	return C.CString(msg)
}
