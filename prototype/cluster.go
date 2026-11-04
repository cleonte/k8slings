package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	clusterName = "k8slings"
	kubeContext = "kind-" + clusterName
)

// kubectl runs kubectl against the k8slings kind cluster only, regardless of
// the user's current context, so we can never touch another cluster.
func kubectl(args ...string) (string, error) {
	full := append([]string{"--context", kubeContext}, args...)
	cmd := exec.Command("kubectl", full...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

func requireTools(tools ...string) error {
	var missing []string
	for _, t := range tools {
		if _, err := exec.LookPath(t); err != nil {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s (install with: brew install %s)",
			strings.Join(missing, ", "), strings.Join(missing, " "))
	}
	return nil
}

func clusterExists() bool {
	out, err := exec.Command("kind", "get", "clusters").Output()
	if err != nil {
		return false
	}
	for _, l := range strings.Fields(string(out)) {
		if l == clusterName {
			return true
		}
	}
	return false
}

func clusterReady() error {
	if out, err := kubectl("get", "nodes", "--no-headers"); err != nil {
		return fmt.Errorf("cluster %q is not reachable (run `k8slings init`, and check Docker is running): %s", clusterName, out)
	}
	return nil
}

func createCluster() error {
	if err := requireTools("docker", "kind", "kubectl"); err != nil {
		return err
	}
	if clusterExists() {
		fmt.Println(dim("kind cluster " + clusterName + " already exists, reusing it"))
		return nil
	}
	fmt.Println(cyan("Creating kind cluster '" + clusterName + "' (first run pulls the node image, ~1-3 min)..."))
	cmd := exec.Command("kind", "create", "cluster", "--name", clusterName, "--wait", "120s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deleteCluster() error {
	if err := requireTools("kind"); err != nil {
		return err
	}
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
