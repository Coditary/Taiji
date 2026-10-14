package hooks

import (
	"path/filepath"
	"strings"
)

func entryMatchesEvent(entry Entry, eventName, toolName string) bool {
	eventName = strings.TrimSpace(eventName)
	entryEvent := strings.TrimSpace(entry.Event)
	if entryEvent == "" {
		return false
	}
	if !eventMatches(entryEvent, eventName, toolName) {
		return false
	}
	if m := strings.TrimSpace(entry.Matcher); m != "" && toolName != "" {
		if !matcherMatches(m, toolName) {
			return false
		}
	}
	return true
}

func eventMatches(entryEvent, eventName, toolName string) bool {
	if entryEvent == eventName {
		return true
	}
	switch entryEvent {
	case EventToolBefore:
		return strings.HasPrefix(eventName, EventToolBefore+".") || eventName == EventToolBefore
	case EventToolAfter:
		return strings.HasPrefix(eventName, EventToolAfter+".") || eventName == EventToolAfter
	case EventFileChanged:
		return eventName == EventFileChanged
	default:
		return entryEvent == eventName
	}
}

func matcherMatches(matcher, toolName string) bool {
	matcher = strings.TrimSpace(matcher)
	if matcher == "" {
		return true
	}
	parts := strings.Split(matcher, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.EqualFold(part, toolName) {
			return true
		}
	}
	return false
}

func conditionsMatch(conds []Condition, toolName string, files []string) bool {
	if len(conds) == 0 {
		return true
	}
	for _, cond := range conds {
		if t := strings.TrimSpace(cond.Tool); t != "" && t != toolName {
			continue
		}
		if glob := strings.TrimSpace(cond.MatchesAnyPath); glob != "" {
			if !anyPathMatches(glob, files) {
				continue
			}
		}
		return true
	}
	return false
}

func anyPathMatches(glob string, files []string) bool {
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if file == "" {
			continue
		}
		ok, err := filepath.Match(glob, file)
		if err == nil && ok {
			return true
		}
		ok, err = filepath.Match(glob, filepath.Base(file))
		if err == nil && ok {
			return true
		}
	}
	return false
}

func toolEventName(base, toolName string) string {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return base
	}
	return base + "." + toolName
}
