package orquestacorereplanner

import (
	"encoding/json"
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
	serialized := string(data)
	if replanProposalHasForbiddenDetailsV0([]string{serialized}) {
		t.Fatalf("%s JSON contains sensitive detail: %s", label, serialized)
	}
}
