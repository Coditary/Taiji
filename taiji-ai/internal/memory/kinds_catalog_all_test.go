package memory

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestKindCatalogAllAndFailureKinds(t *testing.T) {
	catalog := LoadKindCatalog(&taijicfg.Config{Root: findRepoRoot(t)})
	all := catalog.All()
	if len(all) == 0 {
		t.Fatal("expected kinds")
	}
	names := catalog.Names()
	if len(names) != len(all) {
		t.Fatalf("names=%d all=%d", len(names), len(all))
	}
	fk := catalog.FailureKinds()
	if len(fk) == 0 {
		t.Fatal("failure kinds")
	}
	ern := catalog.ErrorRecallKindNames()
	if len(ern) == 0 {
		t.Fatal("error recall names")
	}
}

func TestNilKindCatalogSafe(t *testing.T) {
	var c *KindCatalog
	if c.All() != nil || c.Names() != nil {
		t.Fatal("nil all/names")
	}
	if len(c.FailureKinds()) == 0 {
		t.Fatal("fallback failure kinds")
	}
}
