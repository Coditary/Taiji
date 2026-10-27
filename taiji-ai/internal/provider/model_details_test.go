package provider

import (
	"strings"
	"testing"

	"github.com/coditary/wuji-core/pkg/catalog"
)

func TestFormatModelDetailLinesKnownModel(t *testing.T) {
	lines := FormatModelDetailLines(ModelDetailsPayload{
		Name:             "Test Model",
		ModelID:          "test-1",
		ProviderID:       "openai",
		CatalogKnown:     true,
		ToolCall:         true,
		Reasoning:        false,
		Context:          128000,
		Output:           4096,
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
		CostInputUSD:     3,
		CostOutputUSD:    15,
		Family:           "gpt",
	})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Test Model", "Tool calls        yes", "128k tokens", "Vision / images   yes"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestPayloadFromCatalogDetailsCopiesSlices(t *testing.T) {
	src := catalog.ModelDetails{
		ProviderID: "p", ModelID: "m", Name: "N",
		InputModalities: []string{"text"},
		Notes:           []string{"note"},
	}
	out := payloadFromCatalogDetails(src)
	out.InputModalities[0] = "mutated"
	if src.InputModalities[0] != "text" {
		t.Fatal("expected deep copy of modalities")
	}
}

func TestHasModality(t *testing.T) {
	if !hasModality([]string{"Text", "Image"}, "image") {
		t.Fatal("expected image modality")
	}
	if hasModality(nil, "audio") {
		t.Fatal("expected no match")
	}
}

func TestModalityLineUnknownCatalog(t *testing.T) {
	if got := modalityLine(false, []string{"image"}, "image"); got != "unknown" {
		t.Fatalf("got %q", got)
	}
}
