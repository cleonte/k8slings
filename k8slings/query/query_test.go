package query

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var ns = Options{Namespace: "k8slings-kubectl1"}

func TestParseAccepts(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		opt        Options
		args       []string
		verb       string
		offline    bool
	}{
		{"plain", "kubectl get pods -l app=web -o name", ns, []string{"get", "pods", "--selector=app=web", "--output=name", "--namespace=k8slings-kubectl1"}, "get", false},
		{"through k8slings", "k8slings kubectl get pods", ns, []string{"get", "pods", "--namespace=k8slings-kubectl1"}, "get", false},
		{"jsonpath in single quotes", `kubectl get pods -o jsonpath='{.items[*].metadata.name}'`, ns, []string{"get", "pods", "--output=jsonpath={.items[*].metadata.name}", "--namespace=k8slings-kubectl1"}, "get", false},
		{"quote joined to a word", `kubectl get pods -o 'jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}'`, ns, []string{"get", "pods", `--output=jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}`, "--namespace=k8slings-kubectl1"}, "get", false},
		{"double quotes", `kubectl get pods -l "tier in (a,b)"`, ns, []string{"get", "pods", "--selector=tier in (a,b)", "--namespace=k8slings-kubectl1"}, "get", false},
		{"escaped quote in double quotes", `kubectl get pods -o "jsonpath={.items[?(@.x==\"y\")].metadata.name}"`, ns, []string{"get", "pods", `--output=jsonpath={.items[?(@.x=="y")].metadata.name}`, "--namespace=k8slings-kubectl1"}, "get", false},
		{"attached short value", "kubectl get pods -oname -lapp=web", ns, []string{"get", "pods", "--output=name", "--selector=app=web", "--namespace=k8slings-kubectl1"}, "get", false},
		{"short with equals", "kubectl get pods -o=name", ns, []string{"get", "pods", "--output=name", "--namespace=k8slings-kubectl1"}, "get", false},
		{"long flag with a space", "kubectl get pods --output name --sort-by .metadata.name", ns, []string{"get", "pods", "--output=name", "--sort-by=.metadata.name", "--namespace=k8slings-kubectl1"}, "get", false},
		{"own namespace is fine", "kubectl get pods -n k8slings-kubectl1", ns, []string{"get", "pods", "--namespace=k8slings-kubectl1"}, "get", false},
		{"own namespace attached", "kubectl get pods -nk8slings-kubectl1", ns, []string{"get", "pods", "--namespace=k8slings-kubectl1"}, "get", false},
		{"all namespaces when the exercise opts in", "kubectl get pods -A", Options{Namespace: "k8slings-x", AllNamespaces: true}, []string{"get", "pods", "--all-namespaces=true", "--namespace=k8slings-x"}, "get", false},
		{"comment and blank lines", "# k8slings reads this file\n\n# TODO: x\nkubectl get pods # the pods\n", ns, []string{"get", "pods", "--namespace=k8slings-kubectl1"}, "get", false},
		{"continuation", "kubectl get pods \\\n  -o name \\\n  -l app=web\n", ns, []string{"get", "pods", "--output=name", "--selector=app=web", "--namespace=k8slings-kubectl1"}, "get", false},
		{"crlf", "kubectl get pods\r\n", ns, []string{"get", "pods", "--namespace=k8slings-kubectl1"}, "get", false},
		{"hash inside a word is not a comment", "kubectl get pods -l a=b#c", ns, []string{"get", "pods", "--selector=a=b#c", "--namespace=k8slings-kubectl1"}, "get", false},
		{"boolean with value", "kubectl logs web -p=true", ns, []string{"logs", "web", "--previous=true", "--namespace=k8slings-kubectl1"}, "logs", false},
		{"explain", "kubectl explain deployment.spec.strategy", ns, []string{"explain", "deployment.spec.strategy", "--namespace=k8slings-kubectl1"}, "explain", false},
		{"can-i with --as", "kubectl auth can-i --list --as system:serviceaccount:k8slings-kubectl1:ci", ns, []string{"auth", "can-i", "--list=true", "--as=system:serviceaccount:k8slings-kubectl1:ci", "--namespace=k8slings-kubectl1"}, "auth can-i", false},
		{"create offline", "kubectl create deployment web --image=nginx:1.27 --replicas=3 --dry-run=client -o yaml", Options{}, []string{"create", "deployment", "web", "--image=nginx:1.27", "--replicas=3", "--dry-run=client", "--output=yaml"}, "create deployment", true},
		{"create service", "kubectl create service clusterip web --tcp=80:8080 --dry-run=client -o yaml", Options{}, []string{"create", "service", "clusterip", "web", "--tcp=80:8080", "--dry-run=client", "--output=yaml"}, "create service", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.text, tc.opt)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Args, tc.args) {
				t.Errorf("args\n got %q\nwant %q", got.Args, tc.args)
			}
			if got.Verb != tc.verb || got.Offline != tc.offline {
				t.Errorf("verb=%q offline=%v, want %q %v", got.Verb, got.Offline, tc.verb, tc.offline)
			}
		})
	}
}

func TestParseRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		opt        Options
		want       string // a fragment of the message
	}{
		{"pipe", "kubectl get pods | grep web", ns, "not a shell"},
		{"redirection", "kubectl get pods > out.txt", ns, "not a shell"},
		{"input redirection", "kubectl get pods < in", ns, "not a shell"},
		{"semicolon", "kubectl get pods; kubectl get nodes", ns, "not a shell"},
		{"and", "kubectl get pods && echo hi", ns, "not a shell"},
		{"background", "kubectl get pods &", ns, "not a shell"},
		{"subshell", "kubectl get pods $(echo x)", ns, "command substitution"},
		{"variable", "kubectl get pods $NS", ns, "Variables"},
		{"backtick", "kubectl get pods `echo x`", ns, "Variables"},
		{"dollar in double quotes", `kubectl get pods -o "jsonpath=$x"`, ns, "double quotes"},
		{"backtick in double quotes", "kubectl get pods -o \"jsonpath=`x`\"", ns, "double quotes"},
		{"escaped dollar in double quotes", `kubectl get pods -o "jsonpath=\$x"`, ns, "double quotes"},
		{"glob", "kubectl get pods web-*", ns, "Quote it"},
		{"brace", "kubectl get pods -o jsonpath={.items}", ns, "Quote it"},
		{"bracket", "kubectl get pods --sort-by=.spec.ports[0].port", ns, "Quote it"},
		{"bang", "kubectl get pods -l !canary", ns, "Quote it"},
		{"tilde", "kubectl get pods ~", ns, "Quote it"},
		{"backslash escape", `kubectl get pods a\ b`, ns, "backslash"},
		{"unclosed single quote", "kubectl get pods -l 'a=b", ns, "never closed"},
		{"unclosed double quote", `kubectl get pods -l "a=b`, ns, "never closed"},
		{"two commands", "kubectl get pods\nkubectl get nodes", ns, "One command"},
		{"empty", "# only a comment\n", ns, "empty"},
		{"alias", "k get pods", ns, "Aliases"},
		{"not kubectl", "ls", ns, "start with kubectl"},
		{"bare kubectl", "kubectl", ns, "Write what"},
		{"delete", "kubectl delete pod web", ns, "changes the cluster"},
		{"apply", "kubectl apply -f x.yaml", ns, "changes the cluster"},
		{"exec", "kubectl exec web -- ls", ns, "runs something"},
		{"unknown verb", "kubectl frobnicate", ns, "not available"},
		{"plugin", "kubectl whoami", ns, "not available"},
		{"auth other", "kubectl auth whoami", ns, "only can-i"},
		{"create without dry-run", "kubectl create deployment web --image=nginx:1.27", ns, "--dry-run=client"},
		{"create dry-run server", "kubectl create deployment web --image=nginx:1.27 --dry-run=server", ns, "--dry-run=client"},
		{"create dry-run with a space", "kubectl create deployment web --image=nginx:1.27 --dry-run client", ns, "--dry-run=client"},
		{"create dry-run alone", "kubectl create deployment web --image=nginx:1.27 --dry-run", ns, "--dry-run=client"},
		{"create token", "kubectl create token web --dry-run=client", ns, "not available"},
		{"create nothing", "kubectl create --dry-run=client", ns, "Say what"},
		{"create service without kind", "kubectl create service web --dry-run=client", ns, "which kind"},
		{"create secret tls", "kubectl create secret tls x --dry-run=client", ns, "which kind"},
		{"create from-file", "kubectl create configmap x --from-file=/etc/passwd --dry-run=client", ns, "--from-file"},
		{"create -f", "kubectl create deployment x -f y.yaml --dry-run=client", ns, "-f"},
		{"run", "kubectl run web --image=nginx --dry-run=client", ns, "runs something"},
		{"expose", "kubectl expose deployment web --dry-run=client", ns, "changes the cluster"},
		{"other namespace", "kubectl get pods -n kube-system", ns, "another namespace"},
		{"other namespace long", "kubectl get pods --namespace=kube-system", ns, "another namespace"},
		{"other namespace attached", "kubectl get pods -nkube-system", ns, "another namespace"},
		{"all namespaces", "kubectl get pods -A", ns, "own namespace only"},
		{"all namespaces long", "kubectl get pods --all-namespaces", ns, "own namespace only"},
		{"namespace offline", "kubectl create deployment x --image=y --dry-run=client -n x", Options{}, "no namespace"},
		{"all namespaces offline", "kubectl get pods -A", Options{AllNamespaces: true}, "own namespace only"},
		{"kubeconfig", "kubectl get pods --kubeconfig=/tmp/x", ns, "not available"},
		{"context", "kubectl get pods --context other", ns, "not available"},
		{"server", "kubectl get pods -s https://x", ns, "not available"},
		{"token", "kubectl get pods --token=x", ns, "not available"},
		{"raw", "kubectl get --raw /api", ns, "not available"},
		{"filename", "kubectl get -f x.yaml", ns, "not available"},
		{"kustomize", "kubectl get -k dir", ns, "not available"},
		{"watch", "kubectl get pods -w", ns, "not available"},
		{"follow", "kubectl logs web -f", ns, "not available"},
		{"profile", "kubectl get pods --profile=cpu", ns, "not available"},
		{"cache dir", "kubectl get pods --cache-dir=/tmp/x", ns, "not available"},
		{"as with get", "kubectl get pods --as admin", ns, "not available"},
		{"double dash", "kubectl get pods -- --kubeconfig=x", ns, "--"},
		{"combined shorthands", "kubectl get pods -Ao name", Options{Namespace: "k8slings-x", AllNamespaces: true}, "one by one"},
		{"missing value", "kubectl get pods -o", ns, "needs a value"},
		{"bool with a value", "kubectl get pods --no-headers=maybe", ns, "takes no value"},
		{"output file", "kubectl get pods -o jsonpath-file=/etc/passwd", ns, "-file"},
		{"output custom file", "kubectl get pods -o custom-columns-file=x", ns, "-file"},
		{"output template file", "kubectl get pods -o go-template-file=x", ns, "-file"},
		{"can-i without a question", "kubectl auth can-i", ns, "Say what"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.text, tc.opt)
			if err == nil {
				t.Fatalf("accepted: %q", got.Args)
			}
			var r *RefusedError
			if !errors.As(err, &r) {
				t.Fatalf("want a RefusedError, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("message %q does not contain %q", err, tc.want)
			}
		})
	}
}

// What kubectl gets is built from what the parser understood: nothing of the
// learner's text reaches it except the values of the allowed flags and the
// positional words.
func TestNoArgumentLooksLikeAFlagNobodyChecked(t *testing.T) {
	for _, text := range []string{
		"kubectl get pods -l app=web -o name",
		"kubectl get pods --sort-by=.metadata.name --no-headers",
		"kubectl logs web -c app --previous --tail 5",
		"kubectl explain pod.spec",
	} {
		got, err := Parse(text, ns)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range got.Args {
			if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
				t.Errorf("%q: a short flag %q reached kubectl", text, a)
			}
		}
	}
}

func TestFlagsAreCanonicalForTheCheck(t *testing.T) {
	a, _ := Parse("kubectl get pods -o jsonpath='{.items}'", ns)
	b, _ := Parse("kubectl get pods --output=jsonpath='{.items}'", ns)
	if a.Flags["output"] != "jsonpath={.items}" || !reflect.DeepEqual(a.Flags, b.Flags) {
		t.Errorf("flags: %v and %v", a.Flags, b.Flags)
	}
	if _, ok := a.Flags["namespace"]; ok {
		t.Error("the namespace k8slings adds is not one of the learner's flags")
	}
}
