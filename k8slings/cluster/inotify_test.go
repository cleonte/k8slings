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
