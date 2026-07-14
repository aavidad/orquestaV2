package config

import (
	"reflect"
	"testing"
)

func TestResolveChildEnvironmentReturnsOnlyRequestedPresentValues(t *testing.T) {
	lookup := mapEnvironment(map[string]string{
		"PATH":        "/safe/bin",
		"CODEX_HOME":  "/safe/codex",
		"EMPTY":       "",
		"UNREQUESTED": "must-not-leak",
	})
	resolved, err := resolveChildEnvironment([]string{"PATH", "CODEX_HOME", "EMPTY"}, lookup)
	if err != nil {
		t.Fatalf("resolve child environment: %v", err)
	}
	want := map[string]string{"PATH": "/safe/bin", "CODEX_HOME": "/safe/codex", "EMPTY": ""}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolved = %#v, want %#v", resolved, want)
	}
	if empty, err := resolveChildEnvironment(nil, nil); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty allowlist = %#v, %v", empty, err)
	}
}

func TestResolveChildEnvironmentRejectsInvalidAndDuplicateNames(t *testing.T) {
	lookup := mapEnvironment(map[string]string{"PRESENT": "value"})
	tests := []struct {
		name   string
		names  []string
		lookup environmentLookup
	}{
		{name: "empty", names: []string{""}, lookup: lookup},
		{name: "whitespace", names: []string{" PRESENT"}, lookup: lookup},
		{name: "invalid equals", names: []string{"PRESENT=OTHER"}, lookup: lookup},
		{name: "duplicate", names: []string{"PRESENT", "PRESENT"}, lookup: lookup},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved, err := resolveChildEnvironment(test.names, test.lookup)
			if resolved != nil {
				t.Fatalf("partial environment leaked: %#v", resolved)
			}
			assertConfigError(t, err, ErrorChildEnvironmentInvalid, "")
		})
	}
}

func TestResolveChildEnvironmentOmitsOptionalMissingNames(t *testing.T) {
	lookup := mapEnvironment(map[string]string{"PATH": "/safe/bin", "HOME": "/safe/home"})
	resolved, err := resolveChildEnvironment([]string{"PATH", "HOME", "CODEX_HOME"}, lookup)
	if err != nil {
		t.Fatalf("resolve optional environment: %v", err)
	}
	want := map[string]string{"PATH": "/safe/bin", "HOME": "/safe/home"}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolved = %#v, want %#v", resolved, want)
	}
	if resolved, err := resolveChildEnvironment([]string{"OPTIONAL"}, nil); err != nil || len(resolved) != 0 {
		t.Fatalf("nil lookup = %#v, %v", resolved, err)
	}
}

func TestResolveChildEnvironmentUsesDedicatedSystemLookup(t *testing.T) {
	const name = "ORQUESTA_CONFIG_CHILD_ENV_TEST"
	t.Setenv(name, "visible")
	resolved, err := ResolveChildEnvironment([]string{name})
	if err != nil {
		t.Fatalf("ResolveChildEnvironment() error = %v", err)
	}
	if !reflect.DeepEqual(resolved, map[string]string{name: "visible"}) {
		t.Fatalf("resolved = %#v", resolved)
	}
}
