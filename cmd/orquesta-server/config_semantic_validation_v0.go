package main

import (
	"fmt"
	"strings"
)

// validateServerProjectConfigSemanticV0 contains only deterministic rules over
// decoded data; I/O, environment and startup side effects remain outside it.
func validateServerProjectConfigSemanticV0(config serverProjectConfigFileV0) error {
	if strings.TrimSpace(config.SchemaVersion) != serverProjectConfigSchemaVersionV0 {
		return fmt.Errorf("%s: %s", configFileUnsupportedSchemaCodeV0, serverProjectConfigSchemaVersionV0)
	}
	// Numeric startup accessors have longstanding fallback semantics: zero and
	// negative values behave exactly like an absent value. Do not reject them
	// here or the canonical-file path would diverge from legacy startup.
	return nil
}
