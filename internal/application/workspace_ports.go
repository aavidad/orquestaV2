package application

import (
	"context"
	"errors"
	"strings"

	"orquesta/internal/ports"
)

// WorkspaceManager is the application-owned outbound boundary. Its request
// and result values remain neutral port DTOs; adapters never receive lifecycle
// state or filesystem authority through this interface.
type WorkspaceManager interface {
	Prepare(context.Context, ports.WorkspacePrepareRequest) (ports.WorkspacePrepared, error)
	Inspect(context.Context, ports.WorkspaceInspectRequest) (ports.WorkspaceInspection, error)
	Release(context.Context, ports.WorkspaceReleaseRequest) (ports.WorkspaceReleaseReceipt, error)
}

// VersionControl is the application-owned outbound boundary for immutable
// commits and explicit integration. Goal lifecycle remains owned by Orchestrator.
type VersionControl interface {
	Commit(context.Context, ports.CommitRequest) (ports.CommitResult, error)
	PreviewIntegration(context.Context, ports.IntegrationPreviewRequest) (ports.IntegrationPreview, error)
	Integrate(context.Context, ports.IntegrationRequest) (ports.IntegrationResult, error)
}

type TestAttestationRun = ports.TestAttestationRun

// TestAttestor executes declared tests for one immutable logical run.
type TestAttestor interface {
	Attest(context.Context, ports.TestAttestationRun) (ports.TestAttestationResult, error)
}

type testAttestorCauseError interface {
	error
	CauseCode() string
}

func testAttestorCauseCode(err error) string {
	var cause testAttestorCauseError
	if !errors.As(err, &cause) {
		return ""
	}
	code := cause.CauseCode()
	if len(code) == 0 || len(code) > 160 || !strings.HasPrefix(code, "test_attestor.") {
		return ""
	}
	for _, character := range code {
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return ""
	}
	return code
}

func ValidateTestAttestationRun(run TestAttestationRun) error {
	if err := ports.ValidateTestAttestationRequest(run.Request); err != nil {
		return err
	}
	if err := ports.ValidateSnapshotVerificationRequest(run.Snapshot); err != nil {
		return err
	}
	if run.Request.Subject != run.Snapshot.Subject ||
		run.Request.SubjectDigest != run.Snapshot.SubjectDigest {
		return &ports.TestAttestorContractError{Code: "test_attestor.run_subject_mismatch"}
	}
	return nil
}
