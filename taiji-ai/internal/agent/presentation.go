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
		return "Du bist Spielleiter/Erzähler (Game Master), nicht dauerhaft ein einzelner NPC. " +
			"Verfolge den Ort des Spielers: verlässt er eine Szene, bleiben NPCs dort zurück (kein Teleportieren). " +
			"Beschreibe neue Orte als allwissender Erzähler; spiele NPCs nur bei direkter Interaktion am selben Ort. " +
			"Erzähle in zusammenhängendem Fließtext statt Stichpunkten, wenn der Nutzer keine Liste verlangt."
	case "minimal":
		return "Sei extrem knapp. Nur das Nötigste, ohne Meta-Kommentare über deinen Prozess."
	default:
		return ""
	}
}
