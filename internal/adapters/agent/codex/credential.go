package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (adapter *Adapter) launchWithCredentialLocked(
	ctx context.Context,
	callerContext context.Context,
	request ports.AgentLaunchRequest,
	requestHash string,
	session *resolvedSession,
) (ports.AgentLaunchReceipt, error) {
	var receipt ports.AgentLaunchReceipt
	var launchErr error
	_, useErr := adapter.config.CredentialStore.Use(ctx, adapter.credentialUseRequest(request), func(secret credentials.Secret) error {
		defer secret.Destroy()
		guard, err := credentials.NewLeakGuard(secret)
		if err != nil {
			launchErr = err
			return err
		}
		if err := adapter.preflightCredentialLaunch(secret, guard, request, session); err != nil {
			guard.Destroy()
			launchErr = err
			return err
		}
		environment := adapter.environmentWithSession(adapter.environmentWithCredential(secret), session)
		defer clearEnvironment(environment)
		record, runPath, recordCreated, err := adapter.ensureLaunchRecord(request, requestHash)
		if err != nil {
			guard.Destroy()
			launchErr = err
			return err
		}
		receipt, launchErr = adapter.resumeLaunchRecordLocked(
			ctx, callerContext, request, requestHash, record, runPath, recordCreated, environment, guard, session,
		)
		return launchErr
	})
	if launchErr != nil {
		return ports.AgentLaunchReceipt{}, launchErr
	}
	if useErr != nil {
		if errors.Is(useErr, context.Canceled) || errors.Is(useErr, context.DeadlineExceeded) {
			return ports.AgentLaunchReceipt{}, useErr
		}
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeCredentialUnavailable, Cause: useErr}
	}
	return receipt, nil
}

func (adapter *Adapter) preflightCredentialAuthority(ctx context.Context, request ports.AgentLaunchRequest, session *resolvedSession) error {
	var preflightErr error
	_, useErr := adapter.config.CredentialStore.Use(ctx, adapter.credentialUseRequest(request), func(secret credentials.Secret) error {
		defer secret.Destroy()
		guard, err := credentials.NewLeakGuard(secret)
		if err != nil {
			preflightErr = err
			return err
		}
		defer guard.Destroy()
		preflightErr = adapter.preflightCredentialLaunch(secret, guard, request, session)
		return preflightErr
	})
	if preflightErr != nil {
		return preflightErr
	}
	if useErr != nil {
		if errors.Is(useErr, context.Canceled) || errors.Is(useErr, context.DeadlineExceeded) {
			return useErr
		}
		return &Error{Code: CodeCredentialUnavailable, Cause: useErr}
	}
	return nil
}

func (adapter *Adapter) credentialUseRequest(request ports.AgentLaunchRequest) credentials.UseRequest {
	return credentials.UseRequest{ActorRef: request.ActorRef.String(), RequestRef: "request:codex-launch:" + request.ExecutionRef.String(),
		CredentialRef: adapter.config.CredentialRef, OwnerRef: credentials.OwnerRef(request.ActorRef.String()),
		ScopeRef: credentials.ScopeRef(request.ProjectRef.String()), PurposeRef: credentials.PurposeRef(ProviderRef)}
}

func (record launchRecord) recoveryRequest(receipt ports.AgentLaunchReceipt) ports.AgentLaunchRequest {
	actorRef, _ := goal.NewActorRef(record.ActorRef)
	projectRef, _ := goal.NewProjectRef(record.ProjectRef)
	sessionRef, _ := ports.NewExecutionSessionRef(record.ExecutionSessionRef)
	return ports.AgentLaunchRequest{SessionRef: sessionRef, ProjectRef: projectRef, ActorRef: actorRef,
		GoalRef: receipt.GoalRef, WorkItemRef: receipt.WorkItemRef, ExecutionRef: receipt.ExecutionRef,
		ExecutionAttempt: receipt.ExecutionAttempt, PlanGeneration: receipt.PlanGeneration,
		AppSpecGeneration: receipt.AppSpecGeneration, SpecHash: record.SpecHash}
}

func (adapter *Adapter) recoverExecutionGuards(ctx context.Context, record launchRecord, state *executionState) error {
	needsCredential := adapter.config.CredentialStore != nil
	if !needsCredential && record.ExecutionSessionRef == "" {
		return nil
	}
	if needsCredential && (record.ActorRef == "" || record.ProjectRef == "") {
		return &Error{Code: CodeCredentialUnavailable}
	}
	request := record.recoveryRequest(state.receipt)
	var session *resolvedSession
	var err error
	if record.ExecutionSessionRef != "" {
		session, err = adapter.recoverSession(ctx, request)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
	}
	defer session.destroy()
	var credentialGuard *credentials.LeakGuard
	if needsCredential {
		_, err = adapter.config.CredentialStore.Use(ctx, adapter.credentialUseRequest(request), func(secret credentials.Secret) error {
			credentialGuard, err = credentials.NewLeakGuard(secret)
			return err
		})
		if err != nil {
			credentialGuard.Destroy()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return &Error{Code: CodeCredentialUnavailable, Cause: err}
		}
	}
	state.credentialGuard = credentialGuard
	if session != nil {
		state.sessionGuard = session.guard
		session.guard = nil
	}
	return nil
}

func destroyExecutionGuards(state *executionState) {
	if state != nil {
		for _, guard := range []*credentials.LeakGuard{state.credentialGuard, state.sessionGuard} {
			guard.Destroy()
		}
		state.credentialGuard, state.sessionGuard = nil, nil
	}
}

func recoveryAuthorityFailure(err error) bool {
	code := ErrorCode(err)
	return code == CodeCredentialInvalid || code == CodeCredentialUnavailable || code == CodeSessionInvalid ||
		code == CodeSessionUnavailable || code == CodeSecretLeak
}

func (adapter *Adapter) quarantineLaunchRecoveryLocked(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	record launchRecord,
	terminalRequestHash, runPath string,
	cause error,
) error {
	state, err := adapter.recoveryState(record, runPath, request.ExecutionRef)
	if err != nil {
		return errors.Join(err, cause)
	}
	state.terminalRequestHash = terminalRequestHash
	return adapter.quarantineRecoveryFailureLocked(ctx, state, cause)
}

func (adapter *Adapter) recoveryState(record launchRecord, runPath string, executionRef goal.ExecutionRef) (*executionState, error) {
	receipt, err := adapter.observationReceipt(record, executionRef)
	if err != nil {
		return nil, err
	}
	return &executionState{requestHash: record.RequestHash, terminalRequestHash: record.RequestHash,
		receipt: receipt, maxOutput: record.MaxOutputBytes, runPath: runPath, status: ports.AgentPending}, nil
}

func (adapter *Adapter) quarantineRecoveryFailureLocked(ctx context.Context, state *executionState, cause error) error {
	if !recoveryAuthorityFailure(cause) {
		return cause
	}
	adopted, adoptErr := adapter.adoptPersistedProcessLocked(state)
	if adoptErr != nil {
		return errors.Join(adoptErr, cause)
	}
	if !adopted && state.terminal == nil {
		record, found, inspectErr := adapter.processRecordForState(state)
		var gone bool
		if inspectErr == nil && found {
			gone, inspectErr = adapter.inspectProcessTree(record)
		}
		if inspectErr == nil && found && !gone {
			inspectErr = &Error{Code: CodeProcessCleanupFailed}
		}
		if inspectErr != nil {
			return errors.Join(inspectErr, cause)
		}
	}
	if adopted {
		defer adapter.releaseProcessOwnershipLocked(state)
	}
	firstScrub := adapter.recoveryScrub(state.runPath)
	var stopErr error
	if adopted {
		record := *state.process
		for index, mode := range []ports.AgentStopMode{ports.AgentStopCooperative, ports.AgentStopForced} {
			stopErr = adapter.signalProcessTree(record, mode)
			if errors.Is(stopErr, os.ErrProcessDone) {
				stopErr = nil
				break
			}
			if stopErr != nil {
				if index == 0 && ErrorCode(stopErr) == CodeProcessSignalFailed {
					continue
				}
				break
			}
			timeout := adapter.config.ProcessPipeDrainDelay
			if index > 0 || timeout > adapter.config.Timeout {
				timeout = adapter.config.Timeout
			}
			waitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
			gone, waitErr := waitForExactProcess(waitCtx, record, nil)
			cancel()
			stopErr = waitErr
			if gone {
				stopErr = nil
				break
			}
			if !errors.Is(stopErr, context.DeadlineExceeded) {
				break
			}
		}
	}
	finalScrub := adapter.recoveryScrub(state.runPath)
	if stopErr != nil {
		return errors.Join(&Error{Code: CodeProcessCleanupFailed, Cause: stopErr}, finalScrub, firstScrub, cause)
	}
	if finalScrub != nil {
		return errors.Join(finalScrub, firstScrub, cause)
	}
	if state.terminal == nil {
		terminal := terminalRecord{SchemaVersion: stateSchemaVersion, RequestHash: state.terminalRequestHash,
			Status: ports.AgentFailed, ErrorCode: ErrorCode(cause), ObservedAt: adapter.terminalTime(state.receipt.AcceptedAt)}
		persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput)
		if err != nil {
			return errors.Join(err, firstScrub, cause)
		}
		state.status, state.terminal, state.terminalDurable = persisted.Status, &persisted, true
	}
	return errors.Join(firstScrub, cause)
}

func (adapter *Adapter) recoveryScrub(runPath string) error {
	if err := adapter.credentialOutputScrub(runPath); err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	return nil
}

func (adapter *Adapter) preflightCredentialLaunch(secret credentials.Secret, guard *credentials.LeakGuard, request ports.AgentLaunchRequest, session *resolvedSession) error {
	material := secret.Bytes()
	defer clearBytes(material)
	if bytes.IndexByte(material, 0) >= 0 {
		return &Error{Code: CodeCredentialInvalid}
	}
	prompt, err := adapter.renderAgentPrompt(request)
	if err != nil {
		return err
	}
	runPath := executionPath(request.ExecutionRef)
	surfaces := []credentials.LeakSurface{
		{Name: "prompt", Content: []byte(prompt)},
		{Name: "command", Content: []byte(adapter.command)},
		{Name: "arguments", Content: []byte(strings.Join(adapter.commandArgumentsWithSession(runPath, false, false, session), "\x00"))},
		{Name: "work_root", Content: []byte(adapter.rootPath)},
	}
	surfaces = append(surfaces, credentialLaunchRequestSurfaces(request)...)
	for _, entry := range adapter.environment {
		surfaces = append(surfaces, credentials.LeakSurface{Name: "public_environment", Content: []byte(entry)})
	}
	defer clearLeakSurfaces(surfaces)
	if err := guard.Scan(surfaces); err != nil {
		if credentials.HasErrorCode(err, credentials.ErrorSecretLeak) {
			return &Error{Code: CodeSecretLeak}
		}
		return &Error{Code: CodeCredentialUnavailable, Cause: err}
	}
	return nil
}

func credentialLaunchRequestSurfaces(request ports.AgentLaunchRequest) []credentials.LeakSurface {
	surfaces := make([]credentials.LeakSurface, 0, 24+len(request.PhaseInputRefs)+len(request.PhaseCriterionRefs)+
		len(request.SkillRefs)+len(request.ToolRefs)+len(request.CapabilityRefs)+len(request.WriteSet))
	appendValue := func(name, value string) {
		surfaces = append(surfaces, credentials.LeakSurface{Name: name, Content: []byte(value)})
	}
	appendValues := func(name string, values []string) {
		for index, value := range values {
			appendValue(name+"["+strconv.Itoa(index)+"]", value)
		}
	}
	appendValue("launch_request.execution_ref", request.ExecutionRef.String())
	appendValue("launch_request.goal_ref", request.GoalRef.String())
	appendValue("launch_request.work_item_ref", request.WorkItemRef.String())
	appendValue("launch_request.plan_generation", strconv.FormatUint(uint64(request.PlanGeneration), 10))
	appendValue("launch_request.app_spec_generation", strconv.FormatUint(uint64(request.AppSpecGeneration), 10))
	appendValue("launch_request.execution_attempt", strconv.FormatUint(request.ExecutionAttempt, 10))
	appendValue("launch_request.spec_hash", request.SpecHash)
	appendValue("launch_request.actor_ref", request.ActorRef.String())
	appendValue("launch_request.project_ref", request.ProjectRef.String())
	appendValue("launch_request.objective", request.Objective)
	appendValue("launch_request.phase_ref", request.PhaseRef)
	appendValue("launch_request.phase_key", request.PhaseKey)
	appendValue("launch_request.phase_template_ref", request.PhaseTemplateRef)
	appendValues("launch_request.phase_input_refs", request.PhaseInputRefs)
	appendValues("launch_request.phase_criterion_refs", request.PhaseCriterionRefs)
	appendValue("launch_request.role_key", request.RoleKey)
	appendValues("launch_request.skill_refs", request.SkillRefs)
	appendValues("launch_request.tool_refs", request.ToolRefs)
	appendValues("launch_request.capability_refs", request.CapabilityRefs)
	appendValues("launch_request.write_set", request.WriteSet)
	appendValue("launch_request.output_contract", request.OutputContract)
	appendValue("launch_request.artifact_media_type", request.ArtifactMediaType)
	appendValue("launch_request.idempotency_key", request.IdempotencyKey)
	appendValue("launch_request.max_output_bytes", strconv.FormatInt(request.MaxOutputBytes, 10))
	return surfaces
}

func (adapter *Adapter) environmentWithCredential(secret credentials.Secret) []string {
	material := secret.Bytes()
	defer clearBytes(material)
	environment := append([]string(nil), adapter.environment...)
	return append(environment, codexAPIKeyEnvironment+"="+string(material))
}

func (adapter *Adapter) gateCredentialTerminalLocked(state *executionState, terminal terminalRecord) (terminalRecord, error) {
	guards := []*credentials.LeakGuard{state.credentialGuard, state.sessionGuard}
	if guards[0] == nil && guards[1] == nil {
		return terminal, nil
	}
	defer destroyExecutionGuards(state)
	raw, readErr := adapter.readCredentialOutput(state.runPath, state.maxOutput)
	defer clearBytes(raw)
	var decoded modelResult
	_ = json.Unmarshal(raw, &decoded)
	defer func() { decoded.Artifact = "" }()
	artifactProjection := []byte(terminal.Artifact)
	decodedArtifactProjection := []byte(decoded.Artifact)
	defer clearBytes(artifactProjection)
	defer clearBytes(decodedArtifactProjection)
	surfaces := []credentials.LeakSurface{
		{Name: "diagnostic", Content: terminal.Diagnostic},
		{Name: "last_message", Content: raw},
		{Name: "artifact", Content: artifactProjection},
		{Name: "decoded_artifact", Content: decodedArtifactProjection},
	}
	var scanErr error
	for _, guard := range guards {
		if guard == nil {
			continue
		}
		if err := guard.Scan(surfaces); err != nil {
			scanErr = err
			break
		}
	}
	if readErr == nil && scanErr == nil {
		return terminal, nil
	}
	clearBytes(terminal.Diagnostic)
	terminal.Diagnostic = nil
	terminal.Artifact = ""
	redacted := terminalRecord{SchemaVersion: stateSchemaVersion, RequestHash: terminal.RequestHash,
		Status: ports.AgentFailed, ErrorCode: CodeSecretLeak, ObservedAt: terminal.ObservedAt}
	if scrubErr := adapter.credentialOutputScrub(state.runPath); scrubErr != nil {
		return redacted, &Error{Code: CodeStatePersistenceFailed, Cause: scrubErr}
	}
	if readErr != nil {
		return redacted, nil
	}
	if !credentials.HasErrorCode(scanErr, credentials.ErrorSecretLeak) {
		return redacted, &Error{Code: CodeCredentialUnavailable, Cause: scanErr}
	}
	return redacted, nil
}

func (adapter *Adapter) readCredentialOutput(runPath string, maximum int64) ([]byte, error) {
	filePath := path.Join(runPath, lastMessageFileName)
	info, err := adapter.root.Lstat(filePath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errors.New("codex.credential_output_unverifiable")
	}
	payload, err := adapter.root.ReadFile(filePath)
	if err != nil || int64(len(payload)) > maximum {
		clearBytes(payload)
		return nil, errors.New("codex.credential_output_unverifiable")
	}
	return payload, nil
}

func (adapter *Adapter) scrubCredentialOutput(runPath string) error {
	filePath := path.Join(runPath, lastMessageFileName)
	writeErr := adapter.writePrivateRuntimeFile(filePath, nil)
	artifactErr := adapter.removeCompletionArtifacts(runPath)
	if writeErr == nil {
		return artifactErr
	}
	removeErr := adapter.root.Remove(filePath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return &Error{Code: CodeStatePersistenceFailed, Cause: errors.Join(writeErr, removeErr, artifactErr)}
	}
	return errors.Join(adapter.syncDirectoryCausally(runPath), artifactErr)
}

func clearBytes(material []byte) {
	for index := range material {
		material[index] = 0
	}
}

func clearLeakSurfaces(surfaces []credentials.LeakSurface) {
	for index := range surfaces {
		clearBytes(surfaces[index].Content)
	}
}
