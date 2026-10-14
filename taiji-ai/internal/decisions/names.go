package decisions

import (
	"path/filepath"
	"strings"
)

// ParseProfileName reads a saved decision profile name from decide tool args.
func ParseProfileName(args map[string]any) string {
	if args == nil {
		return ""
	}
	for _, key := range []string{"profile", "name", "decision"} {
		raw, ok := args[key]
		if !ok {
			continue
		}
		if name, ok := raw.(string); ok {
			name = strings.TrimSpace(name)
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func lookupKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.ReplaceAll(name, "_", "-")
}

// Find returns the profile with the given name.
func Find(all []Profile, name string) (Profile, bool) {
	key := lookupKey(name)
	if key == "" {
		return Profile{}, false
	}
	for _, p := range all {
		if lookupKey(p.Name) == key {
			return p, true
		}
		if dir := filepath.Base(filepath.Dir(p.Path)); lookupKey(dir) == key {
			return p, true
		}
	}
	return Profile{}, false
}
