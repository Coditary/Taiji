package agent

import "github.com/coditary/taiji/internal/tools/builtin"

// SessionHUD re-exports the live session snapshot for TUI and embed clients.
type SessionHUD = builtin.SessionHUD

// GetSessionHUD returns cached todo, job, and checkpoint status for a session.
func GetSessionHUD(sessionID string) SessionHUD {
	return builtin.GetSessionHUD(sessionID)
}

// FormatSessionHUDLine renders a compact status footer line.
func FormatSessionHUDLine(h SessionHUD) string {
	return builtin.FormatSessionHUDLine(h)
}
