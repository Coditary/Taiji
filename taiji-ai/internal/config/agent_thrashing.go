package config

const defaultThrashingThreshold = 3

// ThrashingThresholdOrDefault is the consecutive failure count before the circuit breaker fires.
func (a AgentConfig) ThrashingThresholdOrDefault() int {
	if a.ThrashingThreshold <= 0 {
		return defaultThrashingThreshold
	}
	return a.ThrashingThreshold
}

// CacheFriendlyPruningOrDefault reports whether oversized observation pruning runs on fixed step intervals.
func (a AgentConfig) CacheFriendlyPruningOrDefault() bool {
	if a.CacheFriendlyPruning == nil {
		return true
	}
	return *a.CacheFriendlyPruning
}
