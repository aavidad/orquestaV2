package orquestaruncoordinator

import (
	"reflect"
	"testing"
)

func executionRefsV0(executions []RunExecutionSummaryV0) []string {
	refs := make([]string, 0, len(executions))
	for _, execution := range executions {
		refs = append(refs, execution.RunRef)
	}
	return refs
}

func skipRefsV0(skips []RunSkipSummaryV0) []string {
	refs := make([]string, 0, len(skips))
	for _, skip := range skips {
		refs = append(refs, skip.RunRef)
	}
	return refs
}

func rankedRefsV0(ranked []RankedRunSummaryV0) []string {
	refs := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		refs = append(refs, candidate.RunRef)
	}
	return refs
}

func assertRunRefsV0(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("run refs\nwant: %#v\ngot:  %#v", want, got)
	}
}
