package session

import (
	"fmt"
	"strings"
)

// CouncilEnabled reports whether multi-actor council mode is active.
func (s *Session) CouncilEnabled() bool {
	if s == nil {
		return false
	}
	return s.State.Settings.CouncilEnabled && len(s.State.Settings.CouncilActors) > 0
}

// CouncilActorsForGoal returns actors that should respond (explicit @mention wins; else full council).
func CouncilActorsForGoal(s *Session, projectRoot, goal string) []string {
	if s == nil {
		return nil
	}
	if id, _ := ResolveActorMention(projectRoot, goal); id != "" {
		return []string{id}
	}
	if !s.CouncilEnabled() {
		return nil
	}
	actors := append([]string(nil), s.State.Settings.CouncilActors...)
	if len(actors) == 0 {
		return nil
	}
	return actors
}

// ParseCouncilCommand handles /council subcommands. Returns reply text when handled.
func ParseCouncilCommand(s *Session, projectRoot, line string) (reply string, handled bool, err error) {
	if s == nil {
		return "", false, nil
	}
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	if lower == "/council" {
		return formatCouncilList(s, projectRoot), true, nil
	}
	if !strings.HasPrefix(lower, "/council ") {
		return "", false, nil
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return formatCouncilList(s, projectRoot), true, nil
	}
	sub := strings.ToLower(fields[1])
	st := &s.State.Settings
	switch sub {
	case "on", "enable":
		st.CouncilEnabled = true
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Council-Modus aktiv.", true, nil
	case "off", "disable":
		st.CouncilEnabled = false
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Council-Modus aus.", true, nil
	case "clear":
		st.CouncilActors = nil
		st.CouncilEnabled = false
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Council geleert.", true, nil
	case "list":
		return formatCouncilList(s, projectRoot), true, nil
	case "add":
		if len(fields) < 3 {
			return "Nutze: /council add <template-id>", true, nil
		}
		id := strings.TrimSpace(fields[2])
		if _, err := LoadChatTemplate(projectRoot, id); err != nil {
			return "", true, fmt.Errorf("template %q: %w", id, err)
		}
		if !containsFold(st.CouncilActors, id) {
			st.CouncilActors = append(st.CouncilActors, id)
		}
		st.CouncilEnabled = true
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Council: %s hinzugefügt.", id), true, nil
	case "remove", "rm":
		if len(fields) < 3 {
			return "Nutze: /council remove <template-id>", true, nil
		}
		id := strings.TrimSpace(fields[2])
		st.CouncilActors = removeFold(st.CouncilActors, id)
		if len(st.CouncilActors) == 0 {
			st.CouncilEnabled = false
		}
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Council: %s entfernt.", id), true, nil
	default:
		return formatCouncilList(s, projectRoot), true, nil
	}
}

func formatCouncilList(s *Session, projectRoot string) string {
	st := s.State.Settings
	var b strings.Builder
	b.WriteString("Council ")
	if st.CouncilEnabled {
		b.WriteString("(an)")
	} else {
		b.WriteString("(aus)")
	}
	b.WriteString("\n")
	if len(st.CouncilActors) == 0 {
		b.WriteString("  (keine Actors — /council add <id>)\n")
	} else {
		for _, id := range st.CouncilActors {
			title := id
			if doc, err := LoadChatTemplate(projectRoot, id); err == nil && strings.TrimSpace(doc.Name) != "" {
				title = doc.Name
			}
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", id, title))
		}
	}
	b.WriteString("Befehle: on | off | add | remove | clear | list\n")
	b.WriteString("Mit @template-id nur einen Actor ansprechen.")
	return strings.TrimSpace(b.String())
}

func containsFold(list []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), want) {
			return true
		}
	}
	return false
}

func removeFold(list []string, want string) []string {
	want = strings.TrimSpace(want)
	var out []string
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), want) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// StreamPacingNatural reports whether the session wants messenger-style typing delay.
func (s *Session) StreamPacingNatural() bool {
	if s == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.State.Settings.StreamPacing)) {
	case "natural", "natural_typing", "typing", "slow":
		return true
	default:
		return false
	}
}
