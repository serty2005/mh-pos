package idgen

import (
	"errors"
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

func TestUUIDGeneratorNewIDPanicsWhenUUIDv7GenerationFails(t *testing.T) {
	uuid.DisableRandPool()
	uuid.SetRand(&failingThenZeroReader{})
	t.Cleanup(func() {
		uuid.SetRand(nil)
		uuid.DisableRandPool()
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when uuidv7 generation fails")
		}
	}()

	_ = UUIDGenerator{}.NewID()
}

type failingThenZeroReader struct {
	failed bool
}

func (r *failingThenZeroReader) Read(p []byte) (int, error) {
	if !r.failed {
		r.failed = true
		return 0, errors.New("entropy unavailable")
	}
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
