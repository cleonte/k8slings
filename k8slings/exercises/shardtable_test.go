package exercises_test

import (
	_ "embed"
	"strconv"
	"strings"
	"testing"

	"github.com/cleonte/k8slings/k8slings/exercises"
)

// shardCount is the number of CI shards that shards.txt was made for.
const shardCount = 5

//go:embed shards.txt
var shardTable string

// pinnedShards reads shards.txt: the exercises that are placed on a shard by
// hand (by scripts/shards, from the timings of a CI run) and not by the hash of
// their name.
func pinnedShards(t *testing.T) map[string]int {
	t.Helper()
	pinned := map[string]int{}
	for i, l := range strings.Split(shardTable, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Fields(l)
		if len(f) != 2 {
			t.Fatalf("shards.txt line %d: want \"name shard\", got %q", i+1, l)
		}
		s, err := strconv.Atoi(f[1])
		if err != nil || s < 0 || s >= shardCount {
			t.Fatalf("shards.txt line %d: %q is not a shard from 0 to %d", i+1, f[1], shardCount-1)
		}
		if _, dup := pinned[f[0]]; dup {
			t.Fatalf("shards.txt: %s is listed twice", f[0])
		}
		pinned[f[0]] = s
	}
	return pinned
}

// A pinned name that is not an exercise (renamed, removed) would silently fall
// back to the hash, so the table must not keep one.
func TestShardTableNamesAreExercises(t *testing.T) {
	all, err := exercises.List("../../info.toml")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, e := range all {
		known[e.Name] = true
	}
	for name := range pinnedShards(t) {
		if !known[name] {
			t.Errorf("shards.txt pins %q, which is not an exercise; run scripts/shards again", name)
		}
	}
}
