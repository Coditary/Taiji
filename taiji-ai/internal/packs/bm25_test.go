package packs

import "testing"

func TestSearchBM25RanksMatchingNode(t *testing.T) {
	nodes := []PackNode{
		{PackName: "lore", RelPath: "a.md", Body: "dragon fire mountain quest", Meta: NodeFrontmatter{Injection: ModeKeyword}},
		{PackName: "lore", RelPath: "b.md", Body: "accounting spreadsheet tax forms", Meta: NodeFrontmatter{Injection: ModeKeyword}},
	}
	hits := SearchBM25(nodes, "dragon quest adventure", 0)
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Node.RelPath != "a.md" {
		t.Fatalf("top=%q", hits[0].Node.RelPath)
	}
}

func TestKeywordExactHits(t *testing.T) {
	nodes := []PackNode{
		{PackName: "p", RelPath: "k.md", Meta: NodeFrontmatter{Injection: ModeKeyword, Keywords: []string{"Diebe"}}},
		{PackName: "p", RelPath: "other.md", Meta: NodeFrontmatter{Injection: ModeKeyword, Keywords: []string{"Bäcker"}}},
	}
	hits := KeywordExactHits(nodes, "Die Diebe kamen.")
	if len(hits) != 1 || hits[0].RelPath != "k.md" {
		t.Fatalf("hits=%v", hits)
	}
}

func TestTokenizeSkipsShortTokens(t *testing.T) {
	terms := tokenize("a bb cc-dd")
	if len(terms) != 3 || terms[0] != "bb" {
		t.Fatalf("terms=%v", terms)
	}
}
