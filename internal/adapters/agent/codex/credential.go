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
	"orquesta/internal/ports"
)

func (adapter *Adapter) launchWithCredentialLocked(
	ctx context.Context,
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
			ctx, request, requestHash, record, runPath, recordCreated, environment, guard, session,
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

func (adapter *Adapter) credentialUseRequest(request ports.AgentLaunchRequest) credentials.UseRequest {
	return credentials.UseRequest{
		ActorRef: request.ActorRef.String(), RequestRef: "request:codex-launch:" + request.ExecutionRef.String(),
		CredentialRef: adapter.config.CredentialRef, OwnerRef: credentials.OwnerRef(request.ActorRef.String()),
		ScopeRef: credentials.ScopeRef(request.ProjectRef.String()), PurposeRef: credentials.PurposeRef(ProviderRef),
	}
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
	defer func() {
		for index := range surfaces {
			clearBytes(surfaces[index].Content)
		}
	}()
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
	defer func() {
		for _, guard := range guards {
			guard.Destroy()
		}
		state.credentialGuard = nil
		state.sessionGuard = nil
	}()
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
	redacted := terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: terminal.RequestHash,
		Status: ports.AgentFailed, ErrorCode: CodeSecretLeak, ObservedAt: terminal.ObservedAt,
	}
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
	if writeErr == nil {
		return nil
	}
	removeErr := adapter.root.Remove(filePath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return &Error{Code: CodeStatePersistenceFailed, Cause: errors.Join(writeErr, removeErr)}
	}
	if err := adapter.syncDirectoryCausally(runPath); err != nil {
		return err
	}
	return nil
}

func clearBytes(material []byte) {
	for index := range material {
		material[index] = 0
	}
}
