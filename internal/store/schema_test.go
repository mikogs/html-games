package store

import (
	"context"
	"path/filepath"
	"sort"
	"testing"
)

func TestMigrationsMatchStructs(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	applied, err := Migrate(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 {
		t.Fatal("no migrations applied to an empty database")
	}
	// a second run has nothing to do
	if applied, err := Migrate(context.Background(), db); err != nil || len(applied) != 0 {
		t.Fatalf("second Migrate = %v, %v", applied, err)
	}
}

func ptr(v int64) *int64 { return &v }

func TestSortKeyOrder(t *testing.T) {
	// in the order of the hall of fame: higher score, then faster time, then
	// no time; the IDs (the order of a full tie) are the slice index
	entries := []struct {
		score int64
		time  *int64
	}{
		{MaxScore, ptr(0)},
		{MaxScore, nil},
		{500, ptr(10)},
		{500, ptr(11)},
		{500, ptr(MaxTime)},
		{500, nil},
		{499, ptr(0)},
		{1, nil},
		{0, ptr(0)},
		{0, nil},
	}
	keys := make([]int64, len(entries))
	for i, e := range entries {
		keys[i] = SortKey(e.score, e.time)
	}
	if !sort.SliceIsSorted(keys, func(i, j int) bool { return keys[i] < keys[j] }) {
		t.Fatalf("keys are not ascending: %v", keys)
	}
	for i := 1; i < len(keys); i++ {
		if keys[i] == keys[i-1] {
			t.Fatalf("entries %d and %d have the same key %d", i-1, i, keys[i])
		}
	}
}
