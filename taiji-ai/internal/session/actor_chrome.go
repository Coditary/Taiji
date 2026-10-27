package session

import (
	"encoding/json"
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

const actorChromeNamePrefix = "taiji-actor:"

// ActorChrome is persisted UI metadata for who spoke in a chat turn.
type ActorChrome struct {
	TemplateID   string `json:"template_id,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	ThemeColor   string `json:"theme_color,omitempty"`
}

// AuthorLabel formats the TUI assistant header (avatar + display name).
func (c ActorChrome) AuthorLabel() string {
	if c == (ActorChrome{}) {
		return ""
	}
	display := strings.TrimSpace(c.DisplayName)
	if display == "" {
		display = strings.TrimSpace(c.TemplateID)
	}
	if display == "" {
		return ""
	}
	if av := strings.TrimSpace(c.Avatar); av != "" {
		return av + " " + display
	}
	return display
}

// ActorChromeFromSettings builds chrome from the active session template.
func ActorChromeFromSettings(st Settings) ActorChrome {
	return ActorChrome{
		TemplateID:  strings.TrimSpace(st.TemplateID),
		DisplayName: strings.TrimSpace(st.TemplateDisplayName),
		Avatar:      strings.TrimSpace(st.Avatar),
		ThemeColor:  strings.TrimSpace(st.ThemeColor),
	}
}

// EncodeActorChromeName stores chrome in provider.Message.Name for history replay.
func EncodeActorChromeName(chrome ActorChrome) string {
	if chrome.AuthorLabel() == "" {
		return ""
	}
	raw, err := json.Marshal(chrome)
	if err != nil {
		return ""
	}
	return actorChromeNamePrefix + string(raw)
}

// DecodeActorChromeName reads chrome from Message.Name.
func DecodeActorChromeName(name string) (ActorChrome, bool) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, actorChromeNamePrefix) {
		return ActorChrome{}, false
	}
	var chrome ActorChrome
	if err := json.Unmarshal([]byte(name[len(actorChromeNamePrefix):]), &chrome); err != nil {
		return ActorChrome{}, false
	}
	return chrome, true
}

// StampLastAssistantActor records which template spoke for the newest visible reply.
func StampLastAssistantActor(messages []provider.Message, st Settings) {
	chrome := ActorChromeFromSettings(st)
	encoded := EncodeActorChromeName(chrome)
	if encoded == "" {
		return
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != provider.RoleAssistant {
			continue
		}
		if strings.TrimSpace(messages[i].Content) == "" && len(messages[i].ToolCalls) > 0 {
			continue
		}
		if _, ok := DecodeActorChromeName(messages[i].Name); ok {
			return
		}
		messages[i].Name = encoded
		return
	}
}

// ActorChromeFromEventData reads actor_switch / turn event payloads.
func ActorChromeFromEventData(data map[string]any) ActorChrome {
	if len(data) == 0 {
		return ActorChrome{}
	}
	chrome := ActorChrome{}
	if v, ok := data["template_id"].(string); ok {
		chrome.TemplateID = strings.TrimSpace(v)
	}
	if v, ok := data["template_display_name"].(string); ok {
		chrome.DisplayName = strings.TrimSpace(v)
	}
	if v, ok := data["avatar"].(string); ok {
		chrome.Avatar = strings.TrimSpace(v)
	}
	if v, ok := data["theme_color"].(string); ok {
		chrome.ThemeColor = strings.TrimSpace(v)
	}
	return chrome
}
