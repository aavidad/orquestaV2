package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type handler func(context.Context, handlerContext, json.RawMessage) (json.RawMessage, error)

type Executor interface {
	Dispatch(context.Context, Invocation) Result
	Definitions() []Definition
	Limits() APILimits
}

type handlerContext struct {
	access       application.Access
	principal    identity.Principal
	projectRef   goal.ProjectRef
	executionRef goal.ExecutionRef
	requestRef   string
	requestedAt  time.Time
}

type Dispatcher struct {
	application        applicationAPI
	audit              AuditPort
	executionAuthority ExecutionAuthorityResolver
	limits             APILimits
	definitions        []Definition
	byID               map[string]Definition
	handlers           map[string]handler
}

func NewDispatcher(
	orchestrator *application.Orchestrator,
	audit AuditPort,
	limits APILimits,
	executionAuthority ...ExecutionAuthorityResolver,
) (*Dispatcher, error) {
	if orchestrator == nil || audit == nil || !limits.Valid() {
		return nil, errors.New("commands.dependencies_required")
	}
	return newDispatcher(orchestrator, audit, limits, executionAuthority...)
}

func newDispatcher(
	api applicationAPI,
	audit AuditPort,
	limits APILimits,
	executionAuthority ...ExecutionAuthorityResolver,
) (*Dispatcher, error) {
	if api == nil || audit == nil || !limits.Valid() || len(executionAuthority) > 1 {
		return nil, errors.New("commands.dependencies_required")
	}
	var authority ExecutionAuthorityResolver
	if len(executionAuthority) == 1 {
		authority = executionAuthority[0]
	}
	dispatcher := &Dispatcher{
		application: api, audit: audit, executionAuthority: authority, limits: limits,
	}
	dispatcher.handlers = dispatcher.applicationHandlers()
	dispatcher.definitions = cloneDefinitions(compiledDefinitions)
	dispatcher.byID = make(map[string]Definition, len(dispatcher.definitions))
	bindings := make(map[string]struct{}, len(dispatcher.definitions)*3)
	for _, definition := range dispatcher.definitions {
		if err := validateDefinition(definition); err != nil {
			return nil, err
		}
		if _, duplicate := dispatcher.byID[definition.ID]; duplicate {
			return nil, errors.New("commands.definition_duplicate")
		}
		if _, ok := dispatcher.handlers[definition.Handler]; !ok {
			return nil, errors.New("commands.handler_unknown")
		}
		for _, binding := range []string{definition.HTTP.Method + " " + definition.HTTP.Path, definition.MCP.Tool, strings.Join(definition.CLI.Path, "\x00")} {
			if _, duplicate := bindings[binding]; duplicate {
				return nil, errors.New("commands.binding_duplicate")
			}
			bindings[binding] = struct{}{}
		}
		dispatcher.byID[definition.ID] = definition
	}
	return dispatcher, nil
}

func (dispatcher *Dispatcher) Definitions() []Definition {
	if dispatcher == nil {
		return nil
	}
	return cloneDefinitions(dispatcher.definitions)
}

func (dispatcher *Dispatcher) Limits() APILimits {
	if dispatcher == nil {
		return APILimits{}
	}
	return dispatcher.limits
}

func CanonicalDefinitions() []Definition { return cloneDefinitions(compiledDefinitions) }

func (dispatcher *Dispatcher) Dispatch(ctx context.Context, invocation Invocation) Result {
	result := Result{CommandID: invocation.CommandID, CommandVersion: invocation.CommandVersion, RequestRef: invocation.RequestRef}
	if dispatcher == nil || ctx == nil {
		result.Failure = failure(CodeUnavailable)
		return result
	}
	definition, ok := dispatcher.byID[invocation.CommandID]
	if !ok {
		result.Failure = failure(CodeInvalidRequest)
		return result
	}
	result.CommandVersion = definition.Version
	if invocation.CommandVersion != definition.Version || !validOpaque(invocation.RequestRef) {
		result.Failure = failure(CodeInvalidRequest)
		return result
	}
	bound, fail := dispatcher.bindAuthority(ctx, definition, invocation)
	if fail != nil {
		result.Failure = fail
		return result
	}
	if int64(len(invocation.Payload)) > dispatcher.limits.MaxRequestBytes {
		result.Failure = failure(CodeInvalidRequest)
		return result
	}
	payload, err := validatePayload(definition.InputSchema, invocation.Payload)
	if err != nil {
		result.Failure = failure(CodeInvalidRequest)
		return result
	}
	if err := validateListLimit(definition.ID, payload, dispatcher.limits.MaxListLimit); err != nil {
		result.Failure = failure(CodeInvalidRequest)
		return result
	}
	record := admissionRecord(definition, invocation, bound, payload)
	session, err := dispatcher.audit.Begin(ctx, record)
	if err != nil {
		if errors.Is(err, ErrAuditConflict) {
			result.Failure = failure(CodeConflict)
		} else {
			result.Failure = failure(CodeUnavailable)
		}
		return result
	}
	if !sameAdmission(record, session) {
		result.Failure = failure(CodeUnavailable)
		return result
	}
	result.AuditRef = session.Record.Ref
	bound.requestedAt = session.Record.AdmittedAt
	if session.Terminal != nil && session.Terminal.Status != "completed" {
		return replayTerminalFailure(result, *session.Terminal)
	}
	data, callErr := dispatcher.handlers[definition.Handler](ctx, bound, payload)
	if callErr == nil {
		data, err = validatePayload(definition.OutputSchema, data)
		if err != nil {
			callErr = errors.New("commands.output_contract_invalid")
			data = nil
		}
	}
	result.Data = cloneJSON(data)
	result.Failure = applicationFailure(callErr)
	wantTerminal := terminalFor(result, session.OutcomeRef)
	if session.Terminal != nil {
		if !sameTerminal(wantTerminal, *session.Terminal) {
			result.Data = nil
			result.Failure = failure(CodeConflict)
		}
		return result
	}
	completion, completeErr := dispatcher.audit.Complete(ctx, AuditCompletionRequest{RecordRef: session.Record.Ref, Terminal: wantTerminal})
	if completeErr != nil {
		result.Data = nil
		if errors.Is(completeErr, ErrAuditConflict) {
			result.Failure = failure(CodeConflict)
		} else {
			result.Failure = failure(CodeUnavailable)
		}
		return result
	}
	if !sameTerminal(wantTerminal, completion.Terminal) {
		result.Data = nil
		result.Failure = failure(CodeConflict)
		return result
	}
	// Created is an explicit append-only outcome fact. False is valid only
	// when a concurrent exact execution won the unique outcome identity.
	if !completion.Created && completion.Terminal.CompletedAt.IsZero() {
		result.Data = nil
		result.Failure = failure(CodeUnavailable)
	}
	return result
}

func (dispatcher *Dispatcher) bindAuthority(
	ctx context.Context,
	definition Definition,
	invocation Invocation,
) (handlerContext, *Failure) {
	if identity.ValidatePrincipal(invocation.Principal) != nil {
		return handlerContext{}, failure(CodeUnauthenticated)
	}
	projectRef, err := goal.NewProjectRef(invocation.ProjectRef)
	if err != nil {
		return handlerContext{}, failure(CodeInvalidRequest)
	}
	bound := handlerContext{principal: invocation.Principal, projectRef: projectRef, requestRef: invocation.RequestRef}
	var classifiedExecution goal.ExecutionRef
	if invocation.Principal.Kind == identity.PrincipalKindService {
		classifier, ok := dispatcher.executionAuthority.(ExecutionPrincipalClassifier)
		if !ok {
			return handlerContext{}, failure(CodeForbidden)
		}
		found, active, executionRef, classifyErr := classifier.ClassifyExecutionPrincipal(
			ctx, invocation.Principal, projectRef,
		)
		if classifyErr != nil || found != (executionRef.String() != "") || active && !found {
			return handlerContext{}, failure(CodeForbidden)
		}
		if found {
			if !active || !definition.ExecutionBound &&
				definition.Permission != string(identity.PermissionArtifactsRead) {
				return handlerContext{}, failure(CodeForbidden)
			}
			classifiedExecution = executionRef
		} else if definition.ExecutionBound {
			return handlerContext{}, failure(CodeForbidden)
		}
	}
	if definition.ExecutionBound {
		if !validOpaque(invocation.ClaimedExecutionRef) || dispatcher.executionAuthority == nil {
			return handlerContext{}, failure(CodeForbidden)
		}
		claimed, err := goal.NewExecutionRef(invocation.ClaimedExecutionRef)
		if err != nil {
			return handlerContext{}, failure(CodeForbidden)
		}
		resolved := classifiedExecution
		if resolved.String() == "" {
			resolved, err = dispatcher.executionAuthority.ResolveExecution(
				ctx, invocation.Principal, projectRef, claimed,
			)
			if err != nil {
				return handlerContext{}, failure(CodeForbidden)
			}
		}
		if resolved.String() == "" || resolved != claimed {
			return handlerContext{}, failure(CodeForbidden)
		}
		bound.executionRef = resolved
		bound.access, err = application.NewExecutionAccess(invocation.Principal, projectRef, resolved)
		if err != nil {
			return handlerContext{}, failure(CodeUnauthenticated)
		}
		return bound, nil
	}
	if invocation.ClaimedExecutionRef != "" {
		return handlerContext{}, failure(CodeInvalidRequest)
	}
	bound.access, err = application.NewAccess(invocation.Principal, projectRef)
	if err != nil {
		return handlerContext{}, failure(CodeUnauthenticated)
	}
	return bound, nil
}

func admissionRecord(definition Definition, invocation Invocation, bound handlerContext, payload json.RawMessage) CommandAuditRecord {
	inputDigest := digestFields(string(payload))
	schemaDigest := digestFields(string(definition.InputSchema), string(definition.OutputSchema))
	ref := "command-audit:" + digestFields(
		invocation.Principal.Ref.String(), bound.projectRef.String(), definition.ID, definition.Version, invocation.RequestRef,
	)
	return CommandAuditRecord{
		Ref: ref, CommandID: definition.ID, CommandVersion: definition.Version,
		RegistryDigest: registryAdmissionDigest(definition), SchemaDigest: schemaDigest,
		RequestRef: invocation.RequestRef, InputDigest: inputDigest,
		PrincipalRef: invocation.Principal.Ref.String(), ProjectRef: bound.projectRef.String(),
		AuthenticatedExecutionRef: bound.executionRef.String(), ReplayMode: definition.ReplayMode, Status: "admitted",
	}
}

func replayTerminalFailure(result Result, terminal CommandAuditTerminal) Result {
	result.Failure = failure(terminal.ErrorCode)
	want := terminalFor(result, terminal.OutcomeRef)
	if terminal.Status == "completed" || !sameTerminal(want, terminal) {
		result.Failure = failure(CodeUnavailable)
	}
	return result
}

func terminalFor(result Result, outcomeRef string) CommandAuditTerminal {
	status, code := "completed", failureCode(result.Failure)
	if result.Failure != nil {
		status = "rejected"
		if code == CodeUnavailable || code == CodeInternal {
			status = "failed"
		}
	}
	return CommandAuditTerminal{OutcomeRef: outcomeRef, Status: status, ErrorCode: code,
		OutputDigest: resultEnvelopeDigest(result)}
}

// resultEnvelopeDigest is the single terminal outcome authority. It covers the
// complete public Result envelope, not only its data or failure fields.
func resultEnvelopeDigest(result Result) string {
	encoded, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	var value any
	if err := decodeSingle(encoded, &value, true); err != nil {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return digestFields(string(canonical))
}

func sameAdmission(want CommandAuditRecord, session AuditSession) bool {
	got := session.Record
	if !sameAdmissionFacts(want, got) || got.AdmittedAt.IsZero() || !validOpaque(session.OutcomeRef) {
		return false
	}
	if got.Status != "admitted" || got.ErrorCode != "" || got.OutputDigest != "" || !got.CompletedAt.IsZero() {
		return false
	}
	if session.AdmissionCreated && session.Terminal != nil {
		return false
	}
	return session.OutcomeRef == got.Ref+":outcome"
}

func sameAdmissionFacts(left, right CommandAuditRecord) bool {
	return left.Ref == right.Ref && left.CommandID == right.CommandID && left.CommandVersion == right.CommandVersion &&
		left.RegistryDigest == right.RegistryDigest && left.SchemaDigest == right.SchemaDigest &&
		left.RequestRef == right.RequestRef && left.InputDigest == right.InputDigest &&
		left.PrincipalRef == right.PrincipalRef && left.ProjectRef == right.ProjectRef &&
		left.AuthenticatedExecutionRef == right.AuthenticatedExecutionRef && left.ReplayMode == right.ReplayMode &&
		left.Status == right.Status
}

func sameTerminal(want, got CommandAuditTerminal) bool {
	return validTerminal(want, false) && validTerminal(got, true) &&
		got.OutcomeRef == want.OutcomeRef && got.Status == want.Status && got.ErrorCode == want.ErrorCode &&
		got.OutputDigest == want.OutputDigest && !got.CompletedAt.IsZero()
}

func validTerminal(value CommandAuditTerminal, persisted bool) bool {
	if !validOpaque(value.OutcomeRef) || value.OutputDigest == "" || persisted != !value.CompletedAt.IsZero() {
		return false
	}
	switch value.Status {
	case "completed":
		return value.ErrorCode == ""
	case "rejected":
		return isFailureCode(value.ErrorCode) && value.ErrorCode != CodeUnavailable && value.ErrorCode != CodeInternal
	case "failed":
		return value.ErrorCode == CodeUnavailable || value.ErrorCode == CodeInternal
	default:
		return false
	}
}

func validateListLimit(commandID string, payload json.RawMessage, maximum int) error {
	if commandID != "orquesta.goals.list" && commandID != "orquesta.changes.list" && commandID != "orquesta.mailbox.list" {
		return nil
	}
	var value struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(payload, &value); err != nil || value.Limit <= 0 || value.Limit > maximum {
		return errors.New("commands.list_limit_invalid")
	}
	return nil
}
