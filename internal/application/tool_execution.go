package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/tooling"
)

const (
	maxToolRawInputBytes       = 64 << 10
	ToolErrorUnavailable       = "tool.unavailable"
	ToolErrorRequestInvalid    = "tool.request_invalid"
	ToolErrorNotFound          = "tool.not_found"
	ToolErrorPolicyUnsupported = "tool.policy_unsupported"
	ToolErrorInputInvalid      = "tool.input_invalid"
	ToolErrorConnectorFailed   = "tool.connector_failed"
	ToolErrorReceiptInvalid    = "tool.receipt_invalid"
	ToolErrorOutputInvalid     = "tool.output_invalid"
	ToolErrorOutputTooLarge    = "tool.output_too_large"
	ToolErrorArtifactFailed    = "tool.artifact_failed"
)

type ToolInvocationError struct {
	Code string
}

func (failure *ToolInvocationError) Error() string { return failure.Code }
func ToolInvocationErrorCode(err error) string {
	var failure *ToolInvocationError
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

type InvokeToolRequest struct {
	Access                      Access
	RequestRef, ToolID, Version string
	Curation                    ToolCurationRequest
	Input                       json.RawMessage
}
type ToolCurationRequest struct {
	CatalogDigest string
	CapabilityRef goal.CapabilityRef
	Execution     ToolExecutionBinding
}
type ToolInvocationResult struct {
	ToolID, Version, RequestRef, SpecDigest string
	Catalog                                 ToolCatalogBinding
	CapabilityRef                           goal.CapabilityRef
	Execution                               ToolExecutionBinding
	AuthorizationReceiptRefs                []string
	ObservationReceiptRef                   string
	ObservedAt                              time.Time
	Usage                                   governance.ResourceUsage
	InlineOutput                            json.RawMessage
	Artifact                                ports.StoredArtifact
	ArtifactStored                          bool
}

func (orchestrator *Orchestrator) InvokeTool(ctx context.Context, request InvokeToolRequest) (ToolInvocationResult, error) {
	if orchestrator == nil || orchestrator.toolRegistry == nil || orchestrator.curatedToolCatalog == nil ||
		orchestrator.toolExecutor == nil ||
		orchestrator.access == nil || orchestrator.artifacts == nil || orchestrator.clock == nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorUnavailable, nil)
	}
	if ctx == nil || !validToolInvocationRef(request.RequestRef) {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, nil)
	}
	if len(request.Input) > maxToolRawInputBytes {
		return ToolInvocationResult{}, toolFailure(ToolErrorInputInvalid, nil)
	}
	principal, projectRef, err := request.Access.values()
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, err)
	}
	if principal.Kind != identity.PrincipalKindHuman {
		return ToolInvocationResult{}, errForbidden
	}
	catalogRegistration := orchestrator.curatedToolCatalog.Registration()
	catalog := ToolCatalogBinding{
		ID: catalogRegistration.Spec.ID, Version: catalogRegistration.Spec.Version,
		Digest: catalogRegistration.Digest, ReviewRef: catalogRegistration.Spec.ReviewRef, ReviewDigest: catalogRegistration.Spec.ReviewDigest,
	}
	if request.Curation.CatalogDigest != catalog.Digest {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, nil)
	}
	if request.Curation.CapabilityRef.String() == "" ||
		!validToolExecutionBinding(request.Curation.Execution, request.ToolID) {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, nil)
	}
	bindingDigest := toolExecutionBindingDigest(request.Curation.Execution)
	requestedAt := orchestrator.clock.Now().UTC()
	selectionDigest := toolInvocationDigest(
		"catalog-selection", catalog.ID, catalog.Version, catalog.Digest, catalog.ReviewRef, catalog.ReviewDigest,
		request.Curation.CapabilityRef.String(), bindingDigest, request.ToolID, request.Version, request.RequestRef,
		principal.Ref.String(), principal.ActorRef.String(), projectRef.String(), string(request.Input))
	selectionReceipt, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, request.Access, identity.PermissionGoalsGet, "tool-catalog-selection:sha256:"+selectionDigest,
		requestedAt, toolAuthorizationRequestRef(request.RequestRef, "catalog.select"))
	if err != nil {
		return ToolInvocationResult{}, err
	}
	authorizationRole, membershipRevision := selectionReceipt.Decision().Role(),
		selectionReceipt.Decision().MembershipRevision()
	if !orchestrator.toolAuthorizationCurrent(ctx, principal.Ref, projectRef, authorizationRole, membershipRevision) {
		return ToolInvocationResult{}, errForbidden
	}
	record, err := orchestrator.state.GetGoal(ctx, request.Curation.Execution.GoalRef)
	if err != nil || !toolExecutionIsCurrent(record, projectRef, request.Curation.CapabilityRef, request.Curation.Execution) {
		return ToolInvocationResult{}, toolFailure(ToolErrorNotFound, nil)
	}
	scope, err := tooling.NewSkillScopeContext("", projectRef, authorizationRole, request.Curation.Execution.GoalRef)
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, err)
	}
	curatedContext, err := tooling.NewCuratedContext(request.Curation.CapabilityRef, scope)
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorRequestInvalid, err)
	}
	registration, found, err := orchestrator.curatedToolCatalog.ResolveTool(curatedContext, request.ToolID, request.Version)
	if err != nil || !found {
		return ToolInvocationResult{}, toolFailure(ToolErrorNotFound, nil)
	}
	registered, registeredFound := orchestrator.toolRegistry.Lookup(request.ToolID, request.Version)
	if !registeredFound || registered.Digest != registration.Digest {
		return ToolInvocationResult{}, toolFailure(ToolErrorNotFound, nil)
	}
	if registration.Spec.Idempotency != tooling.IdempotencyReadReexecute ||
		registration.Spec.Receipt != tooling.ReceiptObservation || len(registration.Spec.Permissions) == 0 {
		return ToolInvocationResult{}, toolFailure(ToolErrorPolicyUnsupported, nil)
	}
	input, err := orchestrator.toolRegistry.ValidateInput(request.ToolID, request.Version, request.Input)
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorInputInvalid, err)
	}
	invocationDigest := toolInvocationDigest(
		registration.Digest, catalog.ID, catalog.Version, catalog.Digest, catalog.ReviewRef, catalog.ReviewDigest,
		request.Curation.CapabilityRef.String(), bindingDigest, request.RequestRef,
		principal.Ref.String(), principal.ActorRef.String(), projectRef.String(), string(input))
	resourceRef := "tool:" + registration.Spec.ID + "@" + registration.Spec.Version + ":sha256:" + invocationDigest
	authorizationRefs := []string{selectionReceipt.Ref()}
	causalFloor := selectionReceipt.RecordedAt()
	for _, permission := range registration.Spec.Permissions {
		authorizationRef := toolAuthorizationRequestRef(request.RequestRef, "tool.permission:"+string(permission))
		receipt, authorizeErr := orchestrator.authorizeIdempotentWithRequestRef(
			ctx, request.Access, permission, resourceRef, requestedAt, authorizationRef)
		if authorizeErr != nil {
			return ToolInvocationResult{}, authorizeErr
		}
		authorizationRefs = append(authorizationRefs, receipt.Ref())
		decision := receipt.Decision()
		if decision.Role() != authorizationRole || decision.MembershipRevision() != membershipRevision {
			return ToolInvocationResult{}, errForbidden
		}
		if receipt.RecordedAt().After(causalFloor) {
			causalFloor = receipt.RecordedAt()
		}
	}
	if !orchestrator.toolAuthorizationCurrent(ctx, principal.Ref, projectRef, authorizationRole, membershipRevision) {
		return ToolInvocationResult{}, errForbidden
	}
	record, err = orchestrator.state.GetGoal(ctx, request.Curation.Execution.GoalRef)
	if err != nil || !toolExecutionIsCurrent(record, projectRef, request.Curation.CapabilityRef, request.Curation.Execution) {
		return ToolInvocationResult{}, toolFailure(ToolErrorNotFound, nil)
	}
	idempotencyKey := "tool-observation:sha256:" + invocationDigest
	connectorRequest := ToolExecutionRequest{
		ToolID: registration.Spec.ID, Version: registration.Spec.Version,
		RequestRef: request.RequestRef, IdempotencyKey: idempotencyKey, SpecDigest: registration.Digest,
		Catalog: catalog, CapabilityRef: request.Curation.CapabilityRef, Execution: request.Curation.Execution,
		PrincipalRef: principal.Ref, ActorRef: principal.ActorRef, ProjectRef: projectRef,
		AuthorizationReceiptRefs: append([]string(nil), authorizationRefs...),
		Input:                    append([]byte(nil), input...),
		CostMaximum:              registration.Spec.Cost.Maximum, MaxOutputBytes: registration.Spec.Output.MaxBytes,
	}
	observation, err := orchestrator.toolExecutor.InvokeTool(ctx, cloneToolExecutionRequest(connectorRequest))
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorConnectorFailed, err)
	}
	observedCeiling := orchestrator.clock.Now().UTC()
	if !validToolObservationEnvelope(connectorRequest, observation, causalFloor, observedCeiling) {
		return ToolInvocationResult{}, toolFailure(ToolErrorReceiptInvalid, nil)
	}
	if !validToolResourceUsage(observation.Usage) {
		return ToolInvocationResult{}, toolFailure(ToolErrorReceiptInvalid, nil)
	}
	fits, usageErr := governance.Fits(registration.Spec.Cost.Maximum, observation.Usage.Resources)
	if usageErr != nil || !fits {
		return ToolInvocationResult{}, toolFailure(ToolErrorReceiptInvalid, usageErr)
	}
	if int64(len(observation.Output)) > registration.Spec.Output.MaxBytes {
		return ToolInvocationResult{}, toolFailure(ToolErrorOutputTooLarge, nil)
	}
	output, err := orchestrator.toolRegistry.ValidateOutput(registration.Spec.ID, registration.Spec.Version, observation.Output)
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorOutputInvalid, err)
	}
	if !slices.Equal(output, observation.Output) {
		return ToolInvocationResult{}, toolFailure(ToolErrorOutputInvalid, nil)
	}
	wantReceipt, err := ToolObservationReceiptRef(connectorRequest, output, observation.Usage, observation.ObservedAt)
	if err != nil || observation.ReceiptRef != wantReceipt {
		return ToolInvocationResult{}, toolFailure(ToolErrorReceiptInvalid, err)
	}
	result := ToolInvocationResult{
		ToolID: registration.Spec.ID, Version: registration.Spec.Version,
		RequestRef: request.RequestRef, SpecDigest: registration.Digest,
		Catalog: catalog, CapabilityRef: request.Curation.CapabilityRef, Execution: request.Curation.Execution,
		AuthorizationReceiptRefs: append([]string(nil), authorizationRefs...),
		ObservationReceiptRef:    observation.ReceiptRef, ObservedAt: observation.ObservedAt.UTC(),
		Usage: observation.Usage,
	}
	if int64(len(observation.Output)) <= registration.Spec.Output.InlineBytes {
		result.InlineOutput = append(json.RawMessage(nil), output...)
		return result, nil
	}
	putRequest := ports.PutArtifactRequest{MediaType: "application/json", Content: output}
	stored, err := orchestrator.artifacts.Put(ctx, putRequest)
	if err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorArtifactFailed, err)
	}
	if err := ports.ValidateStoredArtifact(putRequest, stored); err != nil {
		return ToolInvocationResult{}, toolFailure(ToolErrorArtifactFailed, err)
	}
	result.Artifact, result.ArtifactStored = stored, true
	return result, nil
}
func validToolObservationEnvelope(request ToolExecutionRequest, observation ToolObservation, causalFloor, observedCeiling time.Time) bool {
	return observation.ToolID == request.ToolID && observation.Version == request.Version &&
		observation.RequestRef == request.RequestRef && observation.IdempotencyKey == request.IdempotencyKey &&
		observation.SpecDigest == request.SpecDigest && observation.Catalog == request.Catalog &&
		observation.CapabilityRef == request.CapabilityRef && observation.Execution == request.Execution &&
		observation.PrincipalRef == request.PrincipalRef && observation.ActorRef == request.ActorRef &&
		observation.ProjectRef == request.ProjectRef &&
		slices.Equal(observation.AuthorizationReceiptRefs, request.AuthorizationReceiptRefs) &&
		!observation.ObservedAt.IsZero() && !observation.ObservedAt.Before(causalFloor) &&
		!observation.ObservedAt.After(observedCeiling)
}
func ToolObservationReceiptRef(request ToolExecutionRequest, output []byte, usage governance.ResourceUsage, observedAt time.Time) (string, error) {
	if !validToolInvocationRef(request.ToolID) || !validToolInvocationRef(request.Version) ||
		!validToolInvocationRef(request.RequestRef) ||
		!validToolDigestRef(request.IdempotencyKey, "tool-observation:sha256:") ||
		!validToolDigestRef(request.SpecDigest, "sha256:") ||
		!validToolInvocationRef(request.Catalog.ID) || !validToolInvocationRef(request.Catalog.Version) ||
		!validToolDigestRef(request.Catalog.Digest, "sha256:") ||
		!validToolInvocationRef(request.Catalog.ReviewRef) ||
		!validToolDigestRef(request.Catalog.ReviewDigest, "sha256:") ||
		request.CapabilityRef.String() == "" || !validToolExecutionBinding(request.Execution, request.ToolID) ||
		request.PrincipalRef.String() == "" ||
		request.ActorRef.String() == "" || request.ProjectRef.String() == "" ||
		len(request.AuthorizationReceiptRefs) == 0 || observedAt.IsZero() ||
		!validToolResourceUsage(usage) {
		return "", toolFailure(ToolErrorReceiptInvalid, nil)
	}
	for _, ref := range request.AuthorizationReceiptRefs {
		if !validToolInvocationRef(ref) {
			return "", toolFailure(ToolErrorReceiptInvalid, nil)
		}
	}
	digest := sha256.New()
	fields := []string{
		"orquesta.tool.observation-receipt.v1", request.ToolID, request.Version, request.RequestRef, request.IdempotencyKey, request.SpecDigest,
		request.Catalog.ID, request.Catalog.Version, request.Catalog.Digest,
		request.Catalog.ReviewRef, request.Catalog.ReviewDigest, request.CapabilityRef.String(),
		toolExecutionBindingDigest(request.Execution), request.PrincipalRef.String(), request.ActorRef.String(),
		request.ProjectRef.String(), strconv.Itoa(len(request.AuthorizationReceiptRefs))}
	fields = append(fields, request.AuthorizationReceiptRefs...)
	fields = append(fields,
		string(output), strconv.FormatInt(usage.Resources.Tokens, 10),
		strconv.FormatInt(usage.Resources.MoneyMicros, 10), string(usage.Resources.Currency),
		strconv.FormatInt(usage.Resources.ActiveTimeNS, 10), strconv.FormatInt(usage.Resources.ProcessSlots, 10),
		strconv.FormatInt(usage.Resources.DiskBytes, 10),
		strconv.FormatUint(uint64(usage.Known), 10), string(usage.Quality),
		observedAt.UTC().Format(time.RFC3339Nano))
	writeToolDigestFields(digest, fields...)
	return "tool-observation-receipt:sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}
func validToolExecutionBinding(binding ToolExecutionBinding, toolID string) bool {
	return binding.ToolRef.String() == "tool:"+toolID && binding.GoalRef.String() != "" &&
		binding.WorkItemRef.String() != "" && binding.ExecutionRef.String() != "" &&
		binding.PlanGeneration != 0 && binding.AppSpecGeneration != 0 && binding.ExecutionAttempt != 0 &&
		goal.IsCanonicalAppSpecHash(binding.SpecHash)
}
func toolExecutionIsCurrent(record GoalRecord, projectRef goal.ProjectRef, capability goal.CapabilityRef, binding ToolExecutionBinding) bool {
	if record.Goal.Ref() != binding.GoalRef || record.Goal.Project() != projectRef ||
		record.Goal.State() != goal.GoalStateRunning || record.Goal.PlanGeneration() != binding.PlanGeneration ||
		record.Goal.AppSpec().Generation() != binding.AppSpecGeneration || record.Goal.SpecHash() != binding.SpecHash {
		return false
	}
	item, found := record.Goal.WorkItem(binding.WorkItemRef)
	current, bound := item.Execution()
	if !found || item.Goal() != binding.GoalRef || item.Project() != projectRef || item.State() != goal.WorkItemStateRunning ||
		!slices.Contains(item.CapabilityRefs(), capability) || !slices.Contains(item.ToolRefs(), binding.ToolRef) ||
		!bound || current != binding.ExecutionRef {
		return false
	}
	matches := 0
	for _, execution := range record.Executions {
		if execution.Ref != binding.ExecutionRef {
			continue
		}
		matches++
		if execution.GoalRef != binding.GoalRef || execution.WorkItemRef != binding.WorkItemRef ||
			execution.State != ExecutionRunning || execution.AttemptNo != binding.ExecutionAttempt ||
			execution.PlanGeneration != binding.PlanGeneration || execution.AppSpecGeneration != binding.AppSpecGeneration ||
			execution.SpecHash != binding.SpecHash {
			return false
		}
	}
	return matches == 1
}
func toolExecutionBindingDigest(binding ToolExecutionBinding) string {
	return toolInvocationDigest("execution-binding", binding.ToolRef.String(), binding.GoalRef.String(),
		binding.WorkItemRef.String(), binding.ExecutionRef.String(), strconv.FormatUint(uint64(binding.PlanGeneration), 10),
		strconv.FormatUint(uint64(binding.AppSpecGeneration), 10), strconv.FormatUint(binding.ExecutionAttempt, 10), binding.SpecHash)
}
func validToolResourceUsage(usage governance.ResourceUsage) bool {
	return governance.ValidateResourceUsage(usage) == nil && usage.Known == governance.AllResourceDimensions &&
		(usage.Quality == governance.UsageQualityMeasured || usage.Quality == governance.UsageQualityExact)
}
func (orchestrator *Orchestrator) toolAuthorizationCurrent(ctx context.Context, principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef, role identity.Role, revision identity.MembershipRevision) bool {
	if role == identity.RolePlatformAdmin && revision == 0 {
		return true
	}
	membership, err := orchestrator.access.Membership(ctx, principalRef, projectRef)
	return err == nil && membership.IsActive() && membership.Role() == role && membership.Revision() == revision
}
func cloneToolExecutionRequest(request ToolExecutionRequest) ToolExecutionRequest {
	request.AuthorizationReceiptRefs = append([]string(nil), request.AuthorizationReceiptRefs...)
	request.Input = append([]byte(nil), request.Input...)
	return request
}
func validToolDigestRef(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
}
func validToolInvocationRef(value string) bool {
	return value != "" && utf8.ValidString(value) && strings.TrimSpace(value) == value &&
		strings.IndexFunc(value, unicode.IsControl) < 0
}
func toolInvocationDigest(fields ...string) string {
	digest := sha256.New()
	writeToolDigestFields(digest, append([]string{"orquesta.tool.invocation.v1"}, fields...)...)
	return hex.EncodeToString(digest.Sum(nil))
}
func toolAuthorizationRequestRef(requestRef, authority string) string {
	digest := sha256.New()
	writeToolDigestFields(digest, "orquesta.tool.authorization.v1", requestRef, authority)
	return "authorization-request:tool:sha256:" + hex.EncodeToString(digest.Sum(nil))
}
func writeToolDigestFields(digest hash.Hash, fields ...string) {
	var size [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
}
func toolFailure(code string, _ error) error {
	return &ToolInvocationError{Code: code}
}
