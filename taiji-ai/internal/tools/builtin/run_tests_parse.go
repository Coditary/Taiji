package builtin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type goTestParseResult struct {
	Passed         bool
	TestCount      int
	PackageCount   int
	ElapsedSeconds float64
	Failures       []goTestFailure
}

type goTestFailure struct {
	Label string
	Log   string
}

func parseGoTestJSON(r io.Reader) (goTestParseResult, error) {
	res := goTestParseResult{Passed: true}
	packages := map[string]bool{}
	failLogs := map[string]string{}
	failed := map[string]bool{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev goTestEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Package != "" {
			packages[ev.Package] = true
		}
		switch ev.Action {
		case "pass":
			if ev.Test != "" {
				res.TestCount++
			} else if ev.Elapsed > 0 {
				res.ElapsedSeconds += ev.Elapsed
			}
		case "fail":
			res.Passed = false
			if ev.Test != "" {
				res.TestCount++
			}
			key := failureKey(ev.Package, ev.Test)
			if key != "" {
				failed[key] = true
				if _, ok := failLogs[key]; !ok {
					failLogs[key] = ""
				}
			}
		case "output":
			if ev.Output == "" {
				continue
			}
			key := failureKey(ev.Package, ev.Test)
			if key == "" {
				continue
			}
			failLogs[key] += ev.Output
		}
	}
	if err := sc.Err(); err != nil {
		return res, err
	}
	res.PackageCount = len(packages)
	if res.ElapsedSeconds <= 0 {
		res.ElapsedSeconds = 0
	}
	keys := make([]string, 0, len(failed))
	for key := range failed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		log := strings.TrimSpace(failLogs[key])
		res.Failures = append(res.Failures, goTestFailure{
			Label: formatFailureLabel(key),
			Log:   log,
		})
	}
	if len(res.Failures) > 0 {
		res.Passed = false
	}
	return res, nil
}

func failureKey(pkg, test string) string {
	pkg = strings.TrimSpace(pkg)
	test = strings.TrimSpace(test)
	if pkg == "" {
		return ""
	}
	if test == "" {
		return pkg + "\t"
	}
	return pkg + "\t" + test
}

func formatFailureLabel(key string) string {
	pkg, test, _ := strings.Cut(key, "\t")
	if strings.TrimSpace(test) == "" {
		return "FAIL: package " + pkg
	}
	return fmt.Sprintf("FAIL: %s (%s)", test, pkg)
}

func formatGoTestSummary(res goTestParseResult) string {
	if res.Passed {
		return fmt.Sprintf("✅ PASS: %d tests across %d packages (%.2fs)",
			res.TestCount, res.PackageCount, res.ElapsedSeconds)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("❌ FAIL: %d failed of %d tests across %d packages (%.2fs)\n",
		len(res.Failures), res.TestCount, res.PackageCount, res.ElapsedSeconds))
	for i, f := range res.Failures {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.Label)
		b.WriteString("\n")
		b.WriteString(f.Log)
	}
	return strings.TrimSpace(b.String())
}

func parseTestTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 30 * time.Second, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("timeout must be positive")
	}
	return d, nil
}
