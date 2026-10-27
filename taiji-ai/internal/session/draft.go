package session

// IsDraft reports whether the session has no visible chat turns yet.
func IsDraft(s *Session) bool {
	if s == nil {
		return false
	}
	return VisibleTurnCount(s.State.Messages) == 0
}
