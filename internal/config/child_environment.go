package config

import (
	"fmt"
	"strings"
)

func resolveChildEnvironment(names []string, lookup environmentLookup) (map[string]string, error) {
	result := make(map[string]string, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if !validChildEnvironmentName(name) {
			return nil, &Error{Code: ErrorChildEnvironmentInvalid, Cause: fmt.Errorf("invalid environment name")}
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, &Error{Code: ErrorChildEnvironmentInvalid, Cause: fmt.Errorf("duplicate environment name")}
		}
		seen[name] = struct{}{}
		if lookup == nil {
			continue
		}
		value, present := lookup(name)
		if !present {
			continue
		}
		result[name] = value
	}
	return result, nil
}

func validChildEnvironmentName(name string) bool {
	return validEnvironmentName(name) && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "=\x00")
}
