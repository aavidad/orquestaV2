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
	CodexGoalResultSchemaV0             = "orquesta_goal_result.v0"
	CodexGoalResultMarkerV0             = "ORQUESTA_GOAL_RESULT_V0"
	CodexGoalResultFileNameV0           = "orquesta_goal_result_v0.json"
	CodexGoalResultFilePrefixV0         = "orquesta_goal_result_"
	CodexGoalMaxPromptBytesV0           = 32 * 1024
	CodexGoalMaxStartPacketBytesV0      = 64 * 1024
	CodexGoalToolOutputMaxBytesV0       = 16 * 1024

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
	DirectionContract  CodexGoalDirectionContractV0          `json:"direction_contract,omitempty"`
}

type CodexGoalDirectionContractV0 struct {
	RequireEarlyCheckpoint    bool                                  `json:"require_early_checkpoint,omitempty"`
	EarlyCheckpointFile       string                                `json:"early_checkpoint_file,omitempty"`
	ToolOutputPolicy          CodexGoalToolOutputPolicyV0           `json:"tool_output_policy,omitempty"`
	RequiredTerminalFields    []string                              `json:"required_terminal_fields,omitempty"`
	AllowedWriteSet           []orquestagoal.GoalWriteScopeV0       `json:"allowed_write_set,omitempty"`
	RequiredArtifactContracts []orquestagoal.GoalArtifactContractV0 `json:"required_artifact_contracts,omitempty"`
}

type CodexGoalToolOutputPolicyV0 struct {
	MaxTextBytes            int      `json:"max_text_bytes,omitempty"`
	RequireBoundedCommands  bool     `json:"require_bounded_commands,omitempty"`
	BoundedCommandHints     []string `json:"bounded_command_hints,omitempty"`
	DurableEvidenceRequired bool     `json:"durable_evidence_required,omitempty"`
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
	Status                string                                    `json:"status"`
	GoalRef               string                                    `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                                    `json:"external_goal_ref,omitempty"`
	Summary               string                                    `json:"summary,omitempty"`
	ArtifactRefs          []string                                  `json:"artifact_refs,omitempty"`
	ArtifactPaths         []string                                  `json:"artifact_paths,omitempty"`
	MaterializedArtifacts []orquestagoal.GoalMaterializedArtifactV0 `json:"materialized_artifacts,omitempty"`
	Checklist             orquestagoal.GoalWorkChecklistV0          `json:"checklist,omitempty"`
	RequiredTestResults   []orquestagoal.GoalRequiredTestResultV0   `json:"required_test_results,omitempty"`
	DomainReceiptRefs     []string                                  `json:"domain_receipt_refs,omitempty"`
	ReworkPlanRefs        []string                                  `json:"rework_plan_refs,omitempty"`
	EvidenceRefs          []string                                  `json:"evidence_refs,omitempty"`
	IssueCode             string                                    `json:"issue_code,omitempty"`
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
		DirectionContract:  codexGoalDirectionContractV0(spec),
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

func codexGoalDirectionContractV0(spec orquestagoal.GoalWorkSpecV0) CodexGoalDirectionContractV0 {
	return CodexGoalDirectionContractV0{
		RequireEarlyCheckpoint: true,
		EarlyCheckpointFile:    "checkpoint_started.txt",
		ToolOutputPolicy: CodexGoalToolOutputPolicyV0{
			MaxTextBytes:            CodexGoalToolOutputMaxBytesV0,
			RequireBoundedCommands:  true,
			BoundedCommandHints:     []string{"head", "tail", "sed -n", "rg --max-count", "rg --files | head"},
			DurableEvidenceRequired: true,
		},
		RequiredTerminalFields: []string{
			"goal_ref",
			"schema_version",
			"status",
			"artifact_paths",
			"materialized_artifacts",
			"checklist",
			"evidence_refs",
		},
		AllowedWriteSet:           append([]orquestagoal.GoalWriteScopeV0(nil), spec.WriteSet...),
		RequiredArtifactContracts: append([]orquestagoal.GoalArtifactContractV0(nil), spec.ArtifactContracts...),
	}
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
		b.WriteString("- Antes de exploracion larga o comandos costosos, materializa un checkpoint temprano dentro del write-set autorizado, por ejemplo checkpoint_started.txt, con objetivo, alcance, siguiente artefacto y evidencia compacta; declaralo despues en artifact_paths, materialized_artifacts y evidence_refs.\n")
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
	if spec.ClosurePolicy.RequireArtifactPaths {
		b.WriteString("- El recibo terminal debe declarar artifact_paths con rutas relativas dentro del write-set.\n")
	}
	if spec.ClosurePolicy.RequireMaterializedArtifacts {
		b.WriteString("- El recibo terminal debe declarar materialized_artifacts: cada fichero/artefacto producido, tipo, path relativo, status valid/invalid/partial/non_publishable y evidence_refs.\n")
	}
	if spec.ClosurePolicy.RequireChecklist {
		b.WriteString("- El recibo terminal debe declarar checklist.expected_refs, checklist.completed_refs, checklist.missing_refs y evidencias de QA/checkpoint.\n")
	}
	if spec.ClosurePolicy.RequireReworkPlanForPartialArtifacts {
		b.WriteString("- Si algun artefacto queda partial, invalid o non_publishable, no cierres complete: devuelve blocked y rework_plan_refs con el plan/fase concreta de reparacion.\n")
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
	b.WriteString("- No vuelques salidas gigantes de herramientas al chat/log: usa comandos acotados como head, tail, sed -n o rg con limites, guarda evidencia durable en el write-set si hace falta y resume refs compactas.\n")
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
	b.WriteString("\",\"schema_version\":\"")
	b.WriteString(CodexGoalResultSchemaV0)
	b.WriteString("\",\"status\":\"complete\",\"summary\":\"...\",\"artifact_refs\":[],\"artifact_paths\":[],\"materialized_artifacts\":[{\"artifact_ref\":\"...\",\"path\":\"...\",\"artifact_type\":\"...\",\"status\":\"valid\",\"evidence_refs\":[],\"issues\":[]}],\"checklist\":{\"expected_refs\":[],\"completed_refs\":[],\"missing_refs\":[],\"evidence_refs\":[]},\"required_test_results\":[{\"test_ref\":\"...\",\"status\":\"passed\",\"evidence_refs\":[]}],\"domain_receipt_refs\":[],\"rework_plan_refs\":[],\"evidence_refs\":[]}.\n")
	b.WriteString("- schema_version es obligatorio y status/estado debe ser terminal explicito: complete si entregas cierre verificable, blocked si hay bloqueo externo o invalid si el resultado no es usable.\n")
	if spec.ClosurePolicy.RequireDomainReceipt && strings.TrimSpace(spec.WorkProfileKind) == "domain_work" {
		b.WriteString("- En domain_work deja domain_receipt_refs vacio salvo que el contrato te haya dado un receipt real; Orquesta lo derivara del ledger despues de submit_artifact.\n")
	}
	if len(spec.RequiredTests) > 0 {
		b.WriteString("- En required_test_results usa literalmente los test_ref declarados arriba; no los resumas ni inventes aliases.\n")
	}
	if len(spec.ArtifactContracts) > 0 {
		b.WriteString("- En artifact_refs usa literalmente los artifact_ref declarados arriba para los artefactos producidos; no uses rutas de fichero como refs.\n")
	}
	if len(spec.WriteSet) > 0 {
		b.WriteString("- En artifact_paths lista todas las rutas relativas de ficheros creados, modificados o verificados para el cierre, incluido el JSON durable de resultado; si escribiste algo fuera del write-set, no lo ocultes: declaralo en artifact_paths y marca status blocked con summary out_of_scope_artifacts.\n")
	}
	b.WriteString("- En materialized_artifacts separa artefactos validos de borradores recuperables: usa status partial/invalid/non_publishable con issues si falta QA, calidad publicable, cobertura, schema o validacion.\n")
	b.WriteString("- En checklist.missing_refs declara lo que falta para completar el contrato; si hay faltantes o materialized_artifacts no validos, devuelve status blocked y rework_plan_refs accionables.\n")
	if resultFilePath := codexGoalResultFilePathV0(spec); resultFilePath != "" {
		b.WriteString("- Antes de marcar el goal como complete, escribe el mismo JSON en ")
		b.WriteString(resultFilePath)
		b.WriteString(" incluyendo \"goal_ref\":\"")
		b.WriteString(spec.GoalRef)
		b.WriteString("\", \"schema_version\":\"")
		b.WriteString(CodexGoalResultSchemaV0)
		b.WriteString("\" y \"status\":\"complete\" para que Orquesta pueda cerrar aunque no haya respuesta final textual.\n")
		b.WriteString("- Materializa primero el directorio del write-set y este archivo durable de resultado; si luego corriges artefactos o tests, actualiza el JSON antes de cerrar.\n")
	}
	b.WriteString("- Incluye en evidence_refs las evidencias requeridas solo si han sido verificadas; no inventes refs para forzar el cierre.\n")
	b.WriteString("- Incluye en artifact_refs solo artefactos producidos o verificados que cumplan el contrato.\n")
	return b.String()
}

func codexGoalResultFilePathV0(spec orquestagoal.GoalWorkSpecV0) string {
	resultFileName := CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)
	for _, scope := range spec.WriteSet {
		path := strings.Trim(strings.TrimSpace(scope.Path), "/")
		if path == "" {
			continue
		}
		if codexGoalWriteScopeIsMarkdownFileV0(path) {
			continue
		}
		return path + "/docs/" + resultFileName
	}
	return ""
}

func CodexGoalResultFileNameForGoalRefV0(goalRef string) string {
	part := codexGoalResultFileSafePartV0(goalRef)
	if part == "" {
		return CodexGoalResultFileNameV0
	}
	return CodexGoalResultFilePrefixV0 + part + ".json"
}

func CodexGoalResultFileNameLooksValidV0(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == CodexGoalResultFileNameV0 {
		return true
	}
	return strings.HasPrefix(name, CodexGoalResultFilePrefixV0) &&
		strings.HasSuffix(name, ".json") &&
		len(strings.TrimSuffix(strings.TrimPrefix(name, CodexGoalResultFilePrefixV0), ".json")) > 0
}

func codexGoalResultFileSafePartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		keep := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if keep {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '-' || r == '_' {
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	return out
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
		if code := codexGoalBackendIssueCodeV0(receipt.IssueCode, err); code != "" {
			result.Issues = []orquestagoal.GoalWorkIssueV0{{Code: code}}
		}
		if strings.TrimSpace(receipt.ExternalGoalRef) != "" {
			result.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
		}
		result.EvidenceRefs = append([]string(nil), receipt.EvidenceRefs...)
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
		if code := codexGoalBackendIssueCodeV0(receipt.IssueCode, err); code != "" {
			result.Issues = []orquestagoal.GoalWorkIssueV0{{Code: code}}
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
		SchemaVersion:         orquestagoal.GoalWorkResultSchemaV0,
		Status:                status,
		GoalRef:               firstNonEmptyCodexGoalStringV0(receipt.GoalRef, request.GoalRef),
		ExternalGoalRef:       firstNonEmptyCodexGoalStringV0(receipt.ExternalGoalRef, request.ExternalGoalRef),
		Summary:               strings.TrimSpace(receipt.Summary),
		ArtifactRefs:          append([]string(nil), receipt.ArtifactRefs...),
		ArtifactPaths:         append([]string(nil), receipt.ArtifactPaths...),
		MaterializedArtifacts: append([]orquestagoal.GoalMaterializedArtifactV0(nil), receipt.MaterializedArtifacts...),
		Checklist:             receipt.Checklist,
		RequiredTestResults:   append([]orquestagoal.GoalRequiredTestResultV0(nil), receipt.RequiredTestResults...),
		DomainReceiptRefs:     append([]string(nil), receipt.DomainReceiptRefs...),
		ReworkPlanRefs:        append([]string(nil), receipt.ReworkPlanRefs...),
		EvidenceRefs:          append([]string(nil), receipt.EvidenceRefs...),
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

func codexGoalBackendIssueCodeV0(explicit string, err error) string {
	if code := strings.TrimSpace(explicit); code != "" {
		return code
	}
	if err == nil {
		return ""
	}
	return codexGoalBackendIssueCodeFromMessageV0(err.Error())
}

func codexGoalBackendIssueCodeFromMessageV0(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "codex_app_server_wrapper_stdio_failed") ||
		strings.Contains(normalized, "resetstdio") ||
		strings.Contains(normalized, "node.cc:751"):
		return "codex_app_server_wrapper_stdio_failed"
	case strings.Contains(normalized, "codex_app_server_tmux_session_exited") ||
		(strings.Contains(normalized, "tmux") &&
			(strings.Contains(normalized, "session exited") ||
				strings.Contains(normalized, "server exited") ||
				strings.Contains(normalized, "exited"))):
		return "codex_app_server_tmux_session_exited"
	case strings.Contains(normalized, "codex_app_server_provider_unauthorized") ||
		strings.Contains(normalized, "401 unauthorized") ||
		(strings.Contains(normalized, "unauthorized") &&
			(strings.Contains(normalized, "api.openai.com/v1/responses") ||
				strings.Contains(normalized, "responses_websocket"))):
		return "codex_app_server_provider_unauthorized"
	case strings.Contains(normalized, "codex_app_server_auth_missing"):
		return "codex_app_server_auth_missing"
	case strings.Contains(normalized, "codex_app_server_goal_provider_limited") ||
		strings.Contains(normalized, "usagelimited") ||
		strings.Contains(normalized, "usage limited") ||
		strings.Contains(normalized, "usage limit") ||
		strings.Contains(normalized, "quota exhausted") ||
		strings.Contains(normalized, "quota limited") ||
		strings.Contains(normalized, "provider limited") ||
		strings.Contains(normalized, `"has_credits":false`) ||
		strings.Contains(normalized, `"has_credits": false`):
		return "codex_app_server_goal_provider_limited"
	case strings.Contains(normalized, "codex_app_server_storage_quota_exceeded") ||
		strings.Contains(normalized, "quota exceeded (os error 122)") ||
		strings.Contains(normalized, "disk quota exceeded") ||
		strings.Contains(normalized, "no space left on device") ||
		strings.Contains(normalized, "enospc"):
		return "codex_app_server_storage_quota_exceeded"
	case strings.Contains(normalized, "codex_app_server_goal_budget_limited") ||
		strings.Contains(normalized, "budgetlimited") ||
		strings.Contains(normalized, "budget limited") ||
		strings.Contains(normalized, "budget limit"):
		return "codex_app_server_goal_budget_limited"
	case strings.Contains(normalized, "codex_app_server_goal_policy_limited") ||
		strings.Contains(normalized, "policylimited") ||
		strings.Contains(normalized, "policy limited") ||
		strings.Contains(normalized, "policy limit"):
		return "codex_app_server_goal_policy_limited"
	case strings.Contains(normalized, "codex_app_server_control_socket_missing") ||
		strings.Contains(normalized, "failed to connect to socket") ||
		strings.Contains(normalized, "app-server-control.sock"):
		return "codex_app_server_control_socket_missing"
	case strings.Contains(normalized, "codex_app_server_standalone_missing") ||
		strings.Contains(normalized, "managed standalone codex install not found"):
		return "codex_app_server_standalone_missing"
	case strings.Contains(normalized, "codex_app_server_command_missing") ||
		strings.Contains(normalized, "executable file not found"):
		return "codex_app_server_command_missing"
	case strings.Contains(normalized, "codex_app_server_permission_denied") ||
		strings.Contains(normalized, "permission denied"):
		return "codex_app_server_permission_denied"
	case strings.Contains(normalized, "codex_app_server_operation_not_permitted") ||
		strings.Contains(normalized, "operation not permitted"):
		return "codex_app_server_operation_not_permitted"
	}
	return ""
}
