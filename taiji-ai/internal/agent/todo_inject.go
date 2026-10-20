package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/tools/builtin"
)

func todoPlanAppendix(sessionID string) string {
	return strings.TrimSpace(builtin.DefaultTodoManager().OpenReminder(sessionID))
}
