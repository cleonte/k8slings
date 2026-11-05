package exercises_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// TestSolutions is the proof that every exercise is solvable and really broken:
// the exercise as shipped must fail, and its solution must pass. Cluster and
// fix exercises are proven by the cluster job instead, so they are skipped.
func TestSolutions(t *testing.T) {
	env := &exercises.Env{Embedded: k8slings.FS, Validator: schema.New(k8slings.Schemas)}
	for _, ex := range load(t) {
		t.Run(ex.Name, func(t *testing.T) {
			if ex.Mode != exercises.ModeManifest {
				t.Skipf("%s mode is proven by the cluster tests", ex.Mode)
			}
			broken, err := fs.ReadFile(k8slings.FS, ex.Source())
			if err != nil {
				t.Fatal(err)
			}
			_, err = ex.RunSource(context.Background(), env, broken)
			var failed *exercises.FailedError
			if !errors.As(err, &failed) {
				t.Fatalf("the unsolved exercise must fail with a FailedError, got %v", err)
			}
			if len(failed.Lines) == 0 {
				t.Fatal("the failure has no message for the learner")
			}

			sol, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ex.RunSource(context.Background(), env, sol); err != nil {
				t.Fatalf("the solution must pass: %v", err)
			}
		})
	}
}
