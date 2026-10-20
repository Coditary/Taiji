package config

// PruneStaleObservationsOrDefault reports whether tool observations should be compacted before model calls.
func (a AgentConfig) PruneStaleObservationsOrDefault() bool {
	if a.PruneStaleObservations == nil {
		return true
	}
	return *a.PruneStaleObservations
}
