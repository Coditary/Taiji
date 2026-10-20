package builtin

import (
	"fmt"
	"strings"
)

// PreservationBlockForSession returns todo_plan and checkpoint metadata that must survive compaction.
func PreservationBlockForSession(sessionID string) string {
	sessionID = normalizeSessionID(sessionID)
	var parts []string
	todos := DefaultTodoManager().FormatList(sessionID)
	if todos != "" && todos != "no todos" {
		parts = append(parts, "### todo_plan (must keep)\n"+todos)
	}
	if meta := DefaultWorkspace().LatestCheckpointMeta(sessionID); meta != nil {
		parts = append(parts, fmt.Sprintf("### workspace_checkpoint (must keep)\n%s — %q — %d file(s)",
			meta.ID, meta.Label, len(meta.Files)))
	}
	return strings.Join(parts, "\n\n")
}
