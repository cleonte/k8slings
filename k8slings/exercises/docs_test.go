package exercises_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
)

// The README states how many exercises and topics there are. This keeps those
// numbers equal to info.toml, so the docs cannot drift.
func TestREADMECounts(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	all := load(t)
	topics := map[string]bool{}
	for _, ex := range all {
		topics[ex.Topic] = true
	}
	for name, want := range map[string]int{"exercises": len(all), "topics": len(topics)} {
		re := regexp.MustCompile(`<!-- count:` + name + ` -->(\d+)<!-- /count -->`)
		m := re.FindSubmatch(readme)
		if m == nil {
			t.Errorf("README.md has no <!-- count:%s -->N<!-- /count --> marker", name)
			continue
		}
		if got := string(m[1]); got != fmt.Sprint(want) {
			t.Errorf("README.md says %s %s, info.toml has %d", got, name, want)
		}
	}
}
