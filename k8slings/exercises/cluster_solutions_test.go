//go:build cluster

package exercises_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// TestClusterSolutions proves the cluster exercises on a real cluster: as
// shipped each one must fail (within a short timeout, and by the exercise's own
// check, not by a broken cluster), and with its solution each one must pass.
// It is run by `make cluster-test`, and needs Docker, kind and kubectl:
//
//	go test -tags cluster -run TestClusterSolutions -timeout 30m ./k8slings/exercises/
//
// K8SLINGS_KEEP=1 leaves the cluster (k8slings-proof) up afterwards, and a later
// run reuses it.
func TestClusterSolutions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	cl := &cluster.Cluster{Name: "k8slings-proof", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: cluster.ExecRunner{}}
	var up bytes.Buffer
	if err := cl.Up(ctx, cluster.UpOptions{Out: &up}); err != nil {
		t.Fatalf("cluster up: %v\n%s", err, up.String())
	}
	t.Log(up.String())
	if os.Getenv("K8SLINGS_KEEP") == "" {
		t.Cleanup(func() { _ = cl.Down(context.Background()) })
	}

	validator := schema.New(k8slings.Schemas)
	solve := &exercises.Env{Embedded: k8slings.FS, Validator: validator, Kube: cl}
	broken := &exercises.Env{Embedded: k8slings.FS, Validator: validator, Kube: cl, Timeout: 15 * time.Second}

	for _, ex := range load(t) {
		if ex.Mode != exercises.ModeCluster {
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			src, err := fs.ReadFile(k8slings.FS, ex.Source())
			if err != nil {
				t.Fatal(err)
			}
			_, err = ex.RunSource(ctx, broken, src)
			var failed *exercises.FailedError
			if !errors.As(err, &failed) {
				t.Fatalf("the unsolved exercise must fail with a FailedError, got %v", err)
			}
			t.Logf("fails as intended: %s", failed.Lines[0])

			sol, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if _, err := ex.RunSource(ctx, solve, sol); err != nil {
				// Show what the cluster looked like, not only which step failed.
				var f *exercises.FailedError
				if errors.As(err, &f) {
					t.Fatalf("the solution must pass: %v\n%s", err, strings.Join(f.Details, "\n"))
				}
				t.Fatalf("the solution must pass: %v", err)
			}
			t.Logf("solution passes in %s", time.Since(start).Round(time.Second))
		})
	}
}
