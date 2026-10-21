package uilocale

import (
	"strings"
	"sync"
)

//go:generate python3 ../../../taiji-tui/scripts/embed_locales.py ../../../taiji-tui/locales /tmp/taiji_ui_locales_embed.cpp --go catalog_gen.go

const catalogSep = "\x1f"

const localePlaceholder = "..."

var (
	catalogOnce sync.Once
	byLocale    map[string]map[string]string
)

func loadCatalog() {
	byLocale = make(map[string]map[string]string)
	for _, entry := range catalogEntries {
		key := entry.module + catalogSep + entry.key
		bucket := byLocale[entry.locale]
		if bucket == nil {
			bucket = make(map[string]string)
			byLocale[entry.locale] = bucket
		}
		bucket[key] = entry.value
	}
}

func catalogBucket(localeID string) map[string]string {
	catalogOnce.Do(loadCatalog)
	id := strings.TrimSpace(localeID)
	if id == "" {
		id = "en"
	}
	if bucket, ok := byLocale[id]; ok {
		return bucket
	}
	return byLocale["en"]
}

// Lookup returns a localized string for module + dotted key (e.g. tools, names.grep).
func Lookup(localeID, module, key string) string {
	module = strings.TrimSpace(module)
	key = strings.TrimSpace(key)
	if module == "" || key == "" {
		return ""
	}
	ck := module + catalogSep + key
	v := catalogBucket(localeID)[ck]
	if v != "" && v != localePlaceholder {
		return v
	}
	if localeID != "" && localeID != "en" {
		return Lookup("en", module, key)
	}
	return ""
}

// PrefixMap returns all keys under module with the given key prefix (e.g. names. → tool id).
func PrefixMap(localeID, module, keyPrefix string) map[string]string {
	module = strings.TrimSpace(module)
	keyPrefix = strings.TrimSpace(keyPrefix)
	out := make(map[string]string)
	bucket := catalogBucket(localeID)
	prefix := module + catalogSep + keyPrefix
	enBucket := catalogBucket("en")
	for ck, value := range bucket {
		if !strings.HasPrefix(ck, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(ck, prefix)
		if suffix == "" || strings.Contains(suffix, ".") {
			continue
		}
		if value == "" || value == localePlaceholder {
			value = enBucket[ck]
		}
		if value == "" {
			continue
		}
		out[suffix] = value
	}
	if len(out) == 0 && localeID != "" && localeID != "en" {
		return PrefixMap("en", module, keyPrefix)
	}
	return out
}
