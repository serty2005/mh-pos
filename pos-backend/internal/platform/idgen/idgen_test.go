package idgen

import (
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUUIDGeneratorNewIDReturnsUUIDv7(t *testing.T) {
	gen := UUIDGenerator{}
	id := gen.NewID()
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("expected valid uuid, got %q: %v", id, err)
	}
	if parsed.Version() != 7 {
		t.Fatalf("expected uuid version 7, got %d (%s)", parsed.Version(), id)
	}
}

func TestUUIDGeneratorNewIDSortsChronologically(t *testing.T) {
	gen := UUIDGenerator{}
	first := gen.NewID()
	time.Sleep(2 * time.Millisecond)
	second := gen.NewID()
	ids := []string{second, first}
	sort.Strings(ids)
	if ids[0] != first || ids[1] != second {
		t.Fatalf("expected lexicographic sort to follow time order: %v", ids)
	}
}
