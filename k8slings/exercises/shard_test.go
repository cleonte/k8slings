//go:build cluster

package exercises_test

import (
	"fmt"
	"hash/crc32"
	"os"
	"testing"
)

// inShard splits the proofs of one test across CI jobs, each with its own
// cluster. K8SLINGS_SHARD="i/n" (i from 0) runs the exercises whose name hashes
// to i; unset runs all of them. The split is by name, so it does not move when
// exercises are added. (It was every n-th exercise by position, and the four
// slowest fix exercises were all at multiples of five, so one shard took 14 minutes
// against 8 to 10 for the others.)
func inShard(t *testing.T, name string) bool {
	t.Helper()
	spec := os.Getenv("K8SLINGS_SHARD")
	if spec == "" {
		return true
	}
	var i, n int
	if _, err := fmt.Sscanf(spec, "%d/%d", &i, &n); err != nil || n < 1 || i < 0 || i >= n {
		t.Fatalf("K8SLINGS_SHARD=%q: want i/n with 0 <= i < n", spec)
	}
	return int(crc32.ChecksumIEEE([]byte(name))%uint32(n)) == i
}
