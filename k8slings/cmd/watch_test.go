package cmd

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWatchLoopRunsOnStartChangeAndCommand(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	record := func(s string) func() {
		return func() { mu.Lock(); calls = append(calls, s); mu.Unlock() }
	}
	changes := make(chan []string)
	r, w := io.Pipe()
	done := make(chan struct{})
	go func() {
		watchLoop(r, changes, watchActions{run: record("run"), list: record("list"), hint: record("hint")})
		close(done)
	}()

	changes <- []string{"a.yaml"}
	go func() { _, _ = w.Write([]byte("h\nl\nr\nq\n")) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watchLoop did not return after q")
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(calls, ","); got != "run,run,hint,list,run" {
		t.Fatalf("calls = %s", got)
	}
}
