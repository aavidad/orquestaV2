package orquestaruntimecodexgoal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	CodexGoalStartPacketSchemaV0        = "codex_goal_start_packet.v0"
	CodexGoalObservationRequestSchemaV0 = "codex_goal_observation_request.v0"
	CodexGoalResultMarkerV0             = "ORQUESTA_GOAL_RESULT_V0"
	CodexGoalResultFileNameV0           = "orquesta_goal_result_v0.json"
	CodexGoalMaxPromptBytesV0           = 32 * 1024
	CodexGoalMaxStartPacketBytesV0      = 64 * 1024

	ErrCodexGoalStarterMissingV0      = "codex_goal_starter_missing"
	ErrCodexGoalObserverMissingV0     = "codex_goal_observer_missing"
	ErrCodexGoalSpecInvalidV0         = "codex_goal_spec_invalid"
	ErrCodexGoalObservationInvalidV0  = "codex_goal_observation_invalid"
	ErrCodexGoalStartRejectedV0       = "codex_goal_start_rejected"
	ErrCodexGoalObservationRejectedV0 = "codex_goal_observation_rejected"
	ErrCodexGoalPromptTooLargeV0      = "codex_goal_prompt_too_large"
	ErrCodexGoalStartPacketTooLargeV0 = "codex_goal_start_packet_too_large"
)

type CodexGoalStartPacketV0 struct {
	SchemaVersion      string                                `json:"schema_version"`
	GoalRef            string                                `json:"goal_ref"`
	RequestRef         string                                `json:"request_ref,omitempty"`
	ProjectRef         string                                `json:"project_ref,omitempty"`
	WorkKind           string                                `json:"work_kind,omitempty"`
	WorkProfileKind    string                                `json:"work_profile_kind,omitempty"`
	Objective          string                                `json:"objective"`
	Prompt             string                                `json:"prompt"`
	ContextRefs        []orquestagoal.GoalContextRefV0       `json:"context_refs,omitempty"`
	RuleRefs           []orquestagoal.GoalRuleRefV0          `json:"rule_refs,omitempty"`
	SkillRefs          []string                              `json:"skill_refs,omitempty"`
	WriteSet           []orquestagoal.GoalWriteScopeV0       `json:"write_set,omitempty"`
	RequiredTests      []orquestagoal.GoalRequiredTestV0     `json:"required_tests,omitempty"`
	AcceptanceCriteria []string                              `json:"acceptance_criteria,omitempty"`
	ArtifactContracts  []orquestagoal.GoalArtifactContractV0 `json:"artifact_contracts,omitempty"`
	EvidenceRefs       []string                              `json:"evidence_refs,omitempty"`
	Budget             orquestagoal.GoalBudgetV0             `json:"budget,omitempty"`
	ClosurePolicy      orquestagoal.GoalClosurePolicyV0      `json:"closure_policy,omitempty"`
	ReworkPolicy       orquestagoal.GoalReworkPolicyV0       `json:"rework_policy,omitempty"`
}

type CodexGoalStartReceiptV0 struct {
	Status          string   `json:"status"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
	IssueCode       string   `json:"issue_code,omitempty"`
}

type CodexGoalObservationRequestV0 struct {
	SchemaVersion   string `json:"schema_version"`
	GoalRef         string `json:"goal_ref"`
	ExternalGoalRef string `json:"external_goal_ref,omitempty"`
}

type CodexGoalObservationReceiptV0 struct {
	Status              string                                  `json:"status"`
	GoalRef             string                                  `json:"goal_ref,omitempty"`
	ExternalGoalRef     string                                  `json:"external_goal_ref,omitempty"`
	Summary             string                                  `json:"summary,omitempty"`
	ArtifactRefs        []string                                `json:"artifact_refs,omitempty"`
	RequiredTestResults []orquestagoal.GoalRequiredTestResultV0 `json:"required_test_results,omitempty"`
	DomainReceiptRefs   []string                                `json:"domain_receipt_refs,omitempty"`
	EvidenceRefs        []string                                `json:"evidence_refs,omitempty"`
	IssueCode           string                                  `json:"issue_code,omitempty"`
}

type CodexGoalStarterPortV0 interface {
	StartCodexGoalV0(context.Context, CodexGoalStartPacketV0) (CodexGoalStartReceiptV0, error)
}

type CodexGoalObserverPortV0 interface {
	ObserveCodexGoalV0(context.Context, CodexGoalObservationRequestV0) (CodexGoalObservationReceiptV0, error)
}

type CodexGoalLauncherV0 struct {
	Starter CodexGoalStarterPortV0
}

type CodexGoalObserverV0 struct {
	Observer CodexGoalObserverPortV0
}

func BuildCodexGoalStartPacketV0(spec orquestagoal.GoalWorkSpecV0) (CodexGoalStartPacketV0, []orquestagoal.GoalWorkIssueV0) {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	spec.DirectorKind = orquestagoal.GoalDirectorKindCodexGoalV0
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return CodexGoalStartPacketV0{}, issues
	}
	prompt := BuildCodexGoalPromptV0(spec)
	if len([]byte(prompt)) > CodexGoalMaxPromptBytesV0 {
		return CodexGoalStartPacketV0{}, []orquestagoal.GoalWorkIssueV0{{
			Code:  ErrCodexGoalPromptTooLargeV0,
			Field: "prompt",
		}}
	}
	packet := CodexGoalStartPacketV0{
		SchemaVersion:      CodexGoalStartPacketSchemaV0,
		GoalRef:            spec.GoalRef,
		RequestRef:         spec.RequestRef,
		ProjectRef:         spec.ProjectRef,
		WorkKind:           spec.WorkKind,
		WorkProfileKind:    spec.WorkProfileKind,
		Objective:          spec.Objective,
		Prompt:             prompt,
		ContextRefs:        append([]orquestagoal.GoalContextRefV0(nil), spec.ContextRefs...),
		RuleRefs:           append([]orquestagoal.GoalRuleRefV0(nil), spec.RuleRefs...),
		SkillRefs:          append([]string(nil), spec.SkillRefs...),
		WriteSet:           append([]orquestagoal.GoalWriteScopeV0(nil), spec.WriteSet...),
		RequiredTests:      append([]orquestagoal.GoalRequiredTestV0(nil), spec.RequiredTests...),
		AcceptanceCriteria: append([]string(nil), spec.AcceptanceCriteria...),
		ArtifactContracts:  append([]orquestagoal.GoalArtifactContractV0(nil), spec.ArtifactContracts...),
		EvidenceRefs:       append([]string(nil), spec.EvidenceRefs...),
		Budget:             spec.Budget,
		ClosurePolicy:      spec.ClosurePolicy,
		ReworkPolicy:       spec.ReworkPolicy,
	}
	packetBytes, err := json.Marshal(packet)
	if err != nil {
		return CodexGoalStartPacketV0{}, []orquestagoal.GoalWorkIssueV0{{
			Code:  ErrCodexGoalSpecInvalidV0,
			Field: "packet",
		}}
	}
	if len(packetBytes) > CodexGoalMaxStartPacketBytesV0 {
		return CodexGoalStartPacketV0{}, []orquestagoal.GoalWorkIssueV0{{
			Code:  ErrCodexGoalStartPacketTooLargeV0,
			Field: "packet",
		}}
	}
	return packet, nil
}

func BuildCodexGoalObservationRequestV0(
	request orquestagoal.GoalObservationRequestV0,
) (CodexGoalObservationRequestV0, []orquestagoal.GoalWorkIssueV0) {
	request = orquestagoal.NormalizeGoalObservationRequestV0(request)
	if issues := orquestagoal.ValidateGoalObservationRequestV0(request); len(issues) > 0 {
		return CodexGoalObservationRequestV0{}, issues
	}
	return CodexGoalObservationRequestV0{
		SchemaVersion:   CodexGoalObservationRequestSchemaV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
	}, nil
}

func BuildCodexGoalPromptV0(spec orquestagoal.GoalWorkSpecV0) string {
	var b strings.Builder
	b.WriteString("Eres el Director operativo interno de este Codex Goal.\n")
	b.WriteString("Orquesta gobierna desde fuera: objetivo, reglas, write-set, tests, artefactos y cierre.\n")
	b.WriteString("No intentes reactivar el loop historico de Orquesta; trabaja dentro del goal hasta complete o blocked.\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(spec.Objective)
	b.WriteString("\n\nMetadatos:\n")
	if spec.RequestRef != "" {
		b.WriteString("- request_ref: ")
		b.WriteString(spec.RequestRef)
		b.WriteString("\n")
	}
	if spec.ProjectRef != "" {
		b.WriteString("- project_ref: ")
		b.WriteString(spec.ProjectRef)
		b.WriteString("\n")
	}
	if spec.WorkKind != "" {
		b.WriteString("- work_kind: ")
		b.WriteString(spec.WorkKind)
		b.WriteString("\n")
	}
	if spec.WorkProfileKind != "" {
		b.WriteString("- work_profile_kind: ")
		b.WriteString(spec.WorkProfileKind)
		b.WriteString("\n")
	}
	b.WriteString("\n\nReglas:\n")
	for _, rule := range spec.RuleRefs {
		b.WriteString("- ")
		b.WriteString(rule.Ref)
		if rule.Enforcement != "" {
			b.WriteString(" [")
			b.WriteString(rule.Enforcement)
			b.WriteString("]")
		}
		b.WriteString("\n")
	}
	b.WriteString("\nContexto:\n")
	for _, ctx := range spec.ContextRefs {
		b.WriteString("- ")
		b.WriteString(ctx.Ref)
		if ctx.Kind != "" {
			b.WriteString(" kind=")
			b.WriteString(ctx.Kind)
		}
		if ctx.Required {
			b.WriteString(" [required]")
		}
		if ctx.Purpose != "" {
			b.WriteString(": ")
			b.WriteString(ctx.Purpose)
		}
		b.WriteString("\n")
	}
	b.WriteString("\nWrite-set autorizado:\n")
	for _, scope := range spec.WriteSet {
		b.WriteString("- ")
		b.WriteString(scope.Path)
		b.WriteString("\n")
	}
	b.WriteString("\nTests requeridos:\n")
	for _, test := range spec.RequiredTests {
		b.WriteString("- ")
		if test.TestRef != "" {
			b.WriteString("test_ref=")
			b.WriteString(test.TestRef)
			if test.CommandRef != "" || test.Command != "" {
				b.WriteString(" ")
			}
		}
		if test.CommandRef != "" {
			b.WriteString("command_ref=")
			b.WriteString(test.CommandRef)
			if test.Command != "" {
				b.WriteString(" ")
			}
		}
		if test.Command != "" {
			b.WriteString("command=")
			b.WriteString(test.Command)
		} else if test.TestRef == "" {
			b.WriteString(test.TestRef)
		}
		b.WriteString("\n")
	}
	b.WriteString("\nCriterios de aceptacion:\n")
	for _, criterion := range spec.AcceptanceCriteria {
		b.WriteString("- ")
		b.WriteString(criterion)
		b.WriteString("\n")
	}
	b.WriteString("\nArtefactos esperados:\n")
	for _, artifact := range spec.ArtifactContracts {
		b.WriteString("- ")
		b.WriteString(artifact.ArtifactRef)
		if artifact.ArtifactType != "" {
			b.WriteString(" type=")
			b.WriteString(artifact.ArtifactType)
		}
		if artifact.Required {
			b.WriteString(" required")
		}
		b.WriteString("\n")
	}
	if len(spec.ArtifactContracts) > 0 && len(spec.WriteSet) > 0 {
		b.WriteString("- Materializa cada artefacto requerido dentro de un write-set autorizado. Para artefactos DomainWork, Orquesta detecta ")
		b.WriteString("<artifact_type>.json, <artifact_type>.md, artifact.json o artifact.md directamente bajo el write-set; usa el artifact_type declarado y contenido verificable.\n")
	}
	if len(spec.WriteSet) > 0 {
		for _, scope := range spec.WriteSet {
			path := strings.Trim(strings.TrimSpace(scope.Path), "/")
			if codexGoalWriteScopeIsMarkdownFileV0(path) {
				b.WriteString("- El write-set ")
				b.WriteString(path)
				b.WriteString(" es un archivo Markdown final: crea o actualiza ese fichero, no crees un directorio con ese nombre.\n")
			}
		}
	}
	b.WriteString("\nCierre:\n")
	b.WriteString("- Devuelve complete solo con evidencias verificables.\n")
	if spec.ClosurePolicy.RequireRequiredTests {
		b.WriteString("- Deben pasar todos los tests requeridos.\n")
	}
	if spec.ClosurePolicy.RequireArtifacts {
		b.WriteString("- Deben existir los artefactos requeridos.\n")
	}
	if spec.ClosurePolicy.RequireDomainReceipt {
		if strings.TrimSpace(spec.WorkProfileKind) == "domain_work" {
			b.WriteString("- El receipt de dominio lo obtiene Orquesta fuera del goal despues de observar tu artefacto; no bloquees por no tener conector REST/MCP de la app externa.\n")
			b.WriteString("- En trabajos domain_work, no intentes llamar conectores REST/MCP de la app externa desde el sandbox salvo que el contrato te entregue ese puerto: materializa el artefacto en el write-set como entrega terminal y deja submit_artifact/receipt al adaptador externo de Orquesta.\n")
			b.WriteString("- No pongas public_blocker, pending, ready=false ni estados no terminales por faltar el conector externo; si el artefacto esta materializado para que Orquesta lo suba, el resultado del goal puede ser complete con domain_receipt_refs vacio.\n")
		} else {
			b.WriteString("- Debe existir receipt de dominio.\n")
		}
	}
	for _, evidenceRef := range spec.ClosurePolicy.RequiredEvidenceRefs {
		b.WriteString("- Evidencia requerida: ")
		b.WriteString(evidenceRef)
		b.WriteString("\n")
	}
	if spec.ReworkPolicy.PreferNewGoal {
		b.WriteString("- Si hace falta rework, prefiere abrir un nuevo goal causal.\n")
	}
	if spec.ReworkPolicy.PreserveArtifacts {
		b.WriteString("- Conserva artefactos aprovechables durante rework.\n")
	}
	if spec.Budget.TokenBudget > 0 || spec.Budget.MaxRuntimeSeconds > 0 || spec.Budget.MaxSubgoals > 0 {
		b.WriteString("- Respeta el presupuesto operativo declarado en el paquete.\n")
	}
	b.WriteString("- Devuelve blocked si falta input externo, permiso, proveedor o cambio de estado externo.\n")
	b.WriteString("- Conserva refs opacas y no publiques HOME, tokens, OAuth, comandos internos de runtime/local ni transcripts completos.\n")
	if len(spec.RequiredTests) > 0 {
		b.WriteString("- Los command de Tests requeridos son parte del contrato neutral acotado; puedes usarlos solo para reportar evidencias de esos tests.\n")
	}
	b.WriteString("\nResultado estructurado obligatorio:\n")
	b.WriteString("- Termina la respuesta final con una sola linea que empiece por ")
	b.WriteString(CodexGoalResultMarkerV0)
	b.WriteString(" seguida de JSON compacto.\n")
	b.WriteString("- El JSON debe usar esta forma: {\"goal_ref\":\"")
	b.WriteString(spec.GoalRef)
	b.WriteString("\",\"summary\":\"...\",\"artifact_refs\":[],\"required_test_results\":[{\"test_ref\":\"...\",\"status\":\"passed\",\"evidence_refs\":[]}],\"domain_receipt_refs\":[],\"evidence_refs\":[]}.\n")
	if spec.ClosurePolicy.RequireDomainReceipt && strings.TrimSpace(spec.WorkProfileKind) == "domain_work" {
		b.WriteString("- En domain_work deja domain_receipt_refs vacio salvo que el contrato te haya dado un receipt real; Orquesta lo derivara del ledger despues de submit_artifact.\n")
	}
	if len(spec.RequiredTests) > 0 {
		b.WriteString("- En required_test_results usa literalmente los test_ref declarados arriba; no los resumas ni inventes aliases.\n")
	}
	if len(spec.ArtifactContracts) > 0 {
		b.WriteString("- En artifact_refs usa literalmente los artifact_ref declarados arriba para los artefactos producidos; no uses rutas de fichero como refs.\n")
	}
	if resultFilePath := codexGoalResultFilePathV0(spec); resultFilePath != "" {
		b.WriteString("- Antes de marcar el goal como complete, escribe el mismo JSON en ")
		b.WriteString(resultFilePath)
		b.WriteString(" incluyendo \"goal_ref\":\"")
		b.WriteString(spec.GoalRef)
		b.WriteString("\" para que Orquesta pueda cerrar aunque no haya respuesta final textual.\n")
		b.WriteString("- Materializa primero el directorio del write-set y este archivo durable de resultado; si luego corriges artefactos o tests, actualiza el JSON antes de cerrar.\n")
	}
	b.WriteString("- Incluye en evidence_refs las evidencias requeridas solo si han sido verificadas; no inventes refs para forzar el cierre.\n")
	b.WriteString("- Incluye en artifact_refs solo artefactos producidos o verificados que cumplan el contrato.\n")
	return b.String()
}

func codexGoalResultFilePathV0(spec orquestagoal.GoalWorkSpecV0) string {
	for _, scope := range spec.WriteSet {
		path := strings.Trim(strings.TrimSpace(scope.Path), "/")
		if path == "" {
			continue
		}
		if codexGoalWriteScopeIsMarkdownFileV0(path) {
			continue
		}
		return path + "/docs/" + CodexGoalResultFileNameV0
	}
	return ""
}

func codexGoalWriteScopeIsMarkdownFileV0(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func (launcher CodexGoalLauncherV0) LaunchGoalWorkV0(ctx context.Context, spec orquestagoal.GoalWorkSpecV0) (orquestagoal.GoalLaunchReceiptV0, error) {
	if launcher.Starter == nil {
		return codexGoalLaunchInvalidReceiptV0(spec.GoalRef, ErrCodexGoalStarterMissingV0), errors.New(ErrCodexGoalStarterMissingV0)
	}
	packet, issues := BuildCodexGoalStartPacketV0(spec)
	if len(issues) > 0 {
		return orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       spec.GoalRef,
			Issues:        issues,
		}, errors.New(ErrCodexGoalSpecInvalidV0)
	}
	receipt, err := launcher.Starter.StartCodexGoalV0(ctx, packet)
	if err != nil {
		result := codexGoalLaunchInvalidReceiptV0(packet.GoalRef, ErrCodexGoalStartRejectedV0)
		if strings.TrimSpace(receipt.IssueCode) != "" {
			result.Issues = []orquestagoal.GoalWorkIssueV0{{Code: strings.TrimSpace(receipt.IssueCode)}}
		}
		if strings.TrimSpace(receipt.ExternalGoalRef) != "" {
			result.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
		}
		return result, err
	}
	status := receipt.Status
	if status == "" {
		status = orquestagoal.GoalStatusAcceptedV0
	}
	result := orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          status,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
		EvidenceRefs:    append([]string(nil), receipt.EvidenceRefs...),
	}
	if receipt.IssueCode != "" {
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{Code: receipt.IssueCode})
	}
	return result, nil
}

func (observer CodexGoalObserverV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if observer.Observer == nil {
		return codexGoalObservationInvalidResultV0(request, ErrCodexGoalObserverMissingV0), errors.New(ErrCodexGoalObserverMissingV0)
	}
	packet, issues := BuildCodexGoalObservationRequestV0(request)
	if len(issues) > 0 {
		return orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusInvalidV0,
			GoalRef:         strings.TrimSpace(request.GoalRef),
			ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
			Issues:          issues,
		}, errors.New(ErrCodexGoalObservationInvalidV0)
	}
	receipt, err := observer.Observer.ObserveCodexGoalV0(ctx, packet)
	if err != nil {
		result := codexGoalObservationInvalidResultV0(request, ErrCodexGoalObservationRejectedV0)
		if strings.TrimSpace(receipt.IssueCode) != "" {
			result.Issues = []orquestagoal.GoalWorkIssueV0{{Code: strings.TrimSpace(receipt.IssueCode)}}
		}
		if strings.TrimSpace(receipt.ExternalGoalRef) != "" {
			result.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
		}
		return result, err
	}
	result := goalWorkResultFromCodexObservationV0(packet, receipt)
	if result.GoalRef != packet.GoalRef {
		result.Status = orquestagoal.GoalStatusInvalidV0
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{
			Code:  orquestagoal.ErrGoalClosureInvalidV0,
			Field: "goal_ref",
		})
		return result, errors.New(ErrCodexGoalObservationInvalidV0)
	}
	if issues := orquestagoal.ValidateGoalWorkResultV0(result); len(issues) > 0 {
		result.Status = orquestagoal.GoalStatusInvalidV0
		result.Issues = append(result.Issues, issues...)
		return result, errors.New(ErrCodexGoalObservationInvalidV0)
	}
	return result, nil
}

func codexGoalLaunchInvalidReceiptV0(goalRef, code string) orquestagoal.GoalLaunchReceiptV0 {
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:        orquestagoal.GoalStatusInvalidV0,
		GoalRef:       strings.TrimSpace(goalRef),
		Issues:        []orquestagoal.GoalWorkIssueV0{{Code: code}},
	}
}

func goalWorkResultFromCodexObservationV0(
	request CodexGoalObservationRequestV0,
	receipt CodexGoalObservationReceiptV0,
) orquestagoal.GoalWorkResultV0 {
	status := strings.TrimSpace(receipt.Status)
	if status == "" {
		status = orquestagoal.GoalStatusRunningV0
	}
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion:       orquestagoal.GoalWorkResultSchemaV0,
		Status:              status,
		GoalRef:             firstNonEmptyCodexGoalStringV0(receipt.GoalRef, request.GoalRef),
		ExternalGoalRef:     firstNonEmptyCodexGoalStringV0(receipt.ExternalGoalRef, request.ExternalGoalRef),
		Summary:             strings.TrimSpace(receipt.Summary),
		ArtifactRefs:        append([]string(nil), receipt.ArtifactRefs...),
		RequiredTestResults: append([]orquestagoal.GoalRequiredTestResultV0(nil), receipt.RequiredTestResults...),
		DomainReceiptRefs:   append([]string(nil), receipt.DomainReceiptRefs...),
		EvidenceRefs:        append([]string(nil), receipt.EvidenceRefs...),
	}
	if receipt.IssueCode != "" {
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{Code: receipt.IssueCode})
	}
	return orquestagoal.NormalizeGoalWorkResultV0(result)
}

func codexGoalObservationInvalidResultV0(
	request orquestagoal.GoalObservationRequestV0,
	code string,
) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         strings.TrimSpace(request.GoalRef),
		ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
		Issues:          []orquestagoal.GoalWorkIssueV0{{Code: code}},
	}
}

func firstNonEmptyCodexGoalStringV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
