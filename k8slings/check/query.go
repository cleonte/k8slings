package check

import (
	"fmt"
	"strings"

	"cel.dev/cel-go/cel"
)

// QuerySpec is the part of a check.yaml that belongs to a query exercise: the
// learner writes one kubectl command, and k8slings compares what it prints.
type QuerySpec struct {
	// Compare is "output" (the default: the output must be the one the reference
	// answer prints against the same cluster) or "manifest" (the output is YAML,
	// and the exercise's `objects` assertions and schema decide).
	Compare string `json:"compare"`
	// Sort sorts the lines of both outputs before comparing, for exercises where
	// the order is not the point.
	Sort bool `json:"sort"`
	// AllNamespaces lets the command use -A.
	AllNamespaces bool `json:"all_namespaces"`
	// Command are assertions on the command itself, in CEL over `verb` (such as
	// "get" or "create deployment"), `positional` (the words that are not flags)
	// and `flags` (the flags written, by long name, with their values).
	Command []Assertion `json:"command"`
	// VariantSteps say when the extra objects of setup-variant.yaml are ready.
	// They run after the variant is applied and before the command is run again.
	VariantSteps []Step `json:"variant_steps"`
}

// CompareOutput and CompareManifest are the values of Compare.
const (
	CompareOutput   = "output"
	CompareManifest = "manifest"
)

func (c *Compiled) compileQuery() error {
	q := c.Query
	if q == nil {
		return nil
	}
	switch q.Compare {
	case "":
		q.Compare = CompareOutput
	case CompareOutput, CompareManifest:
	default:
		return fmt.Errorf("check.yaml: query.compare must be output or manifest, got %q", q.Compare)
	}
	if q.Compare == CompareManifest && q.Sort {
		return fmt.Errorf("check.yaml: query.sort belongs to compare: output")
	}
	if q.Compare == CompareOutput && len(c.Objects) > 0 {
		return fmt.Errorf("check.yaml: objects are for compare: manifest; an output comparison has the reference answer")
	}
	if q.Compare == CompareManifest && len(c.Objects) == 0 {
		return fmt.Errorf("check.yaml: compare: manifest needs objects to check")
	}
	env, err := cel.NewEnv(
		cel.Variable("verb", cel.StringType),
		cel.Variable("positional", cel.ListType(cel.StringType)),
		cel.Variable("flags", cel.MapType(cel.StringType, cel.StringType)),
	)
	if err != nil {
		return err
	}
	for i, a := range q.Command {
		if strings.TrimSpace(a.Message) == "" {
			return fmt.Errorf("check.yaml: query.command[%d] needs a message", i)
		}
		ast, iss := env.Compile(a.Expr)
		if iss.Err() != nil {
			return fmt.Errorf("check.yaml: query.command[%d]: %w", i, iss.Err())
		}
		if ast.OutputType() != cel.BoolType {
			return fmt.Errorf("check.yaml: query.command[%d] must be a boolean expression", i)
		}
		prg, err := env.Program(ast)
		if err != nil {
			return err
		}
		c.commandProgs = append(c.commandProgs, prg)
	}
	c.variantSteps, err = c.compileStepList(q.VariantSteps, "query.variant_steps")
	return err
}

// RunCommand evaluates the assertions on the command and returns the messages
// of those that do not hold.
func (c *Compiled) RunCommand(verb string, positional []string, flags map[string]string) []string {
	if positional == nil {
		positional = []string{}
	}
	var failures []string
	for i, prg := range c.commandProgs {
		out, _, err := prg.Eval(map[string]any{"verb": verb, "positional": positional, "flags": flags})
		if err != nil || out.Value() != true {
			failures = append(failures, c.Query.Command[i].Message)
		}
	}
	return failures
}

// HasVariantSteps reports whether the check says when the variant is ready.
func (c *Compiled) HasVariantSteps() bool { return len(c.variantSteps) > 0 }
