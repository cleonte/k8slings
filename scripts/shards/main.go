// Command shards writes k8slings/exercises/shards.txt: the pinned shard of the
// slowest exercises of the cluster proofs, so that the five CI jobs take about
// as long as each other.
//
//	gh run view <id> --log > run.log
//	go run ./scripts/shards -log run.log -out k8slings/exercises/shards.txt
//
// The input is the log of a cluster workflow run (go test -v prints
// "--- PASS: TestFixSolutions/troubleshooting4 (131.83s)" and the job name says
// "shard 2 of 5"). The slowest -pin exercises are placed one by one on the
// shard that has the least work, counting the work of the exercises that stay
// hashed (crc32 of the name mod -n, see shard_test.go) as it is in the log. Every
// other exercise keeps its hash, so adding an exercise moves no other.
package main

import (
	"flag"
	"fmt"
	"hash/crc32"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var line = regexp.MustCompile(`--- (?:PASS|FAIL): Test(?:Cluster|Fix|Query)Solutions/([A-Za-z0-9]+) \(([0-9.]+)s\)`)

func main() {
	logFile := flag.String("log", "", "log of a cluster workflow run (gh run view ID --log)")
	out := flag.String("out", "k8slings/exercises/shards.txt", "file to write")
	n := flag.Int("n", 5, "number of shards")
	pin := flag.Int("pin", 20, "how many of the slowest exercises to pin")
	flag.Parse()
	if *logFile == "" {
		fmt.Fprintln(os.Stderr, "usage: shards -log run.log [-out file] [-n 5] [-pin 20]")
		os.Exit(2)
	}
	data, err := os.ReadFile(*logFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	secs := map[string]float64{}
	for _, l := range strings.Split(string(data), "\n") {
		m := line.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		d, _ := strconv.ParseFloat(m[2], 64)
		secs[m[1]] += d // an exercise that is in two tests (cluster and query never are) adds up
	}
	if len(secs) == 0 {
		fmt.Fprintln(os.Stderr, "no timings found in the log")
		os.Exit(1)
	}
	names := make([]string, 0, len(secs))
	for k := range secs {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		if secs[names[i]] != secs[names[j]] {
			return secs[names[i]] > secs[names[j]]
		}
		return names[i] < names[j]
	})
	if *pin > len(names) {
		*pin = len(names)
	}
	pinned := names[:*pin]
	load := make([]float64, *n)
	for _, name := range names[*pin:] {
		load[int(crc32.ChecksumIEEE([]byte(name))%uint32(*n))] += secs[name]
	}
	assign := map[string]int{}
	for _, name := range pinned {
		best := 0
		for s := range load {
			if load[s] < load[best] {
				best = s
			}
		}
		assign[name] = best
		load[best] += secs[name]
	}
	sort.Strings(pinned)
	var b strings.Builder
	fmt.Fprintf(&b, "# The shard of the slowest exercises, for %d shards. Written by scripts/shards (see its header);\n# every exercise that is not here goes to crc32(name) mod %d. Format: name shard.\n", *n, *n)
	for _, name := range pinned {
		fmt.Fprintf(&b, "%s %d\n", name, assign[name])
	}
	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("pinned %d exercises; expected work per shard (s):", len(pinned))
	for _, l := range load {
		fmt.Printf(" %.0f", l)
	}
	fmt.Println()
}
