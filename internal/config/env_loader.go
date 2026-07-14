package config

import "os"

type environmentLookup func(string) (string, bool)

// lookupSystemEnvironment is the only process-environment read in the package.
func lookupSystemEnvironment(name string) (string, bool) {
	return os.LookupEnv(name)
}

// CaptureEnvironment reads only names declared by the canonical registry.
// Callers pass this detached map to Resolve; no other package reads globals.
func CaptureEnvironment() (map[string]string, error) {
	registry, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, definition := range registry.keys {
		if value, present := lookupSystemEnvironment(definition.EnvAlias); present {
			result[definition.EnvAlias] = value
		}
	}
	for _, alias := range registry.aliases {
		if alias.Kind != AliasKindEnvironment {
			continue
		}
		if value, present := lookupSystemEnvironment(alias.Name); present {
			result[alias.Name] = value
		}
	}
	return result, nil
}

// ResolveChildEnvironment copies only explicitly named process variables.
func ResolveChildEnvironment(names []string) (map[string]string, error) {
	return resolveChildEnvironment(names, lookupSystemEnvironment)
}
