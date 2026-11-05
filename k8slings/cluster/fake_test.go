package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// fakeRunner answers programs from a handler and records every call.
type fakeRunner struct {
	mu      sync.Mutex
	calls   []Cmd
	missing map[string]bool // programs that are not "installed"
	handle  func(c Cmd) (out string, err error)
}

func (f *fakeRunner) Has(name string) bool { return !f.missing[name] }

func (f *fakeRunner) Run(_ context.Context, c Cmd) error {
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()
	out, err := f.handle(c)
	if c.Stdout != nil {
		fmt.Fprint(c.Stdout, out)
	}
	return err
}

// called returns the calls whose command line starts with prefix.
func (f *fakeRunner) called(prefix string) []Cmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Cmd
	for _, c := range f.calls {
		if strings.HasPrefix(c.Name+" "+strings.Join(c.Args, " "), prefix) {
			out = append(out, c)
		}
	}
	return out
}

var errFake = errors.New("exit status 1")
