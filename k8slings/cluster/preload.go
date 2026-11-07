package cluster

import (
	"context"
	"strings"
)

// PreloadImages are the images the exercises use. They are pulled into the
// nodes while the cluster is created, so that no exercise waits for, or fails
// on, a registry pull. A test checks that every image the cluster exercises
// use is listed here.
var PreloadImages = []string{
	"nginx:1.27",
	"busybox:1.37",
	"hashicorp/http-echo:1.0.0", // troubleshooting19: an image with no shell
}

// Preload pulls the images straight into every node's container runtime. It
// does not use `kind load docker-image`: that copies an image from the Docker
// host, and fails ("content digest ... not found") when Docker uses its
// containerd image store, as current Docker Desktop and Lima setups do.
//
// A failure (an offline machine, a registry rate limit) is reported and
// skipped: the exercise that needs the image pulls it itself, only slower.
func (c *Cluster) Preload(ctx context.Context, o *UpOptions) {
	out, err := output(ctx, c.Runner, c.env(), "kind", "get", "nodes", "--name", c.Name)
	if err != nil {
		o.logf("Could not list the nodes (%s); skipping the image preload.", firstLine(out))
		return
	}
	nodes := strings.Fields(out)
	for _, img := range PreloadImages {
		for _, node := range nodes {
			if _, err := output(ctx, c.Runner, nil, "docker", "exec", node, "crictl", "inspecti", img); err == nil {
				continue // already there
			}
			o.logf("Pulling %s on %s...", img, node)
			if out, err := output(ctx, c.Runner, nil, "docker", "exec", node, "crictl", "pull", img); err != nil {
				o.logf("Could not pull %s on %s (%s); exercises that use it will pull it themselves.", img, node, strings.TrimSpace(lastLines(out, 1)))
				break // the other nodes would fail the same way
			}
		}
	}
}
