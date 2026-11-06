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
	checkCounts(t, "README.md", true)
	checkCounts(t, "CLAUDE.md", true)
	checkCounts(t, "ROADMAP.md", false) // marks only the counts it mentions

}

func checkCounts(t *testing.T, doc string, required bool) {
	t.Helper()
	readme, err := os.ReadFile("../../" + doc)
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
			if required {
				t.Errorf("%s has no <!-- count:%s -->N<!-- /count --> marker", doc, name)
			}
			continue
		}
		if got := string(m[1]); got != fmt.Sprint(want) {
			t.Errorf("%s says %s %s, info.toml has %d", doc, got, name, want)
		}
	}
}
