//go:build cluster

package exercises_test

import (
	"fmt"
	"os"
	"testing"
)

// inShard splits the proofs of one test across CI jobs, each with its own
// cluster. K8SLINGS_SHARD="i/n" (i from 0) runs every n-th exercise starting at
// the i-th; unset runs all of them. The index is the exercise's place in
// info.toml among those of the same mode, so the split is stable.
func inShard(t *testing.T, index int) bool {
	t.Helper()
	spec := os.Getenv("K8SLINGS_SHARD")
	if spec == "" {
		return true
	}
	var i, n int
	if _, err := fmt.Sscanf(spec, "%d/%d", &i, &n); err != nil || n < 1 || i < 0 || i >= n {
		t.Fatalf("K8SLINGS_SHARD=%q: want i/n with 0 <= i < n", spec)
	}
	return index%n == i
}
