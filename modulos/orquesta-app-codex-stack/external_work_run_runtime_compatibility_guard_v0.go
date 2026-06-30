package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	ExternalWorkRunRuntimeCompatibilityMismatchV0          = "external_work_runtime_compatibility_mismatch"
	ExternalWorkRunRuntimeCompatibilityIdentityMissingV0   = "external_work_runtime_identity_missing"
	ExternalWorkRunRuntimeCompatibilityRequiredMissingV0   = "external_work_runtime_compatibility_required_missing"
	ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0 = "orquesta_runtime_binary_sha256"
	ExternalWorkRunRuntimeCompatibilityFieldBuildRefV0     = "orquesta_runtime_build_ref"
	ExternalWorkRunRuntimeCompatibilityFieldCommitRefV0    = "orquesta_runtime_commit_ref"
	ExternalWorkRunRuntimeCompatibilityFieldRequiredV0     = "orquesta_runtime_compatibility_required"
)

type ExternalWorkRunRuntimeCompatibilityGuardConfigV0 struct {
	RuntimeIdentity ExternalWorkRunRuntimeIdentityV0
}

type ExternalWorkRunRuntimeIdentityV0 struct {
	BinarySHA256 string
	BuildRef     string
	CommitRef    string
	EvidenceRefs []string
}

type ExternalWorkRunRuntimeCompatibilityGuardExecutorV0 struct {
	Next   orquestamcp.MCPTransportExternalWorkRunExecutorV0
	Config ExternalWorkRunRuntimeCompatibilityGuardConfigV0
}

func NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(
	next orquestamcp.MCPTransportExternalWorkRunExecutorV0,
	config ExternalWorkRunRuntimeCompatibilityGuardConfigV0,
) ExternalWorkRunRuntimeCompatibilityGuardExecutorV0 {
	config.RuntimeIdentity = normalizeExternalWorkRunRuntimeIdentityV0(config.RuntimeIdentity)
	return ExternalWorkRunRuntimeCompatibilityGuardExecutorV0{Next: next, Config: config}
}

func (executor ExternalWorkRunRuntimeCompatibilityGuardExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunToolResultV0, error) {
	if issue, blocked := executor.blockingIssueV0(input); blocked {
		return orquestamcp.MCPExternalWorkRunToolResultV0{
			Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			RequestID:     strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(input.RequestID, input.ExternalWorkRunRequest.RequestID)),
			CorrelationID: strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(input.CorrelationID, input.ExternalWorkRunRequest.CorrelationID, input.RequestID)),
			NextActions: []string{
				"usar_runtime_orquesta_aprobado",
				"actualizar_contrato_runtime_del_consumidor",
			},
			Errores: []orquestamcp.MCPExternalWorkRunIssueV0{issue},
		}, nil
	}
	if executor.Next == nil {
		return orquestamcp.MCPExternalWorkRunToolResultV0{
			Estado: orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
				Code:  "external_work_runtime_compatibility_guard_next_missing",
				Field: "executor",
			}},
		}, nil
	}
	return executor.Next.Execute(ctx, input)
}

func (executor ExternalWorkRunRuntimeCompatibilityGuardExecutorV0) blockingIssueV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunIssueV0, bool) {
	expectation := externalWorkRunRuntimeCompatibilityExpectationV0(input)
	if !expectation.required && expectation.binarySHA256 == "" && expectation.buildRef == "" && expectation.commitRef == "" {
		return orquestamcp.MCPExternalWorkRunIssueV0{}, false
	}
	identity := normalizeExternalWorkRunRuntimeIdentityV0(executor.Config.RuntimeIdentity)
	if externalWorkRunRuntimeIdentityEmptyV0(identity) {
		return externalWorkRunRuntimeCompatibilityIssueV0(
			ExternalWorkRunRuntimeCompatibilityIdentityMissingV0,
			"runtime_identity:action_restart_with_runtime_identity",
		), true
	}
	if expectation.required && expectation.binarySHA256 == "" && expectation.buildRef == "" && expectation.commitRef == "" {
		return externalWorkRunRuntimeCompatibilityIssueV0(
			ExternalWorkRunRuntimeCompatibilityRequiredMissingV0,
			"input_fields:action_set_expected_runtime_identity",
		), true
	}
	if expectation.binarySHA256 != "" && !strings.EqualFold(identity.BinarySHA256, expectation.binarySHA256) {
		return externalWorkRunRuntimeCompatibilityIssueV0(
			ExternalWorkRunRuntimeCompatibilityMismatchV0,
			ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0+":action_use_approved_orquesta_runtime",
		), true
	}
	if expectation.buildRef != "" && !strings.EqualFold(identity.BuildRef, expectation.buildRef) {
		return externalWorkRunRuntimeCompatibilityIssueV0(
			ExternalWorkRunRuntimeCompatibilityMismatchV0,
			ExternalWorkRunRuntimeCompatibilityFieldBuildRefV0+":action_use_approved_orquesta_runtime",
		), true
	}
	if expectation.commitRef != "" && !strings.EqualFold(identity.CommitRef, expectation.commitRef) {
		return externalWorkRunRuntimeCompatibilityIssueV0(
			ExternalWorkRunRuntimeCompatibilityMismatchV0,
			ExternalWorkRunRuntimeCompatibilityFieldCommitRefV0+":action_use_approved_orquesta_runtime",
		), true
	}
	return orquestamcp.MCPExternalWorkRunIssueV0{}, false
}

type externalWorkRunRuntimeCompatibilityExpectationDataV0 struct {
	required     bool
	binarySHA256 string
	buildRef     string
	commitRef    string
}

func externalWorkRunRuntimeCompatibilityExpectationV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) externalWorkRunRuntimeCompatibilityExpectationDataV0 {
	expectation := externalWorkRunRuntimeCompatibilityExpectationDataV0{}
	for _, field := range externalWorkRunRuntimeCompatibilityFieldsV0(input) {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		value := firstExternalWorkRunRuntimeCompatibilityFieldValueV0(field)
		switch name {
		case ExternalWorkRunRuntimeCompatibilityFieldRequiredV0:
			expectation.required = parseExternalWorkRunRuntimeCompatibilityBoolV0(value)
		case ExternalWorkRunRuntimeCompatibilityFieldBinarySHA256V0,
			"runtime_binary_sha256",
			"required_runtime_binary_sha256":
			expectation.binarySHA256 = value
		case ExternalWorkRunRuntimeCompatibilityFieldBuildRefV0,
			"runtime_build_ref",
			"required_runtime_build_ref":
			expectation.buildRef = value
		case ExternalWorkRunRuntimeCompatibilityFieldCommitRefV0,
			"runtime_commit_ref",
			"required_runtime_commit_ref":
			expectation.commitRef = value
		}
	}
	return expectation
}

func externalWorkRunRuntimeCompatibilityFieldsV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	appChange := externalWorkRunGuardAppChangeV0(input)
	if appChange.ExternalWork == nil {
		return nil
	}
	return appChange.ExternalWork.InputFields
}

func firstExternalWorkRunRuntimeCompatibilityFieldValueV0(field orquestadomainwork.DomainWorkFieldV0) string {
	if strings.TrimSpace(field.Value) != "" {
		return strings.TrimSpace(field.Value)
	}
	for _, value := range field.Values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseExternalWorkRunRuntimeCompatibilityBoolV0(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "si", "required", "requerido":
		return true
	default:
		return false
	}
}

func externalWorkRunRuntimeCompatibilityIssueV0(
	code string,
	field string,
) orquestamcp.MCPExternalWorkRunIssueV0 {
	return orquestamcp.MCPExternalWorkRunIssueV0{
		Code:    code,
		Field:   field,
		Message: "external_work/run requiere un runtime Orquesta compatible con el contrato opt-in del consumidor",
	}
}

func normalizeExternalWorkRunRuntimeIdentityV0(
	identity ExternalWorkRunRuntimeIdentityV0,
) ExternalWorkRunRuntimeIdentityV0 {
	identity.BinarySHA256 = strings.TrimSpace(identity.BinarySHA256)
	identity.BuildRef = strings.TrimSpace(identity.BuildRef)
	identity.CommitRef = strings.TrimSpace(identity.CommitRef)
	identity.EvidenceRefs = compactExternalWorkRunGuardStringsV0(identity.EvidenceRefs)
	return identity
}

func externalWorkRunRuntimeIdentityEmptyV0(identity ExternalWorkRunRuntimeIdentityV0) bool {
	return strings.TrimSpace(identity.BinarySHA256) == "" &&
		strings.TrimSpace(identity.BuildRef) == "" &&
		strings.TrimSpace(identity.CommitRef) == "" &&
		len(identity.EvidenceRefs) == 0
}

var _ orquestamcp.MCPTransportExternalWorkRunExecutorV0 = ExternalWorkRunRuntimeCompatibilityGuardExecutorV0{}
