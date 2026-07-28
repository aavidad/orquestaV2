package gaps

import "orquesta/internal/wizard/catalog"

// CanonicalContext validates caller-supplied facts and returns the
// order-independent explicit pack selection used by every evaluator. It does
// not run gap detection and therefore is safe for request identity/replay.
func CanonicalContext(
	facts Facts,
	packRefs []catalog.PackRef,
) (Facts, []catalog.PackRef, error) {
	if err := validateFacts(facts); err != nil {
		return Facts{}, nil, err
	}
	canonical, err := validatePackRefs(packRefs)
	if err != nil {
		return Facts{}, nil, err
	}
	return facts, canonical, nil
}
