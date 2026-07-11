package orquestaruntimerequiredtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type LocalTrustedGoalRequiredTestIdentityPolicyV0 struct {
	TrustPolicyRef           string
	PolicyEvidenceRef        string
	ImplementerAgentRef      string
	ImplementerCredentialRef string
	ImplementerPrincipalRef  string
	AttestorAgentRef         string
	AttestorCredentialRef    string
	AttestorPrincipalRef     string
}

type LocalGoalRequiredTestAttestationConfigV0 struct {
	ProjectWorkDir           string
	RuntimeRoot              string
	GitCommandPath           string
	AllowedCommands          map[string]string
	DependencySnapshotPath   string
	DependencySnapshotSHA256 string
	PreflightCommands        []string
	MaxRuntime               time.Duration
	MaxOutputBytes           int64
	MaxArtifacts             int
	Identity                 LocalTrustedGoalRequiredTestIdentityPolicyV0
}

// LocalGoalRequiredTestAttestationAdapterV0 is an external adapter. It derives
// checkout metadata and write-set hashes from the configured worktree and runs
// only frozen allowlisted commands without a shell or inherited environment.
type LocalGoalRequiredTestAttestationAdapterV0 struct {
	config LocalGoalRequiredTestAttestationConfigV0
}

func NewLocalGoalRequiredTestAttestationAdapterV0(
	config LocalGoalRequiredTestAttestationConfigV0,
) (*LocalGoalRequiredTestAttestationAdapterV0, error) {
	normalized, err := normalizeLocalGoalRequiredTestAttestationConfigV0(config)
	if err != nil {
		return nil, err
	}
	return &LocalGoalRequiredTestAttestationAdapterV0{config: normalized}, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) BindGoalRequiredTestSpecV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	if adapter == nil {
		return orquestagoal.GoalWorkSpecV0{}, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestagoal.GoalWorkSpecV0{}, err
		}
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		return spec, nil
	}
	identity := adapter.config.Identity
	for field, pair := range map[string][2]string{
		"implementer_agent_ref":              {spec.ImplementerAgentRef, identity.ImplementerAgentRef},
		"implementer_credential_ref":         {spec.ImplementerCredentialRef, identity.ImplementerCredentialRef},
		"required_attestor_trust_policy_ref": {spec.ClosurePolicy.RequiredAttestorTrustPolicyRef, identity.TrustPolicyRef},
	} {
		if strings.TrimSpace(pair[0]) != "" && strings.TrimSpace(pair[0]) != strings.TrimSpace(pair[1]) {
			return orquestagoal.GoalWorkSpecV0{}, fmt.Errorf("goal_required_test_trusted_binding_conflict: %s", field)
		}
	}
	spec.ImplementerAgentRef = identity.ImplementerAgentRef
	spec.ImplementerCredentialRef = identity.ImplementerCredentialRef
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = identity.TrustPolicyRef
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationBindingV0(spec); len(issues) > 0 {
		return orquestagoal.GoalWorkSpecV0{}, fmt.Errorf("goal_required_test_trusted_binding_invalid")
	}
	return spec, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) CaptureGoalRequiredTestFinalSnapshotV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestFinalSnapshotRequestV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	if adapter == nil {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if issues := orquestagoal.ValidateGoalRequiredTestFinalSnapshotRequestV0(request); len(issues) > 0 {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("goal_required_test_final_snapshot_request_invalid")
	}
	checkoutRef, revisionRef, err := adapter.observeCheckoutV0(ctx)
	if err != nil {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, err
	}
	hashes, err := adapter.observeWriteSetHashesV0(request.WriteSet)
	if err != nil {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, err
	}
	snapshot := orquestagoal.FreezeGoalRequiredTestFinalSnapshotV0(orquestagoal.GoalRequiredTestFinalSnapshotV0{
		RunRef: strings.TrimSpace(request.RunRef), GoalRef: strings.TrimSpace(request.GoalRef),
		CheckoutRef: checkoutRef, RevisionRef: revisionRef,
		WriteSetSHA256: strings.ToLower(strings.TrimSpace(request.WriteSetSHA256)),
		Hashes:         hashes, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	evidenceRef, err := adapter.writeSnapshotEvidenceV0(snapshot)
	if err != nil {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, err
	}
	snapshot.EvidenceRefs = []string{evidenceRef}
	if issues := orquestagoal.ValidateGoalRequiredTestFinalSnapshotV0(snapshot); len(issues) > 0 {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("goal_required_test_final_snapshot_invalid")
	}
	return snapshot, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) AttestGoalRequiredTestsV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	if adapter == nil {
		return nil, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	request = orquestagoal.NormalizeGoalRequiredTestAttestationRequestV0(request)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationRequestV0(request); len(issues) > 0 || len(request.RequiredTests) != 1 {
		return nil, fmt.Errorf("goal_required_test_attestation_request_invalid")
	}
	test := request.RequiredTests[0]
	writeSet := make([]orquestagoal.GoalWriteScopeV0, 0, len(request.FinalSnapshot.Hashes))
	for _, hash := range request.FinalSnapshot.Hashes {
		writeSet = append(writeSet, orquestagoal.GoalWriteScopeV0{Path: hash.Ref})
	}
	startedAt := time.Now().UTC()
	before, err := adapter.CaptureGoalRequiredTestFinalSnapshotV0(ctx, orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef, WriteSet: writeSet,
		WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256,
	})
	if err != nil {
		return nil, err
	}
	attestation := adapter.attestationBaseV0(request, test, startedAt)
	attestation.HashesBefore = append([]orquestagoal.GoalAttestedHashV0(nil), before.Hashes...)
	attestation.EvidenceRefs = append(attestation.EvidenceRefs, before.EvidenceRefs...)
	if !orquestagoal.GoalRequiredTestFinalSnapshotIdentityEqualV0(before, request.FinalSnapshot) {
		attestation.Status = orquestagoal.GoalRequiredTestAttestationStatusFailedV0
		attestation.ExitCode = 1
		attestation.HashesAfter = append([]orquestagoal.GoalAttestedHashV0(nil), before.Hashes...)
		attestation.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		attestation.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(attestation)
		return []orquestagoal.GoalRequiredTestAttestationV0{orquestagoal.NormalizeGoalRequiredTestAttestationV0(attestation)}, nil
	}

	attestation.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(attestation)
	preflight, err := adapter.runPreflightV0(ctx, attestation.AttestationRef, request.GoalRef)
	if err != nil {
		return nil, err
	}
	attestation.EvidenceRefs = append(attestation.EvidenceRefs, preflight.EvidenceRefs...)
	if preflight.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
		after, err := adapter.CaptureGoalRequiredTestFinalSnapshotV0(ctx, orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
			RunRef: request.RunRef, GoalRef: request.GoalRef, WriteSet: writeSet,
			WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256,
		})
		if err != nil {
			return nil, err
		}
		attestation.HashesAfter = append([]orquestagoal.GoalAttestedHashV0(nil), after.Hashes...)
		attestation.EvidenceRefs = append(attestation.EvidenceRefs, after.EvidenceRefs...)
		attestation.Status = orquestagoal.GoalRequiredTestAttestationStatusFailedV0
		attestation.FailureCode = orquestagoal.ErrGoalRequiredTestAttestorInfrastructureFailedV0
		attestation.ExitCode = 1
		attestation.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		attestation.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(attestation)
		return []orquestagoal.GoalRequiredTestAttestationV0{orquestagoal.NormalizeGoalRequiredTestAttestationV0(attestation)}, nil
	}
	execution, err := adapter.runFrozenTestV0(ctx, attestation.AttestationRef, request, test)
	if err != nil {
		return nil, err
	}
	after, err := adapter.CaptureGoalRequiredTestFinalSnapshotV0(ctx, orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef, WriteSet: writeSet,
		WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256,
	})
	if err != nil {
		return nil, err
	}
	attestation.HashesAfter = append([]orquestagoal.GoalAttestedHashV0(nil), after.Hashes...)
	attestation.EvidenceRefs = append(attestation.EvidenceRefs, execution.EvidenceRefs...)
	attestation.EvidenceRefs = append(attestation.EvidenceRefs, after.EvidenceRefs...)
	attestation.Status = orquestagoal.GoalRequiredTestAttestationStatusPassedV0
	attestation.ExitCode = 0
	if execution.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 || !reflect.DeepEqual(attestation.HashesBefore, attestation.HashesAfter) {
		attestation.Status = orquestagoal.GoalRequiredTestAttestationStatusFailedV0
		attestation.ExitCode = 1
	}
	attestation.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	attestation.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(attestation)
	return []orquestagoal.GoalRequiredTestAttestationV0{orquestagoal.NormalizeGoalRequiredTestAttestationV0(attestation)}, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) VerifyGoalRequiredTestIdentityV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestIdentityVerificationRequestV0,
) (orquestagoal.GoalRequiredTestIdentityVerificationV0, error) {
	if adapter == nil {
		return orquestagoal.GoalRequiredTestIdentityVerificationV0{}, fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return orquestagoal.GoalRequiredTestIdentityVerificationV0{}, err
		}
	}
	identity := adapter.config.Identity
	verified := strings.TrimSpace(request.ImplementerAgentRef) == identity.ImplementerAgentRef &&
		strings.TrimSpace(request.ImplementerCredentialRef) == identity.ImplementerCredentialRef &&
		strings.TrimSpace(request.AttestorAgentRef) == identity.AttestorAgentRef &&
		strings.TrimSpace(request.AttestorCredentialRef) == identity.AttestorCredentialRef &&
		strings.TrimSpace(request.RequiredTrustPolicyRef) == identity.TrustPolicyRef
	return orquestagoal.GoalRequiredTestIdentityVerificationV0{
		AttestationRef: strings.TrimSpace(request.AttestationRef), TestRef: strings.TrimSpace(request.TestRef),
		Verified: verified, Independent: verified && identity.ImplementerPrincipalRef != identity.AttestorPrincipalRef,
		ImplementerPrincipalRef: identity.ImplementerPrincipalRef, AttestorPrincipalRef: identity.AttestorPrincipalRef,
		AttestorCredentialRef: identity.AttestorCredentialRef, TrustPolicyRef: identity.TrustPolicyRef,
		EvidenceRefs: []string{identity.PolicyEvidenceRef},
	}, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) attestationBaseV0(
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
	test orquestagoal.GoalRequiredTestV0,
	startedAt time.Time,
) orquestagoal.GoalRequiredTestAttestationV0 {
	identity := adapter.config.Identity
	isolatedRef := "goal-required-test-isolated-env-ref-" + localGoalAttestationHashV0(strings.Join([]string{
		request.RunRef, request.GoalRef, request.FinalSnapshot.RevisionRef, test.TestRef,
	}, "\x00"))
	return orquestagoal.GoalRequiredTestAttestationV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef, FinalSnapshotRef: request.FinalSnapshot.SnapshotRef,
		CheckoutRef: request.FinalSnapshot.CheckoutRef, RevisionRef: request.FinalSnapshot.RevisionRef,
		WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256, TestRef: test.TestRef,
		CommandRef: test.CommandRef, CommandSHA256: test.CommandSHA256, DefinitionSHA256: test.DefinitionSHA256,
		ImplementerAgentRef: request.ImplementerAgentRef,
		AttestorAgentRef:    identity.AttestorAgentRef, AttestorCredentialRef: identity.AttestorCredentialRef,
		StartedAt: startedAt.Format(time.RFC3339Nano), IsolatedEnvironmentRef: isolatedRef,
		EvidenceRefs: []string{request.FinalSnapshot.SnapshotRef},
	}
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) runFrozenTestV0(
	ctx context.Context,
	attestationRef string,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
	test orquestagoal.GoalRequiredTestV0,
) (result orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, resultErr error) {
	runsRoot := filepath.Join(adapter.config.RuntimeRoot, "runs")
	if err := os.MkdirAll(runsRoot, 0o700); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	runDir, err := os.MkdirTemp(runsRoot, localGoalAttestationHashV0(attestationRef)+"-")
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(runDir); resultErr == nil && cleanupErr != nil {
			result = orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}
			resultErr = fmt.Errorf("goal_required_test_execution_cleanup_failed: %w", cleanupErr)
		}
	}()
	outputDir := filepath.Join(adapter.config.RuntimeRoot, "evidence", "commands")
	for _, dir := range []string{runDir, outputDir, filepath.Join(runDir, "tmp"), filepath.Join(runDir, "go-cache"), filepath.Join(runDir, "go-path")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
		}
	}
	moduleCache, err := adapter.materializeDependencySnapshotV0(runDir)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	executionContext, cancel := context.WithTimeout(ctx, adapter.config.MaxRuntime)
	defer cancel()
	executor := LocalCommandExecutorV0{
		ProjectWorkDir: adapter.config.ProjectWorkDir, OutputDir: outputDir,
		AllowedCommands: adapter.config.AllowedCommands,
		Env:             adapter.hermeticEnvironmentV0(runDir, moduleCache),
		MaxOutputBytes:  adapter.config.MaxOutputBytes, MaxArtifacts: adapter.config.MaxArtifacts,
	}
	return executor.RunRequiredTestCommandV0(executionContext, orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
		RunRef: request.RunRef, TaskRef: test.TestRef, TestCommand: test.Command,
		CorrelationID: attestationRef, EvidenceRefs: []string{request.FinalSnapshot.SnapshotRef},
	})
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) observeCheckoutV0(ctx context.Context) (string, string, error) {
	rootOutput, err := adapter.runGitV0(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	observedRoot, err := filepath.EvalSymlinks(strings.TrimSpace(rootOutput))
	if err != nil || observedRoot != adapter.config.ProjectWorkDir {
		return "", "", fmt.Errorf("goal_required_test_checkout_mismatch")
	}
	revision, err := adapter.runGitV0(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	revision = strings.TrimSpace(revision)
	if len(revision) < 40 {
		return "", "", fmt.Errorf("goal_required_test_revision_invalid")
	}
	return "local-checkout-ref-" + localGoalAttestationHashV0(observedRoot), revision, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) runGitV0(ctx context.Context, args ...string) (string, error) {
	commandArgs := append([]string{"-C", adapter.config.ProjectWorkDir}, args...)
	cmd := exec.CommandContext(ctx, adapter.config.GitCommandPath, commandArgs...)
	cmd.Env = []string{"LC_ALL=C", "LANG=C"}
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("goal_required_test_git_observation_failed: %w", err)
	}
	return string(output), nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) observeWriteSetHashesV0(
	writeSet []orquestagoal.GoalWriteScopeV0,
) ([]orquestagoal.GoalAttestedHashV0, error) {
	hashes := make([]orquestagoal.GoalAttestedHashV0, 0, len(writeSet))
	for _, scope := range writeSet {
		ref := filepath.ToSlash(strings.TrimSpace(scope.Path))
		hash, err := hashLocalGoalWriteScopeV0(adapter.config.ProjectWorkDir, ref)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, orquestagoal.GoalAttestedHashV0{Ref: ref, SHA256: hash})
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i].Ref < hashes[j].Ref })
	return hashes, nil
}

func hashLocalGoalWriteScopeV0(projectWorkDir string, ref string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(ref))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("goal_required_test_write_scope_invalid")
	}
	path := filepath.Join(projectWorkDir, clean)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return localGoalAttestationHashV0("missing\x00" + filepath.ToSlash(clean)), nil
	}
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if err := hashLocalGoalPathV0(hash, path, info, filepath.ToSlash(clean)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func hashLocalGoalPathV0(hash io.Writer, path string, info os.FileInfo, ref string) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("goal_required_test_write_scope_symlink_prohibited: %s", ref)
	}
	if info.IsDir() {
		_, _ = io.WriteString(hash, "dir\x00"+ref+"\x00")
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			childInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if err := hashLocalGoalPathV0(hash, filepath.Join(path, entry.Name()), childInfo, filepath.ToSlash(filepath.Join(ref, entry.Name()))); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("goal_required_test_write_scope_type_prohibited: %s", ref)
	}
	_, _ = io.WriteString(hash, "file\x00"+ref+"\x00")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(hash, file)
	return err
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) writeSnapshotEvidenceV0(
	snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0,
) (string, error) {
	evidenceDir := filepath.Join(adapter.config.RuntimeRoot, "evidence", "snapshots")
	if err := os.MkdirAll(evidenceDir, 0o700); err != nil {
		return "", err
	}
	refHash := localGoalAttestationHashV0(snapshot.SnapshotRef + "\x00" + snapshot.ObservedAt)
	ref := "goal-required-test-snapshot-evidence-ref-" + refHash
	payload := struct {
		SchemaVersion string                                       `json:"schema_version"`
		EvidenceRef   string                                       `json:"evidence_ref"`
		Snapshot      orquestagoal.GoalRequiredTestFinalSnapshotV0 `json:"snapshot"`
	}{"orquesta.runtime_required_test.goal_snapshot_evidence.v0", ref, snapshot}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, refHash+".json"), append(encoded, '\n'), 0o600); err != nil {
		return "", err
	}
	return ref, nil
}

func normalizeLocalGoalRequiredTestAttestationConfigV0(
	config LocalGoalRequiredTestAttestationConfigV0,
) (LocalGoalRequiredTestAttestationConfigV0, error) {
	project, err := filepath.Abs(strings.TrimSpace(config.ProjectWorkDir))
	if err != nil {
		return config, fmt.Errorf("goal_required_test_project_work_dir_invalid")
	}
	project, err = filepath.EvalSymlinks(project)
	if err != nil {
		return config, fmt.Errorf("goal_required_test_project_work_dir_invalid: %w", err)
	}
	runtimeRoot, err := filepath.Abs(strings.TrimSpace(config.RuntimeRoot))
	if err != nil || runtimeRoot == project || strings.HasPrefix(runtimeRoot+string(filepath.Separator), project+string(filepath.Separator)) {
		return config, fmt.Errorf("goal_required_test_runtime_root_not_isolated")
	}
	if err := os.MkdirAll(runtimeRoot, 0o700); err != nil {
		return config, err
	}
	gitPath := strings.TrimSpace(config.GitCommandPath)
	if !filepath.IsAbs(gitPath) || commandIsShellV0(gitPath) {
		return config, fmt.Errorf("goal_required_test_git_command_invalid")
	}
	if info, err := os.Stat(gitPath); err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return config, fmt.Errorf("goal_required_test_git_command_invalid")
	}
	allowed := map[string]string{}
	for name, path := range config.AllowedCommands {
		name = strings.TrimSpace(name)
		path = strings.TrimSpace(path)
		if name == "" || strings.ContainsAny(name, `/\\`) || commandIsShellV0(name) || !filepath.IsAbs(path) || commandIsShellV0(path) {
			return config, fmt.Errorf("goal_required_test_allowed_command_invalid: %s", name)
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			return config, fmt.Errorf("goal_required_test_allowed_command_invalid: %s", name)
		}
		allowed[name] = path
	}
	if len(allowed) == 0 || config.MaxRuntime <= 0 || config.MaxOutputBytes <= 0 || config.MaxArtifacts <= 0 {
		return config, fmt.Errorf("goal_required_test_attestation_config_incomplete")
	}
	preflightCommands := make([]string, 0, len(config.PreflightCommands))
	for _, command := range config.PreflightCommands {
		command = strings.TrimSpace(command)
		tokens, err := splitCommandV0(command)
		if err != nil || len(tokens) == 0 || commandIsShellV0(tokens[0]) {
			return config, fmt.Errorf("goal_required_test_preflight_command_invalid")
		}
		if _, ok := allowed[tokens[0]]; !ok {
			return config, fmt.Errorf("goal_required_test_preflight_command_not_allowed")
		}
		preflightCommands = append(preflightCommands, command)
	}
	if len(preflightCommands) == 0 {
		return config, fmt.Errorf("goal_required_test_preflight_required")
	}
	snapshotPath, snapshotHash, err := normalizeGoalRequiredTestDependencySnapshotV0(config.DependencySnapshotPath)
	if err != nil {
		return config, err
	}
	if goalRequiredTestGoAllowedV0(allowed) && snapshotPath == "" {
		return config, fmt.Errorf("goal_required_test_go_attestation_config_incomplete")
	}
	identity := config.Identity
	identity.TrustPolicyRef = strings.TrimSpace(identity.TrustPolicyRef)
	identity.PolicyEvidenceRef = strings.TrimSpace(identity.PolicyEvidenceRef)
	identity.ImplementerAgentRef = strings.TrimSpace(identity.ImplementerAgentRef)
	identity.ImplementerCredentialRef = strings.TrimSpace(identity.ImplementerCredentialRef)
	identity.ImplementerPrincipalRef = strings.TrimSpace(identity.ImplementerPrincipalRef)
	identity.AttestorAgentRef = strings.TrimSpace(identity.AttestorAgentRef)
	identity.AttestorCredentialRef = strings.TrimSpace(identity.AttestorCredentialRef)
	identity.AttestorPrincipalRef = strings.TrimSpace(identity.AttestorPrincipalRef)
	for _, value := range []string{
		identity.TrustPolicyRef, identity.PolicyEvidenceRef,
		identity.ImplementerAgentRef, identity.ImplementerCredentialRef, identity.ImplementerPrincipalRef,
		identity.AttestorAgentRef, identity.AttestorCredentialRef, identity.AttestorPrincipalRef,
	} {
		if value == "" {
			return config, fmt.Errorf("goal_required_test_identity_policy_incomplete")
		}
	}
	if identity.ImplementerPrincipalRef == identity.AttestorPrincipalRef ||
		identity.ImplementerCredentialRef == identity.AttestorCredentialRef || identity.ImplementerAgentRef == identity.AttestorAgentRef {
		return config, fmt.Errorf("goal_required_test_identity_policy_not_independent")
	}
	config.ProjectWorkDir = project
	config.RuntimeRoot = runtimeRoot
	config.GitCommandPath = gitPath
	config.AllowedCommands = allowed
	config.DependencySnapshotPath = snapshotPath
	config.DependencySnapshotSHA256 = snapshotHash
	config.PreflightCommands = preflightCommands
	config.Identity = identity
	return config, nil
}

func localGoalAttestationHashV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
