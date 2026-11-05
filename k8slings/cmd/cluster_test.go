package cmd

import (
	"errors"
	"testing"

	"github.com/cleonte/k8slings/k8slings/cluster"
)

// A refused flag must be refused (and explained) before anything looks for a
// cluster, so this needs no Docker.
func TestKubectlRefusesFlagsThatChooseTheCluster(t *testing.T) {
	cmd := KubectlCmd()
	cmd.SetArgs([]string{"get", "pods", "--context", "prod"})
	err := cmd.Execute()
	var forbidden cluster.ErrForbiddenFlag
	if !errors.As(err, &forbidden) || forbidden.Flag != "--context" {
		t.Fatalf("want ErrForbiddenFlag for --context, got %v", err)
	}
}
