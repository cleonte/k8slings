package check

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
)

// quantityFunction adds quantity("64Mi") to CEL: the value of a Kubernetes
// quantity as a number (bytes, cores), so that assertions can compare
// resources written with different suffixes: "500m" against "1", "64Mi" against
// "1Gi". Plain CEL cannot, and comparing the strings is wrong ("9Mi" > "10Mi").
func quantityFunction() cel.EnvOption {
	return cel.Function("quantity",
		cel.Overload("quantity_string", []*cel.Type{cel.StringType}, cel.DoubleType,
			cel.UnaryBinding(func(v ref.Val) ref.Val {
				s, ok := v.Value().(string)
				if !ok {
					return types.NewErr("quantity() wants a string")
				}
				q, err := ParseQuantity(s)
				if err != nil {
					return types.NewErr("%v", err)
				}
				return types.Double(q)
			})),
		// `cpu: 1` and `cpu: 0.5` are valid in a manifest: YAML reads them as
		// numbers, and Kubernetes takes them as the quantities "1" and "0.5".
		cel.Overload("quantity_int", []*cel.Type{cel.IntType}, cel.DoubleType,
			cel.UnaryBinding(func(v ref.Val) ref.Val {
				n, ok := v.Value().(int64)
				if !ok {
					return types.NewErr("quantity() wants a string or a number")
				}
				return types.Double(float64(n))
			})),
		cel.Overload("quantity_double", []*cel.Type{cel.DoubleType}, cel.DoubleType,
			cel.UnaryBinding(func(v ref.Val) ref.Val {
				f, ok := v.Value().(float64)
				if !ok {
					return types.NewErr("quantity() wants a string or a number")
				}
				return types.Double(f)
			})))
}

var quantitySuffixes = []struct {
	suffix string
	factor float64
}{
	// Longest first, so that "Mi" is not read as "M".
	{"Ki", 1 << 10}, {"Mi", 1 << 20}, {"Gi", 1 << 30}, {"Ti", 1 << 40}, {"Pi", 1 << 50}, {"Ei", 1 << 60},
	{"n", 1e-9}, {"u", 1e-6}, {"m", 1e-3},
	{"k", 1e3}, {"M", 1e6}, {"G", 1e9}, {"T", 1e12}, {"P", 1e15}, {"E", 1e18},
}

// ParseQuantity reads a Kubernetes quantity: a number with an optional decimal
// (k, M, G, m, ...) or binary (Ki, Mi, Gi, ...) suffix, or an exponent (1e3).
func ParseQuantity(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty quantity")
	}
	for _, sfx := range quantitySuffixes {
		if strings.HasSuffix(s, sfx.suffix) {
			n, err := strconv.ParseFloat(strings.TrimSuffix(s, sfx.suffix), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid quantity %q", s)
			}
			return n * sfx.factor, nil
		}
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid quantity %q", s)
	}
	return n, nil
}
