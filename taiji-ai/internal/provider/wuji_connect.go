package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Linux AF_UNIX pathname limit (sun_path, excluding NUL).
const unixSocketPathMax = 107

// ValidateUnixSocketPath reports whether path fits in a unix socket address.
func ValidateUnixSocketPath(path string) error {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		abs = strings.TrimSpace(path)
	}
	if len(abs) > unixSocketPathMax {
		return fmt.Errorf(
			"unix socket path too long (%d > %d bytes): %s\n"+
				"Set wuji.endpoint or WUJI_CORE_ADDR to your running Wuji core socket (e.g. $WUJI_ROOT/.wuji/run/drivers/core.sock)",
			len(abs), unixSocketPathMax, abs,
		)
	}
	return nil
}

// FormatWujiConnectError adds actionable hints for common connection failures.
func FormatWujiConnectError(endpoint string, err error) error {
	if err == nil {
		return nil
	}
	sock := unixSocketPathFromEndpoint(endpoint)
	var extra string
	if sock != "" {
		if vErr := ValidateUnixSocketPath(sock); vErr != nil {
			extra = vErr.Error()
		} else if _, statErr := os.Stat(sock); os.IsNotExist(statErr) {
			extra = fmt.Sprintf(
				"socket not found: %s\n"+
					"Install wuji/wuji-core to PATH, or set WUJI_CORE_BIN / WUJI_ROOT / WUJI_CORE_ADDR",
				sock,
			)
		}
	}
	if extra == "" {
		return fmt.Errorf("connect wuji core at %s: %w", endpoint, err)
	}
	return fmt.Errorf("connect wuji core at %s: %w\n%s", endpoint, err, extra)
}

func unixSocketPathFromEndpoint(endpoint string) string {
	ep := strings.TrimSpace(endpoint)
	const prefix = "unix://"
	if !strings.HasPrefix(ep, prefix) {
		return ""
	}
	return strings.TrimPrefix(ep, prefix)
}
