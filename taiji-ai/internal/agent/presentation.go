package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/session"
)

func sessionPresentationPreset(s *session.Session) string {
	if s == nil {
		return ""
	}
	return s.PresentationMode()
}

func presentationPromptAppendix(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "whatsapp", "bubbles", "chat":
		return "Antworte kurz und gesprächig wie in einem Messenger. Keine langen Einleitungen, keine Überschriften unless nötig. Ein Gedanke pro Nachricht."
	case "story", "narrative":
		return "Erzähle in einem zusammenhängenden, literarischen Ton. Fließtext statt Stichpunkte, wenn der Nutzer nicht explizit Listen will."
	case "minimal":
		return "Sei extrem knapp. Nur das Nötigste, ohne Meta-Kommentare über deinen Prozess."
	default:
		return ""
	}
}
