package cluster

import (
	"context"
	"strings"
	"testing"
)

func TestInotifyAdviceMatchesTheDockerHost(t *testing.T) {
	for _, tc := range []struct{ host, wantNow, wantPersist string }{
		{"unix:///Users/me/.lima/docker/sock/docker.sock", "limactl shell docker sudo sysctl -w fs.inotify.max_user_instances=512", "limactl shell docker sh -c"},
		{"unix:///Users/me/.lima/box/sock/docker.sock", "limactl shell box sudo sysctl", "limactl shell box sh -c"},
		{"unix:///Users/me/.colima/default/docker.sock", "colima ssh -- sudo sysctl -w", "colima ssh -- sh -c"},
		{"unix:///var/run/docker.sock", "sudo sysctl -w fs.inotify.max_user_instances=512", "/etc/sysctl.d/99-k8slings.conf"},
		{"", "sudo sysctl -w", "/etc/sysctl.d"},
	} {
		got := InotifyAdvice(tc.host)
		if !strings.Contains(got, tc.wantNow) || !strings.Contains(got, tc.wantPersist) {
			t.Errorf("%q: advice lacks %q and %q:\n%s", tc.host, tc.wantNow, tc.wantPersist, got)
		}
	}
}

func TestInotifyLimitsAreReadFromAContainerOfTheNodeImage(t *testing.T) {
	c, f := newCluster(t, func(cmd Cmd) (string, error) { return "128\n63408\n", nil })
	instances, watches, err := c.InotifyLimits(context.Background())
	if err != nil || instances != 128 || watches != 63408 {
		t.Fatalf("got %d %d %v", instances, watches, err)
	}
	if call := f.called("docker run")[0]; !strings.Contains(strings.Join(call.Args, " "), NodeImage) {
		t.Fatalf("the limits must come from the Docker host's kernel via the node image: %v", call.Args)
	}
}

// On a machine without the node image, docker run pulls it first and its
// progress output comes before the two numbers. This was seen in CI: the check
// was skipped with "unexpected inotify limits" exactly when it mattered most.
func TestInotifyLimitsIgnoreDockerPullProgress(t *testing.T) {
	pull := "Unable to find image 'kindest/node@sha256:a1ed' locally\n" +
		"docker.io/kindest/node@sha256:a1ed: Pulling from kindest/node\n" +
		"0f4652d050f3: Pulling fs layer\n0f4652d050f3: Pull complete\n" +
		"Digest: sha256:a1ed\nStatus: Downloaded newer image for kindest/node@sha256:a1ed\n128\n63408"
	c, _ := newCluster(t, func(Cmd) (string, error) { return pull, nil })
	instances, watches, err := c.InotifyLimits(context.Background())
	if err != nil || instances != 128 || watches != 63408 {
		t.Fatalf("got %d %d %v", instances, watches, err)
	}

	for name, out := range map[string]string{"no numbers": "Error: nope", "one number": "512"} {
		c, _ := newCluster(t, func(Cmd) (string, error) { return out, nil })
		if _, _, err := c.InotifyLimits(context.Background()); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
