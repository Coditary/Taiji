package decision

import (
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// NewService builds a decision service from Taiji config.
func NewService(cfg *taijicfg.Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	decCfg := cfg.ResolvedDecision()
	if !decCfg.EnabledOrDefault() {
		return nil, ErrDisabled
	}
	backendName := strings.ToLower(strings.TrimSpace(decCfg.Backend))
	if backendName == "" {
		backendName = "wuji"
	}
	driverID := strings.TrimSpace(decCfg.Driver)
	if driverID == "" {
		driverID = taijicfg.DefaultDecisionConfig().Driver
	}
	var backend Backend
	switch backendName {
	case "wuji":
		backend = newWujiBackend(cfg, driverID, decCfg.Model)
	case "http":
		backend = newHTTPBackend(decCfg.HTTPURL, driverID, decCfg.Model)
	default:
		return nil, fmt.Errorf("unsupported decision backend %q (supported: wuji, http)", decCfg.Backend)
	}
	return &Service{
		Backend:     backend,
		DriverID:    driverID,
		BackendName: backendName,
	}, nil
}
