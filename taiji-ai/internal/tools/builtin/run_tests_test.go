package builtin

import (
	"strings"
	"testing"
)

func TestParseGoTestJSONSuccess(t *testing.T) {
	in := `{"Action":"run","Package":"pkg/a","Test":"TestFoo"}
{"Action":"output","Package":"pkg/a","Test":"TestFoo","Output":"=== RUN   TestFoo\n"}
{"Action":"pass","Package":"pkg/a","Test":"TestFoo","Elapsed":0.01}
{"Action":"pass","Package":"pkg/a","Elapsed":0.05}
{"Action":"run","Package":"pkg/b","Test":"TestBar"}
{"Action":"pass","Package":"pkg/b","Test":"TestBar","Elapsed":0.02}
{"Action":"pass","Package":"pkg/b","Elapsed":0.03}
`
	res, err := parseGoTestJSON(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed || res.TestCount != 2 || res.PackageCount != 2 {
		t.Fatalf("unexpected: %#v", res)
	}
	summary := formatGoTestSummary(res)
	if !strings.Contains(summary, "✅ PASS") || !strings.Contains(summary, "2 tests") {
		t.Fatalf("summary: %s", summary)
	}
}

func TestParseGoTestJSONFailure(t *testing.T) {
	in := `{"Action":"run","Package":"pkg/a","Test":"TestFail"}
{"Action":"output","Package":"pkg/a","Test":"TestFail","Output":"    foo_test.go:12: boom\n"}
{"Action":"fail","Package":"pkg/a","Test":"TestFail","Elapsed":0.01}
{"Action":"pass","Package":"pkg/a","Test":"TestOk","Elapsed":0.01}
{"Action":"pass","Package":"pkg/a","Elapsed":0.02}
`
	res, err := parseGoTestJSON(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed || len(res.Failures) != 1 {
		t.Fatalf("unexpected: %#v", res)
	}
	if !strings.Contains(res.Failures[0].Log, "foo_test.go:12") {
		t.Fatalf("log: %s", res.Failures[0].Log)
	}
	summary := formatGoTestSummary(res)
	if strings.Contains(summary, "TestOk") {
		t.Fatalf("should not mention passing test: %s", summary)
	}
}
