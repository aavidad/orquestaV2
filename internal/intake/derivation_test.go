package intake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDerivationIdentityIsValidatedAndRecordedInSingleMutationHistory(
	t *testing.T,
) {
	identity, err := NewDerivationIdentity(
		"orquesta.test.deriver",
		"v1",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	state := mustState(t, 2)
	next, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{
			audienceQuestion(true, false),
		},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	history := next.History()
	if len(history) != 1 || history[0].Derivation != identity {
		t.Fatalf("history=%+v want derivation=%+v", history, identity)
	}

	for name, invalid := range map[string]DerivationIdentity{
		"partial": {
			Schema: "orquesta.test.deriver",
		},
		"schema": {
			Schema: "not_a_schema", Version: "v1",
			SemanticDigest: strings.Repeat("a", 64),
		},
		"version": {
			Schema: "orquesta.test.deriver", Version: "V 1",
			SemanticDigest: strings.Repeat("a", 64),
		},
		"digest length": {
			Schema: "orquesta.test.deriver", Version: "v1",
			SemanticDigest: strings.Repeat("a", 63),
		},
		"digest case": {
			Schema: "orquesta.test.deriver", Version: "v1",
			SemanticDigest: strings.Repeat("A", 64),
		},
	} {
		t.Run(name, func(t *testing.T) {
			change := Change{
				StateRef: testStateRef, ExpectedRevision: 1,
				Origin: OriginChat, Issues: []Issue{audienceGap()},
				Questions:  []Question{audienceQuestion(true, false)},
				Derivation: invalid,
			}
			if _, applyErr := Apply(state, change); ErrorCodeOf(applyErr) != ErrorInvalidArgument {
				t.Fatalf("error=%v code=%q", applyErr, ErrorCodeOf(applyErr))
			}
		})
	}
}

func TestHistoricalMutationOmitsZeroDerivationFromCanonicalJSON(t *testing.T) {
	change := Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{
			audienceQuestion(true, false),
		},
	}
	encoded, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "derivation") {
		t.Fatalf("historical change JSON gained derivation: %s", encoded)
	}
	state, err := Apply(mustState(t, 2), change)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err = json.Marshal(state.History()[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "derivation") {
		t.Fatalf("historical mutation JSON gained derivation: %s", encoded)
	}
}
