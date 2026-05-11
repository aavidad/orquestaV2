package orquestacionnucleoapp

import (
	"reflect"
	"testing"
)

func TestResolveWorksetAliasesV0(t *testing.T) {
	t.Run("alias conocido devuelve rutas normalizadas", func(t *testing.T) {
		paths, issues := ResolveWorksetAliasesV0(
			map[string][]string{
				"app": {" orquestacionnucleoapp/./service.go "},
			},
			[]string{" app "},
		)

		assertWorksetAliasPathsV0(t, paths, []string{"orquestacionnucleoapp/service.go"})
		assertWorksetAliasIssuesV0(t, issues, nil)
	})

	t.Run("alias desconocido produce issue publico", func(t *testing.T) {
		paths, issues := ResolveWorksetAliasesV0(
			map[string][]string{"app": {"orquestacionnucleoapp/service.go"}},
			[]string{"missing"},
		)

		assertWorksetAliasPathsV0(t, paths, nil)
		assertWorksetAliasIssueCodesV0(t, issues, []string{WorksetAliasIssueUnknownAliasV0})
		if issues[0].Alias != "missing" {
			t.Fatalf("issue=%+v", issues[0])
		}
	})

	t.Run("alias vacio produce issue publico", func(t *testing.T) {
		paths, issues := ResolveWorksetAliasesV0(
			map[string][]string{"app": {"orquestacionnucleoapp/service.go"}},
			[]string{"  "},
		)

		assertWorksetAliasPathsV0(t, paths, nil)
		assertWorksetAliasIssueCodesV0(t, issues, []string{WorksetAliasIssueEmptyAliasV0})
	})

	t.Run("alias duplicados no duplican rutas", func(t *testing.T) {
		paths, issues := ResolveWorksetAliasesV0(
			map[string][]string{
				"app": {
					"orquestacionnucleoapp/./service.go",
					"orquestacionnucleoapp/service.go",
				},
				"tests": {
					"orquestacionnucleoapp/service.go",
					"orquestacionnucleoapp/service_test.go",
				},
			},
			[]string{"app", "tests", "app"},
		)

		assertWorksetAliasPathsV0(t, paths, []string{
			"orquestacionnucleoapp/service.go",
			"orquestacionnucleoapp/service_test.go",
		})
		assertWorksetAliasIssuesV0(t, issues, nil)
	})
}

func assertWorksetAliasPathsV0(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths=%+v, want %+v", got, want)
	}
}

func assertWorksetAliasIssuesV0(t *testing.T, got []WorksetAliasIssueV0, want []WorksetAliasIssueV0) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues=%+v, want %+v", got, want)
	}
}

func assertWorksetAliasIssueCodesV0(t *testing.T, got []WorksetAliasIssueV0, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("issues=%+v, want codes %+v", got, want)
	}
	for index := range want {
		if got[index].Code != want[index] {
			t.Fatalf("issue[%d]=%+v, want code %s", index, got[index], want[index])
		}
	}
}
