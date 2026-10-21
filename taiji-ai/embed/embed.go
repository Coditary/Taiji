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
	int32_t usage_system_tokens;
	int32_t usage_tools_tokens;
	int32_t usage_skills_tokens;
	int32_t usage_conversation_tokens;
} taiji_session_stats_t;

typedef struct taiji_turn_metrics_t {
	int32_t total_duration_ms;
	int32_t wait_duration_ms;
	int32_t thinking_duration_ms;
	int32_t output_tokens;
	int32_t generation_duration_ms;
	double tokens_per_sec;
	int32_t had_thinking;
	int32_t input_tokens;
	double estimated_cost_usd;
	int32_t tool_calls;
	int32_t tool_duration_ms;
	int32_t steps;
	int32_t context_tokens;
	int32_t context_window;
	double context_used_pct;
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
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unsafe"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/modelref"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools/builtin"
)

type runtimeHandle struct {
	rt        *runtime.Runtime
	rtMu      sync.RWMutex
	rtReady   bool
	rtInitErr error
	rtInitDone chan struct{}

	cfg       *taijicfg.Config
	session   *session.Session
	workDir   string
	localMode bool

	mu              sync.Mutex
	cancel          context.CancelFunc
	lastTurnMetrics agent.RunMetrics
	activeSkills    []string
	onEvent         C.taiji_event_cb
	onEventUser     unsafe.Pointer

	sttMu            sync.Mutex
	sttStreamCancel  context.CancelFunc
	sttStreamDone    chan struct{}
	sttAccum         string
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

	providerID := strings.TrimSpace(cfg.DefaultProvider)
	sess, resumed, err := resolveEmbedSession(cfg.Root, dir, providerID)
	if err != nil {
		setInitError(err.Error())
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	if !resumed {
		if err := session.SaveDirBinding(cfg.Root, dir, sess.State.ID); err != nil {
			setInitError(err.Error())
			if errorOut != nil {
				*errorOut = C.CString(err.Error())
			}
			return 0
		}
	}

	_ = taijicfg.RegisterProject(cfg.Root, localMode != 0)

	h := &runtimeHandle{
		cfg:          cfg,
		session:      sess,
		workDir:      dir,
		localMode:    localMode != 0,
		activeSkills: skills.ActiveFromMessages(sess.State.Messages),
	}

	handlesMu.Lock()
	id := nextHandle
	nextHandle++
	handles[id] = h
	handlesMu.Unlock()

	h.startRuntimeInit(localMode != 0)
	return C.uint64_t(id)
}

//export taiji_runtime_ready
func taiji_runtime_ready(handle C.uint64_t) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return 0
	}
	if h.runtimeReady() {
		return 1
	}
	return 0
}

//export taiji_runtime_error
func taiji_runtime_error(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("invalid taiji handle")
	}
	err := h.runtimeInitError()
	if err == nil {
		return C.CString("")
	}
	if errors.Is(err, errRuntimeStarting) {
		return C.CString("")
	}
	return C.CString(err.Error())
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
	h.closeRuntime()
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

// isUserCancellation reports Esc/taiji_cancel, not provider gRPC "Canceled" transport errors.
func isUserCancellation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "rpc error") || strings.Contains(lower, "grpc") {
		return false
	}
	return strings.Contains(lower, "cancel")
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
		providerID = h.session.ResolveProviderID(h.cfg)
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

	rt, rtErr := h.getRuntime()
	if rtErr != nil {
		cErr := C.CString(rtErr.Error())
		C.call_done(onDone, 0, cErr, nil, userData)
		C.free(unsafe.Pointer(cErr))
		return 0
	}

	maybeAutoCompactSession(ctx, h, providerID, rt)

	result, err := agent.Run(ctx, agent.RunRequest{
		Goal:       goalText,
		ProviderID: providerID,
		Session:    h.session,
		Providers:  rt.Providers,
		Tools:      rt.Tools,
		Skills:        rt.Skills,
		SkillsCatalog: rt.SkillsCatalog,
		Config:        h.cfg,
		RAG:        rt.RAG,
		Memory:     rt.Memory,
		MCP:                rt.MCP,
		DecisionProfiles:   rt.DecisionProfiles,
		Hooks:              rt.Hooks,
		Questions:          rt.Questions,
		WorkDir:    h.workDir,
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeObservation && ev.IsSkill && !strings.HasPrefix(strings.TrimSpace(ev.Text), "error:") {
				h.mu.Lock()
				if ev.Data != nil {
					if raw, ok := ev.Data["skills"]; ok {
						switch skillsList := raw.(type) {
						case []string:
							for _, name := range skillsList {
								h.addActiveSkill(name)
							}
						case []any:
							for _, item := range skillsList {
								if name, ok := item.(string); ok {
									h.addActiveSkill(name)
								}
							}
						}
					}
				}
				if name := strings.TrimSpace(ev.SkillName); name != "" {
					for _, part := range strings.Split(name, ",") {
						h.addActiveSkill(strings.TrimSpace(part))
					}
				}
				h.mu.Unlock()
			}
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
		if isUserCancellation(err) {
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
		if h.session != nil {
			h.activeSkills = skills.ActiveFromMessages(h.session.State.Messages)
		}
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
	rt, err := h.getRuntime()
	if err != nil {
		return
	}
	_, _ = session.AutoName(ctx, h.session, rt.Providers, providerID, userMessage)
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
	if h.session != nil {
		if model := strings.TrimSpace(h.session.State.Settings.Model); model != "" {
			return modelref.Display(model)
		}
	}
	pc, ok := h.cfg.Providers[providerID]
	if !ok {
		return ""
	}
	return modelref.Display(strings.TrimSpace(pc.Model))
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
	if h == nil {
		return "(not connected)"
	}
	rt, err := h.getRuntime()
	if errors.Is(err, errRuntimeStarting) {
		return "(connecting…)"
	}
	if err != nil || rt == nil || rt.MCP == nil {
		return "(not connected)"
	}
	connected := rt.MCP.ConnectedServers()
	if len(connected) == 0 {
		if h.cfg != nil && len(h.cfg.MCP.Servers) > 0 {
			return "(not connected)"
		}
		return "(not configured)"
	}
	sort.Strings(connected)
	lines := make([]string, 0, len(connected))
	for _, name := range connected {
		lines = append(lines, name)
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
	if !ok || h == nil {
		return C.CString("")
	}
	rt, err := h.getRuntime()
	if err != nil || rt == nil || rt.MCP == nil {
		return C.CString("")
	}
	connected := rt.MCP.ConnectedServers()
	sort.Strings(connected)
	return C.CString(strings.Join(connected, "\n"))
}

func (h *runtimeHandle) addActiveSkill(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for _, existing := range h.activeSkills {
		if existing == name {
			return
		}
	}
	h.activeSkills = append(h.activeSkills, name)
	sort.Strings(h.activeSkills)
}

func activeSkillsList(h *runtimeHandle) []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.activeSkills) > 0 {
		return append([]string(nil), h.activeSkills...)
	}
	if h.session != nil {
		return skills.ActiveFromMessages(h.session.State.Messages)
	}
	return nil
}

//export taiji_active_skills
func taiji_active_skills(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("")
	}
	names := activeSkillsList(h)
	return C.CString(strings.Join(names, "\n"))
}

//export taiji_skills_list
func taiji_skills_list(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, _ := handles[uint64(handle)]
	handlesMu.Unlock()
	return C.CString(formatSkillsList(h))
}

//export taiji_decisions_list
func taiji_decisions_list(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, _ := handles[uint64(handle)]
	handlesMu.Unlock()
	return C.CString(formatDecisionsList(h))
}

func formatDecisionsList(h *runtimeHandle) string {
	if h == nil {
		return "Keine gespeicherten Decisions konfiguriert."
	}
	rt, rtErr := h.getRuntime()
	if errors.Is(rtErr, errRuntimeStarting) {
		return "Decisions werden geladen…"
	}
	if rtErr != nil || rt == nil || rt.DecisionProfiles == nil {
		return "Keine gespeicherten Decisions konfiguriert."
	}
	all, err := rt.DecisionProfiles.LoadAll()
	if err != nil {
		return "Decisions laden fehlgeschlagen: " + err.Error()
	}
	return decisions.FormatList(all)
}

func formatSkillsList(h *runtimeHandle) string {
	if h == nil {
		return "Keine Skills konfiguriert."
	}
	rt, rtErr := h.getRuntime()
	if errors.Is(rtErr, errRuntimeStarting) {
		return "Skills werden geladen…"
	}
	if rtErr != nil || rt == nil || rt.Skills == nil {
		return "Keine Skills konfiguriert."
	}
	all, err := rt.Skills.LoadAll()
	if err != nil {
		return "Skills laden fehlgeschlagen: " + err.Error()
	}
	invokable := skills.Invokable(all)
	if len(invokable) == 0 {
		return "Keine aufrufbaren Skills gefunden.\nLege Skills unter ~/.config/taiji/skills/<name>/SKILL.md ab."
	}
	sort.Slice(invokable, func(i, j int) bool { return invokable[i].Name < invokable[j].Name })
	var b strings.Builder
	b.WriteString("Verfügbare Skills:\n")
	for _, s := range invokable {
		line := "- " + s.Name
		if desc := strings.TrimSpace(s.Description); desc != "" {
			line += ": " + desc
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func turnMetricsValuesFromC(out *C.taiji_turn_metrics_t) taijicfg.TurnMetricsValues {
	if out == nil {
		return taijicfg.TurnMetricsValues{}
	}
	return taijicfg.TurnMetricsValues{
		TotalDurationMs:      int64(out.total_duration_ms),
		WaitDurationMs:       int64(out.wait_duration_ms),
		ThinkingDurationMs:   int64(out.thinking_duration_ms),
		GenerationDurationMs: int64(out.generation_duration_ms),
		OutputTokens:         int(out.output_tokens),
		InputTokens:          int(out.input_tokens),
		EstimatedCostUSD:     float64(out.estimated_cost_usd),
		TokensPerSec:         float64(out.tokens_per_sec),
		HadThinking:          out.had_thinking != 0,
		ToolCalls:            int(out.tool_calls),
		ToolDurationMs:       int64(out.tool_duration_ms),
		Steps:                int(out.steps),
		ContextTokens:        int(out.context_tokens),
		ContextWindow:        int(out.context_window),
		ContextUsedPct:       float64(out.context_used_pct),
	}
}

//export taiji_format_turn_metrics
func taiji_format_turn_metrics(handle C.uint64_t, out *C.taiji_turn_metrics_t, maxWidth C.int32_t) *C.char {
	if out == nil {
		return C.CString("")
	}
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return C.CString("")
	}
	ui := effectiveUI(h)
	lines := taijicfg.FormatTurnMetricsLines(turnMetricsValuesFromC(out), ui.TurnMetrics, int(maxWidth), ui.LocaleOrDefault())
	return C.CString(strings.Join(lines, "\n"))
}

//export taiji_ui_set_display_locales
func taiji_ui_set_display_locales(handle C.uint64_t, uiLocale *C.char, toolLabelsLocale *C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return 0
	}
	ui := taijicfg.NormalizeUILocale(C.GoString(uiLocale))
	h.cfg.UI.Locale = ui
	h.cfg.UI.ToolLabels.Locale = ui
	return 1
}

//export taiji_tool_display_title
func taiji_tool_display_title(handle C.uint64_t, toolName *C.char) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return C.CString("")
	}
	ui := effectiveUI(h)
	cfgCopy := *h.cfg
	cfgCopy.UI = ui
	return C.CString(cfgCopy.ToolDisplayTitle(C.GoString(toolName)))
}

//export taiji_tool_scope_label
func taiji_tool_scope_label(handle C.uint64_t, scope *C.char) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return C.CString("Tool")
	}
	ui := effectiveUI(h)
	cfgCopy := *h.cfg
	cfgCopy.UI = ui
	return C.CString(cfgCopy.ToolScopeLabel(C.GoString(scope)))
}

//export taiji_turn_metrics_show_actions
func taiji_turn_metrics_show_actions(handle C.uint64_t) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return 1
	}
	ui := effectiveUI(h)
	if ui.TurnMetrics.ShowActionsOrDefault() {
		return 1
	}
	return 0
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
	out.wait_duration_ms = C.int32_t(m.WaitDurationMs)
	out.thinking_duration_ms = C.int32_t(m.ThinkingDurationMs)
	out.output_tokens = C.int32_t(m.OutputTokens)
	out.generation_duration_ms = C.int32_t(m.GenerationDurationMs)
	out.tokens_per_sec = C.double(m.TokensPerSec)
	out.had_thinking = C.int32_t(boolToInt(m.HadThinking))
	out.input_tokens = C.int32_t(m.InputTokens)
	out.estimated_cost_usd = C.double(m.EstimatedCostUSD)
	out.tool_calls = C.int32_t(m.ToolCalls)
	out.tool_duration_ms = C.int32_t(m.ToolDurationMs)
	out.steps = C.int32_t(m.Steps)
	out.context_tokens = C.int32_t(m.ContextTokens)
	out.context_window = C.int32_t(m.ContextWindow)
	out.context_used_pct = C.double(m.ContextUsedPct)
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

	usage := session.ContextUsageBreakdown{}
	if rt, rtErr := h.getRuntime(); rtErr == nil && rt != nil {
		h.mu.Lock()
		activeSkills := append([]string(nil), h.activeSkills...)
		h.mu.Unlock()
		usage = agent.EstimateContextUsage(agent.RunRequest{
			Session:          h.session,
			Config:           h.cfg,
			Tools:            rt.Tools,
			Skills:           rt.Skills,
			SkillsCatalog:    rt.SkillsCatalog,
			RAG:              rt.RAG,
			Memory:           rt.Memory,
			MCP:              rt.MCP,
			DecisionProfiles: rt.DecisionProfiles,
			WorkDir:          h.workDir,
		}, activeSkills)
	}
	if snap.ContextTokens > 0 {
		usage = session.ScaleUsageBreakdown(usage, snap.ContextTokens)
	}
	out.usage_system_tokens = C.int32_t(usage.SystemTokens)
	out.usage_tools_tokens = C.int32_t(usage.ToolsTokens)
	out.usage_skills_tokens = C.int32_t(usage.SkillsTokens)
	out.usage_conversation_tokens = C.int32_t(usage.ConversationTokens)
	return 1
}

//export taiji_init_error
func taiji_init_error() *C.char {
	initErrMu.Lock()
	msg := initErr
	initErrMu.Unlock()
	return C.CString(msg)
}

func resolveEmbedSession(root, workDir, providerID string) (*session.Session, bool, error) {
	if binding, ok, err := session.LoadDirBinding(root, workDir); err != nil {
		return nil, false, err
	} else if ok {
		sess, err := session.Load(root, binding.SessionID)
		if err == nil {
			return sess, true, nil
		}
	}
	sess, err := session.New(root, "", providerID, "tui-chat")
	if err != nil {
		return nil, false, err
	}
	return sess, false, nil
}

func (h *runtimeHandle) currentSessionID() string {
	if h == nil || h.session == nil {
		return ""
	}
	return strings.TrimSpace(h.session.State.ID)
}

func (h *runtimeHandle) setSession(sess *session.Session) {
	if h == nil {
		return
	}
	h.session = sess
	if sess != nil {
		h.activeSkills = skills.ActiveFromMessages(sess.State.Messages)
	} else {
		h.activeSkills = nil
	}
}

func (h *runtimeHandle) reloadRuntime(ctx context.Context, root string) error {
	if h == nil {
		return fmt.Errorf("runtime handle is nil")
	}
	h.closeRuntime()
	cfg, err := taijicfg.LoadAt(root)
	if err != nil {
		return err
	}
	rt, err := runtime.New(ctx, cfg, runtime.Options{Local: h.localMode, Quiet: true})
	if err != nil {
		return err
	}
	if rt.Questions != nil {
		rt.Questions.SetPrompter(nil)
	}
	h.cfg = cfg
	h.rtMu.Lock()
	h.rt = rt
	h.rtInitErr = nil
	h.rtReady = true
	h.rtMu.Unlock()
	return nil
}

//export taiji_projects_list
func taiji_projects_list(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		return C.CString("{}")
	}
	sidebar, err := session.BuildSidebar(h.cfg.Root, h.localMode, h.currentSessionID())
	if err != nil {
		return C.CString("{}")
	}
	raw, err := session.MarshalSidebarJSON(sidebar)
	if err != nil {
		return C.CString("{}")
	}
	return C.CString(raw)
}

//export taiji_session_turns
func taiji_session_turns(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return C.CString("[]")
	}
	var all []events.Event
	if h.session.Events != nil {
		if loaded, err := events.ReadFile(h.session.Events.Path()); err == nil {
			all = loaded
		}
	}
	turns := session.TurnsWithActivity(h.session.State.Messages, all)
	raw, err := session.MarshalTurnsJSON(turns)
	if err != nil {
		return C.CString("[]")
	}
	return C.CString(raw)
}

func maybeAutoCompactSession(ctx context.Context, h *runtimeHandle, providerID string, rt *runtime.Runtime) {
	if h == nil || h.cfg == nil || h.session == nil || rt == nil {
		return
	}
	if agent.ShouldAutoCompactHistory(h.session.State.Messages, h.cfg) {
		_, _ = agent.CompactSessionHistory(ctx, h.session, rt.Providers, providerID, h.cfg)
		return
	}
	threshold := h.cfg.UI.AutoCompactContextPct
	if threshold <= 0 {
		return
	}
	snap := h.session.ContextSnapshot(h.cfg, providerID)
	if snap.ContextUsedPct < float64(threshold) {
		return
	}
	_, _ = agent.CompactSessionHistory(ctx, h.session, rt.Providers, providerID, h.cfg)
}

//export taiji_session_compact
func taiji_session_compact(handle C.uint64_t, errorOut **C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		if errorOut != nil {
			*errorOut = C.CString("invalid taiji handle")
		}
		return 0
	}

	rt, rtErr := h.getRuntime()
	if rtErr != nil {
		if errorOut != nil {
			*errorOut = C.CString(rtErr.Error())
		}
		return 0
	}

	providerID := strings.TrimSpace(h.cfg.DefaultProvider)
	if providerID == "" {
		providerID = h.session.ResolveProviderID(h.cfg)
	}

	_, err := agent.CompactSessionHistory(context.Background(), h.session, rt.Providers, providerID, h.cfg)
	if err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	return 1
}

//export taiji_session_hud_line
func taiji_session_hud_line(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return nil
	}
	line := builtin.FormatSessionHUDLine(builtin.GetSessionHUD(h.session.State.ID))
	if strings.TrimSpace(line) == "" {
		return nil
	}
	return C.CString(line)
}

//export taiji_session_switch
func taiji_session_switch(handle C.uint64_t, projectRoot *C.char, sessionID *C.char, errorOut **C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		if errorOut != nil {
			*errorOut = C.CString("invalid taiji handle")
		}
		return 0
	}

	root := strings.TrimSpace(C.GoString(projectRoot))
	if root == "" && h.cfg != nil {
		root = h.cfg.Root
	}
	id := strings.TrimSpace(C.GoString(sessionID))
	if id == "" {
		if errorOut != nil {
			*errorOut = C.CString("session id is required")
		}
		return 0
	}

	if h.cfg != nil && root != h.cfg.Root {
		if err := h.reloadRuntime(context.Background(), root); err != nil {
			if errorOut != nil {
				*errorOut = C.CString(err.Error())
			}
			return 0
		}
	}

	sess, err := session.Load(root, id)
	if err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	if err := session.SaveDirBinding(root, h.workDir, sess.State.ID); err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	h.setSession(sess)
	_ = taijicfg.RegisterProject(root, h.localMode)
	return 1
}

//export taiji_session_new
func taiji_session_new(handle C.uint64_t, projectRoot *C.char, errorOut **C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		if errorOut != nil {
			*errorOut = C.CString("invalid taiji handle")
		}
		return 0
	}

	root := strings.TrimSpace(C.GoString(projectRoot))
	if root == "" {
		root = h.cfg.Root
	}
	if root != h.cfg.Root {
		if err := h.reloadRuntime(context.Background(), root); err != nil {
			if errorOut != nil {
				*errorOut = C.CString(err.Error())
			}
			return 0
		}
	}

	providerID := strings.TrimSpace(h.cfg.DefaultProvider)
	if providerID == "" && h.session != nil {
		providerID = strings.TrimSpace(h.session.State.Provider)
	}
	sess, err := session.New(root, "", providerID, "tui-chat")
	if err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	if err := session.SaveDirBinding(root, h.workDir, sess.State.ID); err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	h.setSession(sess)
	_ = taijicfg.RegisterProject(root, h.localMode)
	return 1
}

//export taiji_session_delete
func taiji_session_delete(handle C.uint64_t, projectRoot *C.char, sessionID *C.char, errorOut **C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil {
		if errorOut != nil {
			*errorOut = C.CString("invalid taiji handle")
		}
		return 0
	}

	root := strings.TrimSpace(C.GoString(projectRoot))
	if root == "" {
		root = h.cfg.Root
	}
	id := strings.TrimSpace(C.GoString(sessionID))
	if id == "" {
		if errorOut != nil {
			*errorOut = C.CString("session id is required")
		}
		return 0
	}

	deletingCurrent := h.session != nil && h.session.State.ID == id && root == h.cfg.Root
	if err := session.Delete(root, id); err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}

	if !deletingCurrent {
		return 1
	}

	providerID := strings.TrimSpace(h.cfg.DefaultProvider)
	if providerID == "" && h.session != nil {
		providerID = strings.TrimSpace(h.session.State.Provider)
	}
	sess, err := session.New(h.cfg.Root, "", providerID, "tui-chat")
	if err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	if err := session.SaveDirBinding(h.cfg.Root, h.workDir, sess.State.ID); err != nil {
		if errorOut != nil {
			*errorOut = C.CString(err.Error())
		}
		return 0
	}
	h.setSession(sess)
	return 1
}

//export taiji_question_reply
func taiji_question_reply(handle C.uint64_t, requestID *C.char, answersJSON *C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return 0
	}
	rt, err := h.getRuntime()
	if err != nil || rt == nil || rt.Questions == nil {
		return 0
	}
	id := strings.TrimSpace(C.GoString(requestID))
	if id == "" {
		return 0
	}
	raw := strings.TrimSpace(C.GoString(answersJSON))
	if raw == "" {
		return 0
	}
	var answers []question.Answer
	if err := json.Unmarshal([]byte(raw), &answers); err != nil {
		return 0
	}
	if err := rt.Questions.Reply(question.ReplyInput{RequestID: id, Answers: answers}); err != nil {
		return 0
	}
	return 1
}

//export taiji_question_reject
func taiji_question_reject(handle C.uint64_t, requestID *C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return 0
	}
	rt, err := h.getRuntime()
	if err != nil || rt == nil || rt.Questions == nil {
		return 0
	}
	id := strings.TrimSpace(C.GoString(requestID))
	if id == "" {
		return 0
	}
	if err := rt.Questions.Reject(id); err != nil {
		return 0
	}
	return 1
}

//export taiji_settings_get
func taiji_settings_get(handle C.uint64_t, scope *C.char) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString("{}")
	}
	if h.cfg != nil {
		res, err := provider.RefreshProvidersFromWuji(context.Background(), h.cfg)
		if err == nil && res.Connected && (res.Added > 0 || res.Updated > 0) {
			_ = h.cfg.Save()
		}
	}
	payload := buildSettingsPayload(h, C.GoString(scope))
	raw, err := json.Marshal(payload)
	if err != nil {
		return C.CString("{}")
	}
	return C.CString(string(raw))
}

//export taiji_settings_apply
func taiji_settings_apply(handle C.uint64_t, jsonPayload *C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return 0
	}
	var payload settingsPayload
	if err := json.Unmarshal([]byte(C.GoString(jsonPayload)), &payload); err != nil {
		return 0
	}
	if err := applySettingsPayload(h, payload); err != nil {
		return 0
	}
	return 1
}

func (h *runtimeHandle) emitSTTPartial(text string) {
	if h == nil || strings.TrimSpace(text) == "" {
		return
	}
	h.mu.Lock()
	onEvent := h.onEvent
	onEventUser := h.onEventUser
	h.mu.Unlock()
	if onEvent == nil {
		return
	}
	raw, err := json.Marshal(map[string]string{
		"type": "stt_partial",
		"text": text,
	})
	if err != nil {
		return
	}
	cJSON := C.CString(string(raw))
	C.call_event(onEvent, cJSON, onEventUser)
	C.free(unsafe.Pointer(cJSON))
}

//export taiji_stt_toggle
func taiji_stt_toggle(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil {
		return C.CString(`{"error":"runtime not ready"}`)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out := h.sttToggle(ctx)
	raw, err := json.Marshal(out)
	if err != nil {
		return C.CString(`{"error":"encode result"}`)
	}
	return C.CString(string(raw))
}

//export taiji_template_apply
func taiji_template_apply(handle C.uint64_t, templateID *C.char) C.int {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.cfg == nil || h.session == nil {
		return 0
	}
	id := strings.TrimSpace(C.GoString(templateID))
	if id == "" {
		return 0
	}
	if err := session.ApplyChatTemplate(h.session, h.cfg.Root, id); err != nil {
		return 0
	}
	return 1
}

//export taiji_presentation_mode
func taiji_presentation_mode(handle C.uint64_t) *C.char {
	handlesMu.Lock()
	h, ok := handles[uint64(handle)]
	handlesMu.Unlock()
	if !ok || h == nil || h.session == nil {
		return C.CString("markdown")
	}
	return C.CString(h.session.PresentationMode())
}
