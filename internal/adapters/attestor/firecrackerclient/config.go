// Package firecrackerclient implements the unprivileged Firecracker-backed
// TestAttestor. Privileged launch and transport stay behind the injected
// launcher contract.
package firecrackerclient

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
	"orquesta/internal/testattestorprotocol/rawdrive"
)

const (
	AttestorRef = "attestor:firecracker:local:v1"
	PolicyRef   = "policy:test-attestor:firecracker:v1"

	snapshotCopyBlockSize = 128 << 10
)

const (
	codeUnavailable       = "test_attestor.unavailable"
	codeConfigInvalid     = "test_attestor.config_invalid"
	codePolicyMismatch    = "test_attestor.policy_mismatch"
	codeToolUnsupported   = "test_attestor.tool_unsupported"
	codeSnapshotInvalid   = "test_attestor.snapshot_stream_invalid"
	codeSnapshotLimit     = "test_attestor.snapshot_limit_exceeded"
	codeResourceUnsafe    = "test_attestor.resource_governor_unsafe"
	codeResourceLimit     = "test_attestor.resource_budget_exceeded"
	codeLauncherIdentity  = "test_attestor.launcher_unauthorized"
	codeAssetMismatch     = "test_attestor.asset_identity_mismatch"
	codeInputInvalid      = "test_attestor.input_invalid"
	codeOutputInvalid     = "test_attestor.output_invalid"
	codeExecutionFailed   = "test_attestor.execution_failed"
	codeExecutionTimeout  = "test_attestor.execution_timeout"
	codeOutputLimit       = "test_attestor.output_limit_exceeded"
	codeIsolationIdentity = "test_attestor.isolation_identity_unsafe"
	codeNetworkUnsafe     = "test_attestor.isolation_network_unsafe"
	codeCleanupFailed     = "test_attestor.cleanup_failed"
	codeClockInvalid      = "test_attestor.timestamps_invalid"
)

// SnapshotStreamSource is a trusted composition dependency. OpenSnapshotStream
// and Close must cooperate with context cancellation; the adapter bounds its
// own shutdown but cannot safely force an arbitrary external implementation.
type SnapshotStreamSource interface {
	OpenSnapshotStream(context.Context, ports.SnapshotVerificationRequest) (io.ReadCloser, error)
	SnapshotIdentity() (string, string)
}

type Limits struct {
	Timeout           time.Duration
	CleanupTimeout    time.Duration
	MaxOutputBytes    uint64
	MaxSubjectBytes   int64
	MaxConcurrentRuns int
	GuestMemoryMiB    uint32
	MemoryMaxBytes    uint64
	PIDsMax           uint32
	CPUQuotaMicros    uint64
}

type Config struct {
	Launcher            launcherprotocol.Client
	SnapshotSource      SnapshotStreamSource
	Now                 func() time.Time
	ExpectedAssetDigest string
	Limits              Limits
}

type PolicyIdentity struct {
	Ref    string
	Digest string
}

func validateConfig(config Config) error {
	limits := config.Limits
	if config.Launcher == nil || config.SnapshotSource == nil || config.Now == nil ||
		!validDigest(config.ExpectedAssetDigest) ||
		limits.Timeout <= 0 || limits.CleanupTimeout <= 0 ||
		limits.CleanupTimeout >= limits.Timeout ||
		limits.MaxOutputBytes == 0 ||
		limits.MaxOutputBytes > rawdrive.MaxCapturedOutputBytes ||
		limits.MaxSubjectBytes < int64(len(snapshotMagic)) ||
		uint64(limits.MaxSubjectBytes) > rawdrive.MaxSnapshotBytes ||
		limits.MaxConcurrentRuns <= 0 || limits.GuestMemoryMiB == 0 ||
		limits.MemoryMaxBytes == 0 || limits.PIDsMax == 0 ||
		limits.CPUQuotaMicros == 0 {
		return contractError(codeConfigInvalid)
	}
	return nil
}

func configuredPolicyIdentity(
	config Config,
	snapshotIdentity, launcherIdentity launcherprotocol.Identity,
	effectiveUID, effectiveGID int,
) PolicyIdentity {
	digest := sha256.New()
	for _, value := range []string{
		"orquesta.test-attestor.firecracker.policy.v1",
		PolicyRef,
		AttestorRef,
		"launcher:v1;raw-drive:v1;sealed-input;private-sealed-output;no-network;no-vsock;no-serial",
		config.ExpectedAssetDigest,
		snapshotIdentity.Ref,
		snapshotIdentity.Digest,
		launcherIdentity.Ref,
		launcherIdentity.Digest,
		strconv.Itoa(effectiveUID),
		strconv.Itoa(effectiveGID),
		config.Limits.Timeout.String(),
		config.Limits.CleanupTimeout.String(),
		strconv.FormatUint(config.Limits.MaxOutputBytes, 10),
		strconv.FormatInt(config.Limits.MaxSubjectBytes, 10),
		strconv.Itoa(config.Limits.MaxConcurrentRuns),
		strconv.FormatUint(uint64(config.Limits.GuestMemoryMiB), 10),
		strconv.FormatUint(config.Limits.MemoryMaxBytes, 10),
		strconv.FormatUint(uint64(config.Limits.PIDsMax), 10),
		strconv.FormatUint(config.Limits.CPUQuotaMicros, 10),
		strconv.FormatUint(launcherprotocol.CPUPeriodMicrosV0, 10),
	} {
		writeDigestField(digest, value)
	}
	return PolicyIdentity{Ref: PolicyRef, Digest: hex.EncodeToString(digest.Sum(nil))}
}

func snapshotIdentity(source SnapshotStreamSource) launcherprotocol.Identity {
	ref, digest := source.SnapshotIdentity()
	return launcherprotocol.Identity{Ref: ref, Digest: digest}
}

func validDependencyIdentity(identity launcherprotocol.Identity) bool {
	return validReference(identity.Ref) && validDigest(identity.Digest)
}

func writeDigestField(digest hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}

func validateRun(run ports.TestAttestationRun, policy PolicyIdentity) error {
	if err := ports.ValidateTestAttestationRequest(run.Request); err != nil {
		return err
	}
	if err := ports.ValidateSnapshotVerificationRequest(run.Snapshot); err != nil ||
		run.Request.Subject != run.Snapshot.Subject ||
		run.Request.SubjectDigest != run.Snapshot.SubjectDigest {
		return contractError("test_attestor.run_subject_mismatch")
	}
	if run.Request.Subject.PolicyRef != policy.Ref ||
		run.Request.Subject.PolicyDigest != policy.Digest {
		return contractError(codePolicyMismatch)
	}
	for _, spec := range run.Request.RequiredTests {
		if spec.ToolRef().String() != "tool:go" {
			return contractError(codeToolUnsupported)
		}
	}
	return nil
}

func outputDriveBytes(run ports.TestAttestationRun) (uint64, error) {
	outcomes := make([]rawdrive.RequiredTestOutcome, 0, len(run.Request.RequiredTests))
	for _, spec := range run.Request.RequiredTests {
		outcomes = append(outcomes, rawdrive.RequiredTestOutcome{
			RequiredTestRef: spec.Ref().String(),
			ExitCode:        1,
			OutputDigest:    strings.Repeat("0", sha256.Size*2),
		})
	}
	resultBytes, err := rawdrive.OutputDriveSize(rawdrive.Output{
		SubjectDigest: run.Request.SubjectDigest,
		RunNonce:      strings.Repeat("0", sha256.Size*2),
		Verdict:       rawdrive.VerdictFailed,
		Outcomes:      outcomes,
	})
	if err != nil {
		return 0, contractError(codeConfigInvalid)
	}
	errorBytes, err := rawdrive.OutputDriveSize(rawdrive.Output{
		SubjectDigest: run.Request.SubjectDigest,
		RunNonce:      strings.Repeat("0", sha256.Size*2),
		ErrorCode:     "test_attestor." + strings.Repeat("x", rawdrive.MaxErrorCodeBytes-len("test_attestor.")),
	})
	if err != nil {
		return 0, contractError(codeConfigInvalid)
	}
	if errorBytes > resultBytes {
		return errorBytes, nil
	}
	return resultBytes, nil
}

func toRawInput(run ports.TestAttestationRun, nonce string, maximumOutput uint64) rawdrive.Input {
	tests := make([]rawdrive.RequiredTest, 0, len(run.Request.RequiredTests))
	for _, spec := range run.Request.RequiredTests {
		tests = append(tests, rawdrive.RequiredTest{
			Ref:              spec.Ref().String(),
			ToolRef:          spec.ToolRef().String(),
			Arguments:        spec.Arguments(),
			WorkingDirectory: spec.WorkingDirectory(),
		})
	}
	return rawdrive.Input{
		SubjectDigest:  run.Request.SubjectDigest,
		RunNonce:       nonce,
		PolicyRef:      run.Request.Subject.PolicyRef,
		PolicyDigest:   run.Request.Subject.PolicyDigest,
		MaxOutputBytes: maximumOutput,
		RequiredTests:  tests,
	}
}

func contractError(code string) error {
	return &ports.TestAttestorContractError{Code: code}
}

type causedError interface {
	error
	CauseCode() string
}

func translateFailure(err error) error {
	if err == nil {
		return nil
	}
	var contract *ports.TestAttestorContractError
	if errors.As(err, &contract) {
		return contractError(translateCauseCode(contract.CauseCode()))
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return contractError(codeExecutionTimeout)
	}
	if errors.Is(err, context.Canceled) {
		return contractError(codeUnavailable)
	}
	var caused causedError
	if !errors.As(err, &caused) {
		return contractError(codeUnavailable)
	}
	return contractError(translateCauseCode(caused.CauseCode()))
}

func translateCauseCode(code string) string {
	if translated, found := rawCauseTranslations[code]; found {
		return translated
	}
	if allowedContractCause(code) {
		return code
	}
	return codeUnavailable
}

func allowedContractCause(code string) bool {
	switch code {
	case codeUnavailable,
		codeConfigInvalid,
		codePolicyMismatch,
		codeToolUnsupported,
		codeSnapshotInvalid,
		codeSnapshotLimit,
		codeResourceUnsafe,
		codeResourceLimit,
		codeLauncherIdentity,
		codeAssetMismatch,
		codeInputInvalid,
		codeOutputInvalid,
		codeExecutionFailed,
		codeExecutionTimeout,
		codeOutputLimit,
		codeIsolationIdentity,
		codeNetworkUnsafe,
		codeCleanupFailed,
		codeClockInvalid,
		"test_attestor.run_subject_mismatch",
		"test_attestor.subject_digest_mismatch",
		"test_attestor.request_invalid",
		"test_attestor.required_tests_invalid",
		"test_attestor.required_tests_mismatch",
		"test_attestor.artifact_invalid",
		"test_attestor.identity_mismatch",
		"test_attestor.subject_mismatch",
		"test_attestor.verdict_invalid",
		"test_attestor.outcomes_invalid",
		"test_attestor.verdict_mismatch",
		"test_attestor.subject_identity_invalid",
		"test_attestor.subject_binding_invalid",
		"test_attestor.manifest_invalid",
		"test_attestor.report_invalid":
		return true
	default:
		return false
	}
}

var rawCauseTranslations = map[string]string{
	"test_attestor.firecracker_launcher_unavailable":         codeUnavailable,
	"test_attestor.firecracker_launcher_config_invalid":      codeConfigInvalid,
	"test_attestor.firecracker_launcher_privilege_required":  codeConfigInvalid,
	"test_attestor.firecracker_launcher_peer_unauthorized":   codeLauncherIdentity,
	"test_attestor.firecracker_launcher_server_unauthorized": codeLauncherIdentity,
	"test_attestor.firecracker_launcher_protocol_invalid":    codeOutputInvalid,
	"test_attestor.firecracker_launcher_descriptor_invalid":  codeOutputInvalid,
	"test_attestor.firecracker_launcher_input_invalid":       codeInputInvalid,
	"test_attestor.firecracker_launcher_output_invalid":      codeOutputInvalid,
	"test_attestor.firecracker_launcher_assets_unsafe":       codeAssetMismatch,
	"test_attestor.firecracker_launcher_network_unsafe":      codeNetworkUnsafe,
	"test_attestor.firecracker_launcher_runtime_root_unsafe": codeResourceUnsafe,
	"test_attestor.firecracker_launcher_resource_unsafe":     codeResourceUnsafe,
	"test_attestor.firecracker_launcher_execution_failed":    codeExecutionFailed,
	"test_attestor.firecracker_launcher_execution_timeout":   codeExecutionTimeout,
	"test_attestor.firecracker_launcher_cleanup_failed":      codeCleanupFailed,
	"test_attestor.firecracker_launcher_response_invalid":    codeOutputInvalid,

	rawdrive.CodeInvalid:          codeOutputInvalid,
	rawdrive.CodeLimit:            codeResourceLimit,
	rawdrive.CodeTruncated:        codeOutputInvalid,
	rawdrive.CodeTampered:         codeOutputInvalid,
	rawdrive.CodeTrailingData:     codeOutputInvalid,
	rawdrive.CodeAlignment:        codeOutputInvalid,
	rawdrive.CodeIO:               codeOutputInvalid,
	rawdrive.CodeMetadataInvalid:  codeInputInvalid,
	rawdrive.CodeSnapshotInvalid:  codeSnapshotInvalid,
	rawdrive.CodeOutputInvalid:    codeOutputInvalid,
	rawdrive.CodeOutputIncomplete: codeOutputInvalid,

	"test_attestor.firecracker.guest_input_invalid":       codeInputInvalid,
	"test_attestor.firecracker.guest_snapshot_invalid":    codeSnapshotInvalid,
	"test_attestor.firecracker.guest_snapshot_limit":      codeSnapshotLimit,
	"test_attestor.firecracker.guest_materialize_failed":  codeSnapshotInvalid,
	"test_attestor.firecracker.guest_tool_unsupported":    codeToolUnsupported,
	"test_attestor.firecracker.guest_execution_failed":    codeExecutionFailed,
	"test_attestor.firecracker.guest_execution_timeout":   codeExecutionTimeout,
	"test_attestor.firecracker.guest_output_limit":        codeOutputLimit,
	"test_attestor.firecracker.guest_no_tests":            "test_attestor.required_tests_invalid",
	"test_attestor.firecracker.guest_identity_failed":     codeIsolationIdentity,
	"test_attestor.firecracker.guest_network_available":   codeNetworkUnsafe,
	"test_attestor.firecracker.guest_output_write_failed": codeOutputInvalid,
	"test_attestor.firecracker.guest_scratch_unavailable": codeResourceUnsafe,
	"test_attestor.firecracker.guest_scratch_capacity":    codeResourceLimit,
	"test_attestor.firecracker.guest_cleanup_failed":      codeCleanupFailed,
}

func validDigest(value string) bool {
	return len(value) == sha256.Size*2 &&
		strings.Trim(value, "0123456789abcdef") == ""
}

func validReference(value string) bool {
	return value != "" && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

var snapshotMagic = []byte("ORQ-SNAPSHOT-2\x00")
