package acceptance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV23WizardGapExactReplay(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	applicationSource := readV23WizardSnapshotSource(t, root,
		"internal/application/wizard_gaps.go")
	storeSource := readV23WizardSnapshotSource(t, root,
		"internal/adapters/state/sqlite/wizard_gaps_snapshots.go")
	publicSource := readV23WizardSnapshotSource(t, root,
		"internal/commands/handlers_wizard_gaps.go")
	e2eSource := readV23WizardSnapshotSource(t, root,
		"internal/bootstrap/wizard_gaps_snapshot_e2e_test.go")
	for name, source := range map[string]string{
		"application": applicationSource,
		"store":       storeSource,
		"public":      publicSource,
		"e2e":         e2eSource,
	} {
		if source == "" {
			t.Fatalf("%s source empty", name)
		}
	}
	if !strings.Contains(applicationSource,
		"EvaluationReplayExact: !wizardGapsResultSnapshotEmpty(snapshot)") ||
		!strings.Contains(applicationSource,
			"result.EvaluationReplayExact != hasExactSnapshot") ||
		!strings.Contains(storeSource, "gaps.RestoreResultSnapshot") ||
		!strings.Contains(publicSource, `json:"evaluation_replay_exact"`) ||
		strings.Contains(publicSource, `json:"evaluation_snapshot"`) ||
		!strings.Contains(e2eSource, "bytes.Equal(replayed.Data, first.Data)") ||
		!strings.Contains(e2eSource, "shutdownRuntime(t, firstRuntime)") {
		t.Fatal("exact Wizard replay contract is incomplete")
	}
}

func readV23WizardSnapshotSource(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
