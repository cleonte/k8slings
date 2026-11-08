// Package query reads the answer of a query exercise: one kubectl command that
// the learner types into a file. The text never reaches a shell. It is split into
// words by a small tokenizer that accepts only the part of POSIX shell syntax
// whose meaning is the same with or without a shell (words, single quotes,
// double quotes), and then checked against an allowlist of read-only verbs and
// flags. What comes out is an argument list for kubectl, built from the flags the
// parser understood, never from the learner's text.
//
// The package does not import os/exec: running the command is the cluster
// package's job.
package query

import (
	"fmt"
	"slices"
	"strings"
)

// RefusedError says why a command is not allowed in a query exercise. Its
// message is written for the learner.
type RefusedError struct{ Msg string }

func (e *RefusedError) Error() string { return e.Msg }

func refuse(format string, args ...any) error {
	return &RefusedError{Msg: fmt.Sprintf(format, args...)}
}

// Options is what an exercise says about the command.
type Options struct {
	// Namespace is the namespace k8slings adds to the command (the exercise's
	// own). Empty for an offline exercise, which has no namespace: -n and -A are
	// then refused.
	Namespace string
	// AllNamespaces lets the command use -A.
	AllNamespaces bool
}

// Command is a command that passed every rule.
type Command struct {
	// Verb is "get", "describe", "explain", "logs", "events", "api-resources",
	// "api-versions", "auth can-i" or "create <what>" (create deployment).
	Verb string
	// Positional are the words that are not flags, after the verb.
	Positional []string
	// Flags maps the long name of each flag the learner gave to its value (the
	// last one wins; "true" for a flag without value), without the namespace
	// that k8slings adds.
	Flags map[string]string
	// Args is what kubectl gets: the verb, the positional words, the flags in
	// their long form, and the namespace.
	Args []string
	// Offline says that the command needs no cluster (create ... --dry-run=client).
	Offline bool
}

// Parse reads and checks the text of an answer.
func Parse(text string, opt Options) (*Command, error) {
	cmds, err := tokenize(text)
	if err != nil {
		return nil, err
	}
	switch len(cmds) {
	case 0:
		return nil, refuse("The answer is empty: write one kubectl command.")
	case 1:
	default:
		return nil, refuse("One command per exercise: found %d. Put the second one in its own exercise, or use one command that prints both.", len(cmds))
	}
	words := cmds[0]
	switch {
	case words[0] == "kubectl":
		words = words[1:]
	case words[0] == "k8slings" && len(words) > 1 && words[1] == "kubectl":
		words = words[2:]
	case words[0] == "k":
		return nil, refuse("Aliases are not read: write kubectl (or k8slings kubectl), not k.")
	default:
		return nil, refuse("The command has to start with kubectl (or k8slings kubectl), not %q.", words[0])
	}
	if len(words) == 0 {
		return nil, refuse("Write what kubectl should do: kubectl get ..., kubectl explain ...")
	}
	return check(words, opt)
}

// tokenize splits the text into commands (one per line, a backslash at the end
// of a line continues it) and each into words.
func tokenize(text string) ([][]string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var (
		cmds   [][]string
		words  []string
		cur    strings.Builder
		inWord bool
	)
	endWord := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	endCmd := func() {
		endWord()
		if len(words) > 0 {
			cmds = append(cmds, words)
			words = nil
		}
	}
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\n':
			endCmd()
		case r == ' ' || r == '\t':
			endWord()
		case r == '#' && !inWord:
			for i+1 < len(rs) && rs[i+1] != '\n' {
				i++
			}
		case r == '\\':
			if i+1 < len(rs) && rs[i+1] == '\n' {
				i++ // a line continuation: the command goes on
				continue
			}
			return nil, refuse("A backslash is only read at the end of a line. Put the text in quotes instead.")
		case r == '\'':
			inWord = true
			i++
			for i < len(rs) && rs[i] != '\'' {
				cur.WriteRune(rs[i])
				i++
			}
			if i >= len(rs) {
				return nil, refuse("A single quote is never closed.")
			}
		case r == '"':
			inWord = true
			i++
			for i < len(rs) && rs[i] != '"' {
				switch c := rs[i]; {
				case c == '$' || c == '`':
					return nil, refuse("Inside double quotes a shell expands $ and backticks, and k8slings does not: use single quotes ('...').")
				case c == '\\' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\'):
					i++
					cur.WriteRune(rs[i])
				case c == '\\' && i+1 < len(rs) && rs[i+1] == '\n':
					i++ // continuation inside the quotes
				case c == '\\' && i+1 < len(rs) && (rs[i+1] == '$' || rs[i+1] == '`'):
					return nil, refuse("Inside double quotes a shell expands $ and backticks, and k8slings does not: use single quotes ('...').")
				default:
					cur.WriteRune(c)
				}
				i++
			}
			if i >= len(rs) {
				return nil, refuse("A double quote is never closed.")
			}
		case strings.ContainsRune("|&;<>()", r):
			return nil, refuse("k8slings runs one kubectl command, not a shell: pipes, redirections, && and ; are not available. Let kubectl do the work (-o, --sort-by, --field-selector).")
		case r == '$' || r == '`':
			return nil, refuse("Variables and command substitution are not available: write the value itself.")
		case strings.ContainsRune("*?[]{}~!", r):
			return nil, refuse("Quote it: an unquoted %q is special to a shell (write it inside single quotes, '...').", string(r))
		default:
			inWord = true
			cur.WriteRune(r)
		}
	}
	endCmd()
	return cmds, nil
}

type flagDef struct {
	long  string
	short string
	value bool // takes a value
}

var (
	fNamespace = flagDef{"namespace", "n", true}
	fAllNS     = flagDef{"all-namespaces", "A", false}
	fOutput    = flagDef{"output", "o", true}
	fSelector  = flagDef{"selector", "l", true}
	fNoHeaders = flagDef{"no-headers", "", false}
)

func vflag(long string) flagDef { return flagDef{long: long, value: true} }
func bflag(long string) flagDef { return flagDef{long: long} }

// verbs are the read-only commands, each with the flags it may use. A flag that
// is not listed is refused: that is how --raw, -f, --follow, -k, --from-file,
// --profile and every flag nobody has thought about stay out.
var verbs = map[string][]flagDef{
	"get": {fNamespace, fAllNS, fOutput, fSelector, fNoHeaders,
		vflag("field-selector"), vflag("sort-by"), bflag("show-labels"), {"label-columns", "L", true},
		bflag("ignore-not-found"), bflag("show-kind"), vflag("template")},
	"describe":      {fNamespace, fAllNS, fSelector, bflag("show-events")},
	"explain":       {fNamespace, fOutput, bflag("recursive"), vflag("api-version")},
	"logs":          {fNamespace, fSelector, {"container", "c", true}, {"previous", "p", false}, vflag("tail"), vflag("since"), bflag("timestamps"), vflag("limit-bytes"), bflag("prefix"), bflag("all-containers")},
	"events":        {fNamespace, fAllNS, fOutput, vflag("for"), vflag("types"), fNoHeaders},
	"api-resources": {fNamespace, fOutput, fNoHeaders, vflag("namespaced"), vflag("api-group"), vflag("verbs"), vflag("sort-by"), vflag("categories"), bflag("cached")},
	"api-versions":  {fNamespace},
	"auth can-i":    {fNamespace, fAllNS, vflag("as"), bflag("list"), {"quiet", "q", false}, fNoHeaders, vflag("subresource")},
}

// createWhat lists what `kubectl create` may generate. All of it is printed
// with --dry-run=client and never reaches the server.
var createWhat = map[string][]string{
	"deployment": nil, "configmap": nil, "namespace": nil, "serviceaccount": nil,
	"job": nil, "cronjob": nil, "role": nil, "clusterrole": nil, "rolebinding": nil,
	"clusterrolebinding": nil, "quota": nil, "poddisruptionbudget": nil, "priorityclass": nil,
	"ingress": nil,
	"service": {"clusterip", "nodeport", "loadbalancer", "externalname"},
	"secret":  {"generic"},
}

var createFlags = []flagDef{fNamespace, fOutput, vflag("dry-run"), vflag("image"), vflag("replicas"),
	vflag("port"), vflag("tcp"), vflag("node-port"), vflag("from-literal"), vflag("verb"), vflag("resource"),
	vflag("resource-name"), vflag("serviceaccount"), vflag("user"), vflag("group"), vflag("clusterrole"),
	vflag("role"), vflag("schedule"), vflag("hard"), vflag("min-available"), vflag("max-unavailable"),
	vflag("selector"), vflag("value"), vflag("rule"), vflag("external-name"), vflag("type"), bflag("command"),
	vflag("restart")}

// outputFormats are the values of -o. The *-file forms read a file of the
// learner's machine and are not among them.
var outputFormats = []string{"name", "wide", "json", "yaml", "jsonpath", "jsonpath-as-json", "custom-columns", "go-template", "template", "plaintext", "plaintext-openapiv2"}

func check(words []string, opt Options) (*Command, error) {
	verb, rest := words[0], words[1:]
	switch verb {
	case "auth":
		if len(rest) == 0 || rest[0] != "can-i" {
			return nil, refuse("Of kubectl auth only can-i is available here.")
		}
		verb, rest = "auth can-i", rest[1:]
	case "create":
	default:
		if _, ok := verbs[verb]; !ok {
			return nil, refuse("kubectl %s is not available in a query exercise. Allowed: get, describe, explain, logs, events, api-resources, api-versions, auth can-i, and create ... --dry-run=client. %s", verb, changesHint(verb))
		}
	}

	defs := verbs[verb]
	if verb == "create" {
		defs = createFlags
	}
	cmd := &Command{Verb: verb, Flags: map[string]string{}}
	var flagArgs []string
	for i := 0; i < len(rest); i++ {
		w := rest[i]
		if !strings.HasPrefix(w, "-") || w == "-" {
			cmd.Positional = append(cmd.Positional, w)
			continue
		}
		if w == "--" {
			return nil, refuse("-- is not available: nothing here runs a command in a container.")
		}
		var def flagDef
		var val string
		var hasVal bool
		if strings.HasPrefix(w, "--") {
			name, v, eq := strings.Cut(w[2:], "=")
			j := slices.IndexFunc(defs, func(d flagDef) bool { return d.long == name })
			if j < 0 {
				return nil, refuse("The flag --%s is not available for kubectl %s here.", name, verb)
			}
			def, val, hasVal = defs[j], v, eq
		} else {
			short := w[1:2]
			j := slices.IndexFunc(defs, func(d flagDef) bool { return d.short == short })
			if j < 0 {
				return nil, refuse("The flag -%s is not available for kubectl %s here.", short, verb)
			}
			def = defs[j]
			rem := w[2:]
			switch {
			case def.value:
				val = strings.TrimPrefix(rem, "=")
				hasVal = rem != ""
			case strings.HasPrefix(rem, "="): // -p=true
				val, hasVal = rem[1:], true
			case rem != "":
				return nil, refuse("Write the flags one by one, not combined (%s).", w)
			}
		}
		switch {
		case def.long == "dry-run":
			if !hasVal || val != "client" {
				return nil, refuse("--dry-run has to be written --dry-run=client: nothing is created, the YAML is only printed.")
			}
		case def.value && !hasVal:
			i++
			if i >= len(rest) {
				return nil, refuse("The flag --%s needs a value.", def.long)
			}
			val = rest[i]
		case !def.value && hasVal && val != "true" && val != "false":
			return nil, refuse("The flag --%s takes no value (or =true / =false).", def.long)
		case !def.value && !hasVal:
			val = "true"
		}
		switch def.long {
		case "output":
			kind, _, _ := strings.Cut(val, "=")
			if !slices.Contains(outputFormats, kind) {
				return nil, refuse("The output format %q is not available (a -file format would read your files).", val)
			}
		case "namespace":
			if opt.Namespace == "" {
				return nil, refuse("This exercise has no namespace: leave -n out.")
			}
			if val != opt.Namespace {
				return nil, refuse("This exercise runs in the namespace %s: another namespace is not available.", opt.Namespace)
			}
			continue // k8slings adds it itself
		case "all-namespaces":
			if opt.Namespace == "" || !opt.AllNamespaces {
				return nil, refuse("-A is not available in this exercise: it works in its own namespace only.")
			}
		}
		if def.long != "all-namespaces" || val == "true" {
			cmd.Flags[def.long] = val
		}
		flagArgs = append(flagArgs, "--"+def.long+"="+val)
	}

	if verb == "create" {
		if len(cmd.Positional) == 0 {
			return nil, refuse("Say what to create: kubectl create deployment ...")
		}
		what := cmd.Positional[0]
		subs, ok := createWhat[what]
		if !ok {
			return nil, refuse("kubectl create %s is not available here: it prints YAML for deployments, services, configmaps and similar.", what)
		}
		if subs != nil {
			if len(cmd.Positional) < 2 || !slices.Contains(subs, cmd.Positional[1]) {
				return nil, refuse("Say which kind of %s: %s.", what, strings.Join(subs, ", "))
			}
		}
		if cmd.Flags["dry-run"] != "client" {
			return nil, refuse("kubectl create changes the cluster. Only --dry-run=client is available: it prints the YAML and creates nothing.")
		}
		cmd.Verb = "create " + what
		cmd.Offline = true
	}
	if verb == "auth can-i" && len(cmd.Positional) == 0 && cmd.Flags["list"] != "true" {
		return nil, refuse("Say what to ask: kubectl auth can-i get pods, or --list.")
	}
	if _, ok := cmd.Flags["as"]; ok && verb != "auth can-i" {
		return nil, refuse("--as is only available with kubectl auth can-i.")
	}

	if verb == "create" {
		cmd.Args = append([]string{"create"}, cmd.Positional...)
	} else if verb == "auth can-i" {
		cmd.Args = append([]string{"auth", "can-i"}, cmd.Positional...)
	} else {
		cmd.Args = append([]string{verb}, cmd.Positional...)
	}
	cmd.Args = append(cmd.Args, flagArgs...)
	if opt.Namespace != "" {
		cmd.Args = append(cmd.Args, "--namespace="+opt.Namespace)
	}
	return cmd, nil
}

func changesHint(verb string) string {
	switch verb {
	case "apply", "create", "delete", "patch", "edit", "replace", "label", "annotate", "scale", "set", "rollout", "expose", "run", "taint", "cordon", "drain", "uncordon", "autoscale", "cp", "exec", "attach", "port-forward", "debug", "proxy":
		return "'" + verb + "' changes the cluster or runs something in it; query exercises only read."
	}
	return ""
}
