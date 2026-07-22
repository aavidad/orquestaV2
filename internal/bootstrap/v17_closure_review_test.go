package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV17ProductionCompositionWiresIndependentTestAttestor(t *testing.T) {
	compositionSource := v17ReadProductionTree(t, ".")
	for _, marker := range []string{
		`"orquesta/internal/adapters/attestor/bubblewrap"`,
		"openBuildTestAttestor(",
		"bubblewrap.New(config)",
		"TestAttestorProvider()",
		"TestAttestorBubblewrapCommand()",
		"TestAttestorGoToolchainRoot()",
		"RuntimeMaxOutputBytes()",
		"TestAttestorMaxSubjectBytes()",
		"TestAttestor:",
		"TestAttestationPolicy:",
	} {
		if !strings.Contains(compositionSource, marker) {
			t.Errorf("V17_GATE_PRODUCT_WIRING_PENDING: production composition lacks %q", marker)
		}
	}
	if strings.Count(compositionSource, "bubblewrap.New(config)") != 1 ||
		strings.Contains(compositionSource, "bubblewrap.Prepare(") || strings.Contains(compositionSource, "*bubblewrap.Prepared") {
		t.Fatal("V17_GATE_SINGLE_ADAPTER: Build must create one direct attestor")
	}
}

func TestV17ArchitectureKeepsOneWriterAndNoAttestorAuthority(t *testing.T) {
	source := v17ReadProductionTree(t, "../application", "../ports")
	for _, forbidden := range []string{
		"type TestLifecycle ", "type AttestationLifecycle ",
		"type TestStore ", "type AttestationStore ", "type EvidenceStore ",
		"type TestQueue ", "type AttestationQueue ",
		"type TestScheduler ", "type AttestationScheduler ",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("V17 duplicate private authority found: %q", forbidden)
		}
	}
	if !strings.Contains(source, "RecordTestAttested(context.Context, TestAttestedState) error") ||
		!strings.Contains(source, "func (orchestrator *Orchestrator) processAttestTest(") {
		t.Fatal("V17 attestation must remain one application transition through the canonical StateRepository")
	}
}

func TestV17SecurityPrimitivesAreWiredNotDeadCode(t *testing.T) {
	bubblewrap := v17ReadProductionTree(t, "../adapters/attestor/bubblewrap")
	for _, marker := range []string{
		"openPinnedInputs(", "sealExecutable(", "fd-pinned-inputs",
		"readSnapshotStream(", "sealedSubjectFile(", "UseCgroupFD",
		`"--remount-ro"`, `"--cap-drop"`, `"--disable-userns"`, `"--assert-userns-disabled"`,
	} {
		if !strings.Contains(bubblewrap, marker) {
			t.Errorf("V17_GATE_STREAM_SANDBOX_PENDING: production bubblewrap lacks %q", marker)
		}
	}
	if strings.Contains(bubblewrap, "WorkspacePathResolver") {
		t.Error("V17_GATE_TRUSTED_INPUT_WIRING_PENDING: production bubblewrap does not capture and execute pinned inputs")
	}
	for _, forbidden := range []string{
		"MountSetattr(", "MOUNT_ATTR_RDONLY", "CAP_SYS_ADMIN", "CAP_SETPCAP",
		"dropAllCapabilities(", "extractSnapshotStream(", `"--cap-add"`, "supervisor",
	} {
		if strings.Contains(bubblewrap, forbidden) {
			t.Errorf("V17_GATE_REJECTED_SANDBOX_PRESENT: production bubblewrap contains %q", forbidden)
		}
	}
	gitlocal := v17ReadProductionTree(t, "../adapters/workspace/gitlocal")
	for _, marker := range []string{"OpenSnapshotStream(", "cat-file", "snapshotObjectHash("} {
		if !strings.Contains(gitlocal, marker) {
			t.Errorf("V17_GATE_GIT_OBJECT_STREAM_PENDING: gitlocal lacks %q", marker)
		}
	}
	if strings.Contains(gitlocal, "MaterializeVerifiedTestWorkspace(") {
		t.Error("V17_GATE_WORKTREE_AUTHORITY_PRESENT: insecure materialized worktree remains")
	}
}

func TestV17ObjectStreamHasNoDiscardedPostVerificationResult(t *testing.T) {
	source := v17ReadProductionTree(t, "../application", "../ports", "../adapters/attestor/bubblewrap")
	if strings.Contains(source, "VerifySnapshot(") || strings.Contains(source, "SnapshotVerificationResult") {
		t.Fatal("superseded diagnostic snapshot verification remains in V17 hot path")
	}
}

func TestAttestationUsesOneSnapshotCaptureAndNoDiagnosticVerification(t *testing.T) {
	application := v17ReadProductionTree(t, "../application")
	attestor := v17ReadProductionTree(t, "../adapters/attestor/bubblewrap")
	if strings.Contains(application, "OpenSnapshotStream(") || strings.Contains(application, "VerifySnapshot(") ||
		strings.Count(attestor, "OpenSnapshotStream(ctx, run.Snapshot)") != 1 || strings.Contains(attestor, "VerifySnapshot(") {
		t.Fatal("attestation must consume one object stream and no diagnostic traversal")
	}
}

func TestV17RuntimeOwnsAndClosesWorkspaceAdapter(t *testing.T) {
	source := v17ReadProductionTree(t, ".")
	for _, marker := range []string{
		"cleanup.add(func() { _ = workspace.Close() })",
		"testAttestor.closer, workspace,",
		"testAttestor: testAttestor, workspace: workspace,",
		"runtime.testAttestor.Close()",
		"runtime.workspace.Close()",
		"runtime.repository.Close()",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("runtime ownership/cleanup missing %q", marker)
		}
	}
	if strings.Index(source, "runtime.testAttestor.Close()") > strings.Index(source, "runtime.workspace.Close()") ||
		strings.Index(source, "runtime.workspace.Close()") > strings.Index(source, "runtime.repository.Close()") {
		t.Fatal("runtime shutdown order must be attestor, workspace, repository")
	}
}

func v17ReadProductionTree(t *testing.T, directories ...string) string {
	t.Helper()
	var source strings.Builder
	for _, directory := range directories {
		matches, err := filepath.Glob(filepath.Join(directory, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range matches {
			if strings.HasSuffix(match, "_test.go") {
				continue
			}
			content, err := os.ReadFile(match)
			if err != nil {
				t.Fatal(err)
			}
			source.Write(content)
		}
	}
	return source.String()
}
