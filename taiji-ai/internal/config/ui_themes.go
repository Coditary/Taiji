package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const uiThemesSubdir = "themes"

var themeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// GlobalUIThemesDir returns ~/.config/taiji/themes (or $XDG_CONFIG_HOME/taiji/themes).
func GlobalUIThemesDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "taiji", uiThemesSubdir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "taiji", uiThemesSubdir)
}

// ValidThemeID reports whether id is safe as a theme filename stem.
func ValidThemeID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	return themeIDPattern.MatchString(id)
}
