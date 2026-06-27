package orquestaweb

import (
	"strings"
	"testing"
)

func TestAutoprogrammingHTMLV0RenderizaSelectorGoalBatch(t *testing.T) {
	body := autoprogrammingHTMLV0()

	for _, want := range []string{
		`id="goal-list"`,
		`function normalizedGoals(value)`,
		`button.dataset.goalIndex`,
		`currentGoalIndex = index`,
		`observeGoal(true)`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML autoprogramming no contiene %q", want)
		}
	}
	if strings.Contains(body, `((value || {}).goals || [])[0]`) {
		t.Fatalf("HTML autoprogramming vuelve a fijar silenciosamente goals[0]")
	}
}
