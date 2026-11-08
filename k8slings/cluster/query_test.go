package cluster

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func newQueryCluster(f *fakeRunner) *Cluster {
	return &Cluster{Name: "k8slings", Kubeconfig: "/owned/kubeconfig", Runner: f}
}

func TestQueryRunsKubectlWithTheOwnedKubeconfigAndSplitsTheStreams(t *testing.T) {
	f := &fakeRunner{handle: func(c Cmd) (string, error) { return "web-1\n", nil }}
	res, err := newQueryCluster(f).Query(t.Context(), []string{"get", "pods", "--output=name"}, false)
	if err != nil || res.Stdout != "web-1\n" || res.Stderr != "" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	calls := f.called("kubectl")
	if len(calls) != 1 || strings.Join(calls[0].Args[:4], " ") != "--kubeconfig /owned/kubeconfig --context kind-k8slings" {
		t.Fatalf("calls = %+v", calls)
	}
	for _, e := range calls[0].Env {
		if strings.HasPrefix(e, "KUBECONFIG=") {
			t.Errorf("KUBECONFIG must be removed from the environment, got %s", e)
		}
	}
}

func TestQueryStillRefusesForbiddenFlags(t *testing.T) {
	f := &fakeRunner{handle: func(Cmd) (string, error) { return "", nil }}
	_, err := newQueryCluster(f).Query(t.Context(), []string{"get", "pods", "--kubeconfig=/other"}, false)
	if !errors.As(err, new(ErrForbiddenFlag)) || len(f.calls) != 0 {
		t.Fatalf("err=%v calls=%d: CheckArgs is the last check for every path", err, len(f.calls))
	}
}

func TestQueryWithoutKubectl(t *testing.T) {
	f := &fakeRunner{missing: map[string]bool{"kubectl": true}, handle: func(Cmd) (string, error) { return "", nil }}
	if _, err := newQueryCluster(f).Query(t.Context(), []string{"get", "pods"}, false); !errors.Is(err, ErrNoKubectl) {
		t.Fatalf("err = %v", err)
	}
}

func TestOfflineQueryUsesAKubeconfigThatLeadsNowhere(t *testing.T) {
	var seen string
	f := &fakeRunner{handle: func(c Cmd) (string, error) {
		seen = c.Args[1]
		data, err := os.ReadFile(seen)
		if err != nil || !strings.Contains(string(data), "https://127.0.0.1:1") || !strings.Contains(string(data), "name: kind-k8slings-proof") || !strings.Contains(string(data), "token: offline") || !strings.Contains(c.Args[3], "kind-k8slings-proof") {
			t.Errorf("the offline kubeconfig is wrong: %q %v", data, err)
		}
		return "kind: Deployment\n", nil
	}}
	c := newQueryCluster(f)
	c.Name = "k8slings-proof" // the context is kind-<name>, not always kind-k8slings
	if _, err := c.Query(t.Context(), []string{"create", "deployment", "web", "--dry-run=client"}, true); err != nil {
		t.Fatal(err)
	}
	if seen == "/owned/kubeconfig" || seen == "" {
		t.Fatalf("the owned kubeconfig was used: %q", seen)
	}
	if _, err := os.Stat(seen); !os.IsNotExist(err) {
		t.Error("the temporary kubeconfig must be removed")
	}
}

func TestQueryKeepsReadingPastTheCapAndSaysSo(t *testing.T) {
	big := strings.Repeat("x", QueryMaxOutput+10)
	f := &fakeRunner{handle: func(Cmd) (string, error) { return big, nil }}
	res, err := newQueryCluster(f).Query(t.Context(), []string{"get", "pods"}, false)
	if err != nil || !res.Truncated || len(res.Stdout) != QueryMaxOutput {
		t.Fatalf("truncated=%v len=%d err=%v", res.Truncated, len(res.Stdout), err)
	}
}

func TestCapWriterNeverFails(t *testing.T) {
	w := &capWriter{max: 4}
	for _, s := range []string{"ab", "cdef", "gh"} {
		if n, err := w.Write([]byte(s)); n != len(s) || err != nil {
			t.Fatalf("Write(%q) = %d, %v", s, n, err)
		}
	}
	if w.buf.String() != "abcd" || !w.truncated {
		t.Fatalf("buf=%q truncated=%v", w.buf.String(), w.truncated)
	}
}

func TestQueryReportsATimeout(t *testing.T) {
	f := &fakeRunner{handle: func(Cmd) (string, error) { return "", errFake }}
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	res, err := newQueryCluster(f).Query(ctx, []string{"get", "pods"}, false)
	if err == nil || !res.TimedOut {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}
