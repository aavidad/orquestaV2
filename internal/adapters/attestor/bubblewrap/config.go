// Package bubblewrap implements the local fail-closed TestAttestor.
package bubblewrap

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
)

const (
	AttestorRef, PolicyRef                                    = "attestor:bubblewrap:local:v3", "policy:test-attestor:bubblewrap:v3"
	CodeUnavailable, CodeConfigInvalid                        = "test_attestor.bubblewrap_unavailable", "test_attestor.bubblewrap_config_invalid"
	CodeIdentityUnsafe, CodeBinaryUnsafe                      = "test_attestor.bubblewrap_identity_unsafe", "test_attestor.bubblewrap_binary_unsafe"
	CodeToolchainUnsafe, CodeToolUnsupported                  = "test_attestor.bubblewrap_toolchain_unsafe", "test_attestor.bubblewrap_tool_unsupported"
	CodePolicyMismatch, CodeClockInvalid                      = "test_attestor.bubblewrap_policy_mismatch", "test_attestor.bubblewrap_clock_invalid"
	CodeExecutionFailed, CodeExecutionTimeout                 = "test_attestor.bubblewrap_execution_failed", "test_attestor.bubblewrap_execution_timeout"
	CodeOutputLimit, CodeCleanupFailed                        = "test_attestor.bubblewrap_output_limit", "test_attestor.bubblewrap_cleanup_failed"
	CodeSnapshotInvalid, CodeSnapshotLimit                    = "test_attestor.snapshot_stream_invalid", "test_attestor.snapshot_limit_exceeded"
	CodeResourceUnsafe, CodeResourceLimit                     = "test_attestor.resource_governor_unsafe", "test_attestor.resource_budget_exceeded"
	goToolRef, workspaceMount, toolchainMount, goCommand      = "tool:go", "/subject", "/toolchain", "/toolchain/bin/go"
	snapshotEntryTag, snapshotEndTag                     byte = 1, 255
	maxSubjectEntries                                         = 1_000_000
)

var snapshotStreamMagic = []byte("ORQ-SNAPSHOT-2\x00")

type SnapshotStreamSource interface {
	OpenSnapshotStream(context.Context, ports.SnapshotVerificationRequest) (io.ReadCloser, error)
	SnapshotIdentity() (string, string)
}
type Limits struct {
	Timeout, CleanupTimeout                                                                     time.Duration
	MaxOutputBytes, MaxSubjectBytes, MaxConcurrentRuns, MemoryMaxBytes, PIDsMax, CPUQuotaMicros int64
}
type Config struct {
	BubblewrapCommand, ToolchainRoot, CgroupRoot string
	SnapshotSource                               SnapshotStreamSource
	Now                                          func() time.Time
	Limits                                       Limits
}
type Error struct{ Code string }

func (err *Error) Error() string { return err.Code }
func (err *Error) CauseCode() string {
	if err == nil {
		return ""
	}
	return err.Code
}
func ErrorCode(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

type PolicyIdentity struct{ Ref, Digest string }

func SubjectEntryLimit(bytes int64) int64 {
	limit := bytes/1024 + 1
	if limit > maxSubjectEntries {
		return maxSubjectEntries
	}
	return limit
}

func configuredPolicyIdentity(config Config, trusted, resource, sourceRef, sourceDigest string, uid, gid int) PolicyIdentity {
	digest := sha256.New()
	for _, value := range []string{"orquesta.test-attestor.bubblewrap.policy.v4", PolicyRef, AttestorRef,
		"one-bwrap-per-test;fd-pinned-inputs;sealed-object-memfds;cgroup-v2;no-shell", trusted, resource,
		sourceRef, sourceDigest, strconv.Itoa(uid), strconv.Itoa(gid), config.Limits.Timeout.String(),
		config.Limits.CleanupTimeout.String(), strconv.FormatInt(config.Limits.MaxOutputBytes, 10),
		strconv.FormatInt(config.Limits.MaxSubjectBytes, 10), strconv.FormatInt(config.Limits.MaxConcurrentRuns, 10),
		strconv.FormatInt(config.Limits.MemoryMaxBytes, 10), strconv.FormatInt(config.Limits.PIDsMax, 10),
		strconv.FormatInt(config.Limits.CPUQuotaMicros, 10)} {
		writeDigestField(digest, value)
	}
	return PolicyIdentity{PolicyRef, hex.EncodeToString(digest.Sum(nil))}
}
func writeDigestField(digest hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}
func validIdentity(ref, digest string) bool {
	return ref != "" && len(ref) <= 256 && strings.TrimSpace(ref) == ref && !strings.ContainsRune(ref, 0) && len(digest) == sha256.Size*2 && strings.Trim(digest, "0123456789abcdef") == ""
}
func validConfig(config Config) bool {
	limits := config.Limits
	return config.SnapshotSource != nil && config.Now != nil && limits.Timeout > 0 && limits.CleanupTimeout > 0 && limits.CleanupTimeout < limits.Timeout && limits.MaxOutputBytes > 0 && limits.MaxSubjectBytes > 0 && limits.MaxConcurrentRuns > 0 && int64(int(limits.MaxConcurrentRuns)) == limits.MaxConcurrentRuns && limits.MemoryMaxBytes > 0 && limits.PIDsMax > 0 && limits.CPUQuotaMicros > 0
}

func validateRun(run ports.TestAttestationRun, policy PolicyIdentity) error {
	if err := ports.ValidateTestAttestationRequest(run.Request); err != nil {
		return err
	}
	if err := ports.ValidateSnapshotVerificationRequest(run.Snapshot); err != nil || run.Request.Subject != run.Snapshot.Subject || run.Request.SubjectDigest != run.Snapshot.SubjectDigest {
		return snapshotInvalid()
	}
	if run.Request.Subject.PolicyRef != policy.Ref || run.Request.Subject.PolicyDigest != policy.Digest {
		return &Error{Code: CodePolicyMismatch}
	}
	for _, spec := range run.Request.RequiredTests {
		if spec.ToolRef().String() != goToolRef {
			return &Error{Code: CodeToolUnsupported}
		}
	}
	return nil
}
