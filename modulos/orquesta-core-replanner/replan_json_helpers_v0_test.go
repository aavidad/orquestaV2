package orquestacorereplanner

import (
	"encoding/json"
	"strings"
	"testing"
)

func assertReplanJSONHasNoForbiddenDetailsV0(t *testing.T, label string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", label, err)
	}
	if !json.Valid(data) {
		t.Fatalf("%s JSON is invalid: %s", label, data)
	}
	serialized := strings.ToLower(string(data))
	for _, forbidden := range []string{
		"provider", "home", "oauth", "db", "database", "model", "runtime",
		"prompt", "prompts", "transcript", "transcripts",
	} {
		if containsForbiddenReplanProposalFragmentV0(serialized, forbidden) {
			t.Fatalf("%s JSON contains forbidden detail %q: %s", label, forbidden, serialized)
		}
	}
}
