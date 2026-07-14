package config

import "os"

type environmentLookup func(string) (string, bool)

// lookupSystemEnvironment is the only process-environment read in the package.
func lookupSystemEnvironment(name string) (string, bool) {
	return os.LookupEnv(name)
}

// ResolveChildEnvironment copies only explicitly named process variables.
func ResolveChildEnvironment(names []string) (map[string]string, error) {
	return resolveChildEnvironment(names, lookupSystemEnvironment)
}
