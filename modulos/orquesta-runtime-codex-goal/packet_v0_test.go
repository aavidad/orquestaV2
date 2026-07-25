package orquestaruntimecodexgoal

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion(t *testing.T) {
	packet, issues := BuildCodexGoalStartPacketV0(validCodexGoalSpecV0())
	if len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	if packet.WorkspaceRef != orquestagoal.GoalWorkspaceRefForGoalV0(packet.GoalRef) ||
		packet.ProviderRef != CodexGoalProviderRefV0 {
		t.Fatalf("workspace authority=%+v", packet)
	}
	for _, expected := range []string{
		"Director operativo interno",
		"Orquesta gobierna desde fuera",
		"request-ref-001",
		"implementation",
		"docs/estado_actual_2026-05-17.md",
		"kind=doc",
		"modulos/orquesta-goal",
		"test_ref=test-ref-goal",
		"go test -count=1 ./modulos/orquesta-goal",
		"criterio-ref-goal-first",
		"artifact-ref-goal-summary",
		"Materializa cada artefacto requerido dentro de un write-set autorizado",
		"<artifact_type>.json",
		"checkpoint temprano en su runtime ignorado por Git",
		"checkpoint_started_*",
		"No vuelques salidas gigantes de herramientas",
		"max_text_bytes=16384",
		"rg --max-count",
		"rg --files | head",
		"write_set_enforcement=workspace_write_guard",
		"sandbox minimo workspace-write",
		"El recibo terminal debe declarar artifact_paths",
		"materialized_artifacts",
		"checklist.expected_refs",
		"rework_plan_refs",
		"Evidencia requerida: evidence-ref-required",
		CodexGoalResultMarkerV0,
		CodexGoalResultSchemaV0,
		CodexGoalResultFileNameForGoalRefV0("goal-ref-001"),
		CodexGoalRuntimeReceiptRelativeDirV0("goal-ref-001") + "/" + CodexGoalResultFileNameForGoalRefV0("goal-ref-001"),
		"goal_ref",
		"schema_version",
		"status/estado debe ser terminal explicito",
		"reason_code",
		"checkpoint_started",
		"usa literalmente los test_ref",
		"usa literalmente los artifact_ref",
		"\"artifact_paths\":[]",
		"\"materialized_artifacts\"",
		"\"checklist\"",
		"lista solo rutas relativas de producto o validacion",
		"separa artefactos validos de borradores recuperables",
		"checklist.missing_refs",
		"Si escribiste un artefacto de producto fuera del write-set, no lo ocultes",
		"status blocked con summary out_of_scope_artifacts",
		"Resultado estructurado obligatorio",
		"Los command de Tests requeridos son parte del contrato neutral acotado",
		"Antes de leer ficheros completos en un write-set de codigo",
		"orquesta.codebase.query.v0",
		"POST /api/v0/codebase/query",
		"relevant_snippets",
		"codebase_broker_unavailable",
	} {
		if !strings.Contains(packet.Prompt, expected) {
			t.Fatalf("prompt no contiene %q:\n%s", expected, packet.Prompt)
		}
	}
	if packet.SchemaVersion != CodexGoalStartPacketSchemaV0 {
		t.Fatalf("schema=%q", packet.SchemaVersion)
	}
	if packet.RequestRef != "request-ref-001" ||
		packet.ProjectRef != "project-ref-orquesta" ||
		packet.WorkKind != "idle_self_improvement" ||
		packet.WorkProfileKind != "implementation" ||
		packet.TaskCostClass != CodexGoalTaskCostClassCodeV0 ||
		len(packet.AcceptanceCriteria) != 1 ||
		packet.AcceptanceCriteria[0] != "criterio-ref-goal-first" ||
		len(packet.ArtifactContracts) != 1 ||
		packet.ArtifactContracts[0].ArtifactRef != "artifact-ref-goal-summary" ||
		len(packet.EvidenceRefs) != 1 ||
		packet.EvidenceRefs[0] != "evidence-ref-input" ||
		!packet.ClosurePolicy.RequireRequiredTests ||
		!packet.ReworkPolicy.PreferNewGoal ||
		packet.Budget.MaxSubgoals != 2 ||
		len(packet.RequiredTests) != 1 ||
		!packet.DirectionContract.RequireEarlyCheckpoint ||
		packet.DirectionContract.EarlyCheckpointFile != "checkpoint_started.txt" ||
		packet.DirectionContract.ToolOutputPolicy.MaxTextBytes != CodexGoalToolOutputMaxBytesV0 ||
		!packet.DirectionContract.ToolOutputPolicy.RequireBoundedCommands ||
		!packet.DirectionContract.ToolOutputPolicy.DurableEvidenceRequired ||
		packet.DirectionContract.WriteSetEnforcement != CodexGoalWriteSetEnforcementV0 ||
		packet.DirectionContract.MinimumSandbox != CodexGoalMinimumSandboxV0 ||
		len(packet.DirectionContract.AllowedWriteSet) != 1 ||
		len(packet.DirectionContract.RequiredArtifactContracts) != 1 ||
		packet.ContextBudget.ContextBudgetTotalBytes <= 0 ||
		packet.ContextBudget.StaticPromptBytes <= 0 ||
		packet.ContextBudget.DynamicContextBytes <= 0 ||
		packet.ContextBudget.MaterializedContextBytes <= 0 ||
		packet.RequiredTests[0].Command != "go test -count=1 ./modulos/orquesta-goal" {
		t.Fatalf("packet no conserva gobierno: %+v", packet)
	}
	for _, want := range []string{"artifact_paths", "materialized_artifacts", "checklist", "evidence_refs"} {
		if !hasGoalStringForTestV0(packet.DirectionContract.RequiredTerminalFields, want) {
			t.Fatalf("direction_contract no exige %s: %+v", want, packet.DirectionContract)
		}
	}
	if !hasGoalStringForTestV0(packet.DirectionContract.ToolOutputPolicy.BoundedCommandHints, "rg --max-count") {
		t.Fatalf("direction_contract sin hints de comandos acotados: %+v", packet.DirectionContract.ToolOutputPolicy)
	}
}

func TestBuildCodexGoalStartPacketV0PropagaIntentManifestV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.IntentManifestRef = "intent-manifest-ref-request-ref-001"
	spec.IntentManifestSHA256 = strings.Repeat("a", 64)
	packet, issues := BuildCodexGoalStartPacketV0(spec)
	if len(issues) != 0 || packet.IntentManifestRef != spec.IntentManifestRef || packet.IntentManifestSHA256 != spec.IntentManifestSHA256 {
		t.Fatalf("packet=%+v issues=%+v", packet, issues)
	}
	if !strings.Contains(packet.Prompt, "intent_manifest_path: .orquesta-runtime/intent-manifests/intent-manifest-ref-request-ref-001.json") {
		t.Fatalf("prompt missing manifest path: %s", packet.Prompt)
	}
}

func TestBuildCodexGoalStartPacketV0PropagaReasoningEffortExactoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.ReasoningEffort = orquestagoal.GoalReasoningEffortXHighV0

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if packet.ReasoningEffort != orquestagoal.GoalReasoningEffortXHighV0 {
		t.Fatalf("reasoning_effort=%q", packet.ReasoningEffort)
	}
	raw, err := json.Marshal(packet)
	if err != nil || !strings.Contains(string(raw), `"reasoning_effort":"xhigh"`) {
		t.Fatalf("packet json=%s err=%v", raw, err)
	}
}

func TestBuildCodexGoalStartPacketV0ProyectaAutorizacionesDestructivasAlContratoDireccionV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.DestructiveAuthorizations = []orquestagoal.GoalDestructiveChangeAuthorizationV0{{
		Kind: orquestagoal.GoalDestructiveChangeAuthorizationRenameV0, PreviousPath: "modulos/orquesta-goal/antes.go", CurrentPath: "modulos/orquesta-goal/despues.go",
	}}
	packet, issues := BuildCodexGoalStartPacketV0(spec)
	if len(issues) != 0 || len(packet.DirectionContract.DestructiveAuthorizations) != 1 ||
		packet.DirectionContract.DestructiveAuthorizations[0] != spec.DestructiveAuthorizations[0] {
		t.Fatalf("packet=%+v issues=%+v", packet, issues)
	}
}

func TestBuildCodexGoalStartPacketV0SoloExigeAnalizadorConWriteSetCodigoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "docs"}}
	packet, issues := BuildCodexGoalStartPacketV0(spec)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if strings.Contains(packet.Prompt, "Antes de leer ficheros completos en un write-set de codigo") {
		t.Fatalf("prompt exige analizador en write-set documental:\n%s", packet.Prompt)
	}

	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}}
	packet, issues = BuildCodexGoalStartPacketV0(spec)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.Contains(packet.Prompt, "orquesta.codebase.query.v0") ||
		!strings.Contains(packet.Prompt, "POST /api/v0/codebase/query") ||
		!strings.Contains(packet.Prompt, "codebase_broker_unavailable") ||
		!strings.Contains(packet.Prompt, "module_exports") {
		t.Fatalf("prompt no exige analizador en write-set codigo:\n%s", packet.Prompt)
	}
}

func TestBuildCodexGoalStartPacketV0AnalizadorDegradaSiBrokerNoInyectadoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}}
	spec.SkillRefs = nil

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, expected := range []string{
		"si aparece en el toolbelt MCP local",
		"POST /api/v0/codebase/query",
		"Si el broker no esta disponible",
		"no bloquees el trabajo por esa ausencia",
		"`rg`/`sed` acotados",
		"codebase_broker_unavailable",
		"sin arrancar indexadores propios ni codebase-memory-mcp",
	} {
		if !strings.Contains(packet.Prompt, expected) {
			t.Fatalf("prompt no contiene fallback %q:\n%s", expected, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalStartPacketV0DerivaTaskCostClassDesdeWriteSetV0(t *testing.T) {
	tests := []struct {
		name     string
		writeSet []orquestagoal.GoalWriteScopeV0
		want     string
	}{
		{
			name: "doc si todos los scopes son markdown",
			writeSet: []orquestagoal.GoalWriteScopeV0{
				{Path: "docs/runbooks/incidencia.md"},
				{Path: "docs/bitacora_correccion_pericial_2026-07-03.md"},
			},
			want: CodexGoalTaskCostClassDocV0,
		},
		{
			name: "doc si el write-set es carpeta docs",
			writeSet: []orquestagoal.GoalWriteScopeV0{
				{Path: "docs"},
				{Path: "docs/runbooks"},
			},
			want: CodexGoalTaskCostClassDocV0,
		},
		{
			name:     "code si no hay markdown",
			writeSet: []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}},
			want:     CodexGoalTaskCostClassCodeV0,
		},
		{
			name: "mixed si conviven markdown y codigo",
			writeSet: []orquestagoal.GoalWriteScopeV0{
				{Path: "docs/runbooks/incidencia.md"},
				{Path: "cmd/orquesta-server"},
			},
			want: CodexGoalTaskCostClassMixedV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validCodexGoalSpecV0()
			spec.WriteSet = tt.writeSet

			packet, issues := BuildCodexGoalStartPacketV0(spec)

			if len(issues) != 0 {
				t.Fatalf("issues=%+v", issues)
			}
			if packet.TaskCostClass != tt.want {
				t.Fatalf("task_cost_class=%q want %q", packet.TaskCostClass, tt.want)
			}
			wantPrompt := "task_cost_class: " + tt.want
			if !strings.Contains(packet.Prompt, wantPrompt) {
				t.Fatalf("prompt no contiene %q:\n%s", wantPrompt, packet.Prompt)
			}
		})
	}
}

func TestBuildCodexGoalContextBudgetV0NoPublicaPromptCompletoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.Objective = "No serializar este objetivo completo como metrica secreta."
	spec.ContextRefs = []orquestagoal.GoalContextRefV0{{
		Kind:    "code_context",
		Ref:     "code-context-ref-goal-001",
		Purpose: "fragmento materializado acotado",
	}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	metricJSON := mustMarshalCodexGoalTestV0(t, packet.ContextBudget)
	for _, want := range []string{
		"context_budget_total_bytes",
		"static_prompt_bytes",
		"dynamic_context_bytes",
		"materialized_context_bytes",
	} {
		if !strings.Contains(metricJSON, want) {
			t.Fatalf("context budget no contiene %q: %s", want, metricJSON)
		}
	}
	if strings.Contains(metricJSON, "No serializar") ||
		strings.Contains(metricJSON, "fragmento materializado") {
		t.Fatalf("context budget publico contenido de prompt/contexto: %s", metricJSON)
	}
}

func TestBuildCodexGoalPromptV0OrdenaPrefijoEstableAntesDeDinamicoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.Objective = "Implementar objetivo dinamico CTX-TASK-801C."
	spec.RuleRefs = []orquestagoal.GoalRuleRefV0{
		{Kind: "repo", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
		{Kind: "security", Ref: "rule-ref-hard-no-secrets", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
	}
	spec.SkillRefs = []string{"skill-ref-codex-toolbelt-v0"}
	spec.ContextRefs = []orquestagoal.GoalContextRefV0{{
		Kind: "live_state",
		Ref:  "state-ref-live-001",
	}}
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	prompt := packet.Prompt
	stable := promptIndexForTestV0(t, prompt, "Prefijo estable cacheable:")
	agents := promptIndexForTestV0(t, prompt, "AGENTS.md [hard]")
	toolbelt := promptIndexForTestV0(t, prompt, "skill-ref-codex-toolbelt-v0")
	hardRule := promptIndexForTestV0(t, prompt, "rule-ref-hard-no-secrets [hard]")
	dynamic := promptIndexForTestV0(t, prompt, "Contexto dinamico del goal")
	objective := promptIndexForTestV0(t, prompt, "Objetivo:\nImplementar objetivo dinamico CTX-TASK-801C.")
	liveState := promptIndexForTestV0(t, prompt, "state-ref-live-001 kind=live_state")
	writeSet := promptIndexForTestV0(t, prompt, "Write-set autorizado:")
	if !(stable < agents && stable < toolbelt && stable < hardRule &&
		agents < dynamic && toolbelt < dynamic && hardRule < dynamic &&
		dynamic < objective && dynamic < liveState && dynamic < writeSet) {
		t.Fatalf("orden estable/dinamico inesperado: stable=%d agents=%d toolbelt=%d hard=%d dynamic=%d objective=%d live=%d write=%d\n%s",
			stable, agents, toolbelt, hardRule, dynamic, objective, liveState, writeSet, prompt)
	}
}

func TestBuildCodexGoalStartPacketV0ProjectaPromptCacheKeyEstableV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.RuleRefs = []orquestagoal.GoalRuleRefV0{{Kind: "repo", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0}}
	spec.SkillRefs = []string{"skill-ref-codex-toolbelt-v0"}

	changed := spec
	changed.GoalRef = "goal-ref-002"
	changed.Objective = "Otro objetivo dinamico."
	changed.ContextRefs = []orquestagoal.GoalContextRefV0{{Kind: "live_state", Ref: "state-ref-live-002"}}
	changed.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}}
	changed.RequiredTests = []orquestagoal.GoalRequiredTestV0{{TestRef: "test-ref-other", Command: "go test -count=1 ./cmd/orquesta-server"}}

	first, firstIssues := BuildCodexGoalStartPacketV0(spec)
	second, secondIssues := BuildCodexGoalStartPacketV0(changed)

	if len(firstIssues) != 0 || len(secondIssues) != 0 {
		t.Fatalf("issues first=%+v second=%+v", firstIssues, secondIssues)
	}
	if first.Prompt == second.Prompt {
		t.Fatalf("prompts dinamicos no cambiaron")
	}
	if first.PromptCache.CacheKey == "" ||
		first.PromptCache.StablePrefixSHA256 == "" ||
		first.PromptCache.StablePrefixBytes <= 0 ||
		first.PromptCache.DynamicSuffixBytes <= 0 {
		t.Fatalf("prompt_cache incompleto: %+v", first.PromptCache)
	}
	if first.PromptCache.CacheKey != second.PromptCache.CacheKey ||
		first.PromptCache.StablePrefixSHA256 != second.PromptCache.StablePrefixSHA256 {
		t.Fatalf("cache key debe depender del prefijo estable, first=%+v second=%+v", first.PromptCache, second.PromptCache)
	}
}

func TestCodexGoalPromptCacheProjectionV0SePropagaComoEvidenceRefV0(t *testing.T) {
	launcher := CodexGoalLauncherV0{Starter: &recordingCodexGoalStarterV0{
		receipt: CodexGoalStartReceiptV0{
			Status:               orquestagoal.GoalStatusAcceptedV0,
			ExternalGoalRef:      "external-goal-ref-cache-001",
			RuntimeGenerationRef: "generation-ref-cache-001",
			PromptCache: CodexGoalPromptCacheProjectionV0{
				CacheKey:          "provider-cache-key-hit-001",
				CachedInputTokens: 2048,
			},
		},
	}}

	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	for _, want := range []string{
		"evidence-ref-codex-goal-prompt-cache-key-provider-cache-key-hit-001",
		"evidence-ref-codex-goal-cached-input-tokens-2048",
	} {
		if !hasGoalStringForTestV0(receipt.EvidenceRefs, want) {
			t.Fatalf("receipt no proyecta %s: %+v", want, receipt)
		}
	}

	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-001",
			ExternalGoalRef: "external-goal-ref-cache-001",
			PromptCache: CodexGoalPromptCacheProjectionV0{
				CacheKey:          "provider-cache-key-observe-001",
				CachedInputTokens: 512,
			},
		},
	}}

	result, observeErr := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-cache-001",
	})

	if observeErr != nil {
		t.Fatalf("observe: %v", observeErr)
	}
	for _, want := range []string{
		"evidence-ref-codex-goal-prompt-cache-key-provider-cache-key-observe-001",
		"evidence-ref-codex-goal-cached-input-tokens-512",
	} {
		if !hasGoalStringForTestV0(result.EvidenceRefs, want) {
			t.Fatalf("result no proyecta %s: %+v", want, result)
		}
	}
}

func TestBuildCodexGoalPromptV0IncluyePropositoDeContextRefs(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.ContextRefs = append(spec.ContextRefs, orquestagoal.GoalContextRefV0{
		Kind:    "input_field_value",
		Ref:     "input-field-output_contract-value-abc123",
		Purpose: `Campo input_fields.output_contract inlineado de forma acotada: {"artifact_type":"content_block"}`,
	})

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, want := range []string{
		"input-field-output_contract-value-abc123 kind=input_field_value",
		`Campo input_fields.output_contract inlineado de forma acotada`,
		`"artifact_type":"content_block"`,
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0DomainWorkDelegaReceiptEnAdaptadorV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WorkProfileKind = "domain_work"
	spec.ClosurePolicy.RequireDomainReceipt = true

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, want := range []string{
		"En trabajos domain_work",
		"materializa el artefacto en el write-set",
		"deja submit_artifact/receipt al adaptador externo de Orquesta",
		"no bloquees por no tener conector REST/MCP",
		"domain_receipt_refs vacio",
		"No pongas public_blocker, pending, ready=false",
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0NoTrataMarkdownWriteSetComoDirectorioV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/INCIDENCIA_TEST.md"}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	for _, forbidden := range []string{
		"external/opes/INCIDENCIA_TEST.md/docs/" + CodexGoalResultFileNameV0,
		"Materializa primero el directorio del write-set",
	} {
		if strings.Contains(packet.Prompt, forbidden) {
			t.Fatalf("prompt contiene ruta/instruccion prohibida %q:\n%s", forbidden, packet.Prompt)
		}
	}
	for _, want := range []string{
		"external/opes/INCIDENCIA_TEST.md es un archivo Markdown final",
		"no crees un directorio con ese nombre",
	} {
		if !strings.Contains(packet.Prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0NoCuelgaResultadoDurableBajoFicherosCodigoOScriptV0(t *testing.T) {
	for _, path := range []string{
		"modulos/orquesta-app-codex-stack/stack_v0.go",
		"scripts/smoke_opes_lifecycle_real.sh",
	} {
		t.Run(path, func(t *testing.T) {
			spec := validCodexGoalSpecV0()
			spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: path}}

			packet, issues := BuildCodexGoalStartPacketV0(spec)

			if len(issues) != 0 {
				t.Fatalf("issues=%+v", issues)
			}
			for _, forbidden := range []string{
				path + "/docs/" + CodexGoalResultFileNameV0,
				path + "/docs/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef),
				"Materializa primero el directorio del write-set",
			} {
				if strings.Contains(packet.Prompt, forbidden) {
					t.Fatalf("prompt contiene ruta/instruccion imposible %q:\n%s", forbidden, packet.Prompt)
				}
			}
		})
	}
}

func TestBuildCodexGoalPromptV0UsaRuntimeParaResultadoDurableV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{
		{Path: "external/opes/INCIDENCIA_TEST.md"},
		{Path: "external/opes/control_audio"},
	}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if strings.Contains(packet.Prompt, "external/opes/INCIDENCIA_TEST.md/docs/"+CodexGoalResultFileNameV0) {
		t.Fatalf("prompt cuelga resultado bajo markdown:\n%s", packet.Prompt)
	}
	want := CodexGoalRuntimeReceiptRelativeDirV0(spec.GoalRef) + "/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)
	if !strings.Contains(packet.Prompt, want) ||
		strings.Contains(packet.Prompt, "external/opes/control_audio/docs/"+CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)) {
		t.Fatalf("prompt no usa runtime para resultado durable: want=%s\n%s", want, packet.Prompt)
	}
}

func TestBuildCodexGoalPromptV0NoUsaFicheroCodigoNiDirectorioParaRecibosV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{
		{Path: "modulos/orquesta-app-codex-stack/stack_v0.go"},
		{Path: "modulos/orquesta-app-codex-stack"},
	}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if strings.Contains(packet.Prompt, "modulos/orquesta-app-codex-stack/stack_v0.go/docs/"+CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)) {
		t.Fatalf("prompt cuelga resultado bajo fichero Go:\n%s", packet.Prompt)
	}
	for _, forbidden := range []string{
		"modulos/orquesta-app-codex-stack/stack_v0.go/docs/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef),
		"modulos/orquesta-app-codex-stack/docs/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef),
	} {
		if strings.Contains(packet.Prompt, forbidden) {
			t.Fatalf("prompt cuelga recibo bajo write-set: %s\n%s", forbidden, packet.Prompt)
		}
	}
}

func TestBuildCodexGoalPromptV0UsaResultadoRuntimeUnicoPorGoalRefV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.GoalRef = "goal-ref-autoprogramming-backlog-srv-task-022-a54b0a70"
	spec.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}}

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	want := CodexGoalRuntimeReceiptRelativeDirV0(spec.GoalRef) + "/" + CodexGoalResultFileNameForGoalRefV0(spec.GoalRef)
	if !strings.Contains(packet.Prompt, want) ||
		strings.Contains(packet.Prompt, "modulos/orquesta-server/docs/"+CodexGoalResultFileNameV0+" ") {
		t.Fatalf("prompt=%s want=%s", packet.Prompt, want)
	}
}

func TestCodexGoalResultFileNameForGoalRefV0NormalizaNombreV0(t *testing.T) {
	goalRef := " Goal Ref/APG 005: cierre "
	digest := sha256.Sum256([]byte(goalRef))
	want := "orquesta_goal_result_goal-ref-apg-00-" + fmt.Sprintf("%x", digest) + ".json"
	got := CodexGoalResultFileNameForGoalRefV0(goalRef)
	if got != want ||
		!CodexGoalResultFileNameLooksValidV0(got) ||
		!CodexGoalResultFileNameLooksValidV0(CodexGoalResultFileNameV0) {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestCodexGoalPhysicalNamesV0AreInjectiveAndBounded(t *testing.T) {
	tests := []struct {
		name        string
		left, right string
	}{
		{name: "case", left: "Goal-Ref-001", right: "goal-ref-001"},
		{name: "substitution", left: "goal/ref/001", right: "goal ref 001"},
		{name: "truncation", left: strings.Repeat("a", 200) + "-left", right: strings.Repeat("a", 200) + "-right"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left := codexGoalResultFileSafePartV0(test.left)
			right := codexGoalResultFileSafePartV0(test.right)
			leftDigest := sha256.Sum256([]byte(test.left))
			rightDigest := sha256.Sum256([]byte(test.right))
			if left == right || len(left) > codexGoalPhysicalNameMaxBytesV0 || len(right) > codexGoalPhysicalNameMaxBytesV0 ||
				!strings.HasSuffix(left, fmt.Sprintf("%x", leftDigest)) ||
				!strings.HasSuffix(right, fmt.Sprintf("%x", rightDigest)) {
				t.Fatalf("left=%q right=%q", left, right)
			}
			leftDir := CodexGoalRuntimeReceiptRelativeDirV0(test.left)
			rightDir := CodexGoalRuntimeReceiptRelativeDirV0(test.right)
			if leftDir == rightDir || !strings.HasSuffix(leftDir, left) || !strings.HasSuffix(rightDir, right) {
				t.Fatalf("left_dir=%q right_dir=%q", leftDir, rightDir)
			}
		})
	}
}

func TestBuildCodexGoalStartPacketV0RechazaPromptDemasiadoGrande(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.Objective = strings.Repeat("x", CodexGoalMaxPromptBytesV0)

	packet, issues := BuildCodexGoalStartPacketV0(spec)

	if len(issues) != 1 ||
		issues[0].Code != ErrCodexGoalPromptTooLargeV0 ||
		issues[0].Field != "prompt" ||
		packet.SchemaVersion != "" {
		t.Fatalf("packet=%+v issues=%+v", packet, issues)
	}
}

func TestCodexGoalLauncherV0RechazaSpecInvalido(t *testing.T) {
	launcher := CodexGoalLauncherV0{Starter: fakeCodexGoalStarterV0{}}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{})
	if err == nil {
		t.Fatal("expected error")
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexGoalLauncherV0LlamaStarterInyectado(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{receipt: CodexGoalStartReceiptV0{
		Status: orquestagoal.GoalStatusAcceptedV0, ExternalGoalRef: "external-goal-ref-001",
		RuntimeGenerationRef: "generation-ref-launch-propagation-001",
	}}
	launcher := CodexGoalLauncherV0{Starter: starter}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if !starter.called {
		t.Fatal("starter no llamado")
	}
	if receipt.Status != orquestagoal.GoalStatusAcceptedV0 {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.ExternalGoalRef != "external-goal-ref-001" {
		t.Fatalf("external=%q", receipt.ExternalGoalRef)
	}
	if receipt.RuntimeGenerationRef != "generation-ref-launch-propagation-001" {
		t.Fatalf("runtime generation=%q", receipt.RuntimeGenerationRef)
	}
	if receipt.WorkspaceAuthoritySchemaVersion != orquestagoal.GoalWorkspaceAuthoritySchemaV0 ||
		receipt.WorkspaceRef != orquestagoal.GoalWorkspaceRefForGoalV0(receipt.GoalRef) ||
		receipt.ProviderRef != CodexGoalProviderRefV0 {
		t.Fatalf("workspace authority=%+v", receipt)
	}
	if receipt.ContextBudget.ContextBudgetTotalBytes <= 0 ||
		receipt.ContextBudget.StaticPromptBytes <= 0 ||
		receipt.ContextBudget.DynamicContextBytes <= 0 {
		t.Fatalf("receipt sin context budget: %+v", receipt.ContextBudget)
	}
}

func TestCodexGoalLauncherV0FallaCerradoSiStartExitosoNoDevuelveGeneracionV0(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{receipt: CodexGoalStartReceiptV0{
		Status: orquestagoal.GoalStatusAcceptedV0, ExternalGoalRef: "external-goal-ref-missing-generation",
		EvidenceRefs: []string{"evidence-ref-start-without-generation"},
	}}
	receipt, err := (CodexGoalLauncherV0{Starter: starter}).LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())
	if err == nil || err.Error() != ErrCodexGoalRuntimeGenerationMissingV0 ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		receipt.WorkspaceAuthoritySchemaVersion != "" || receipt.WorkspaceRef != "" || receipt.ProviderRef != "" ||
		receipt.RuntimeGenerationRef != "" || !hasGoalIssueCodeForTestV0(receipt.Issues, ErrCodexGoalRuntimeGenerationMissingV0) {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestCodexGoalLauncherV0LigaManifestEnReceiptVersionadoV0(t *testing.T) {
	spec := validCodexGoalSpecV0()
	spec.RequestRef = "request-ref-codex-receipt-manifest-001"
	spec.IntentManifestRef = "intent-manifest-ref-" + spec.RequestRef
	spec.IntentManifestSHA256 = strings.Repeat("a", 64)
	starter := &recordingCodexGoalStarterV0{receipt: CodexGoalStartReceiptV0{
		Status: orquestagoal.GoalStatusAcceptedV0, ExternalGoalRef: "external-goal-ref-receipt-manifest-001",
		RuntimeGenerationRef: "generation-ref-receipt-manifest-001",
	}}
	receipt, err := (CodexGoalLauncherV0{Starter: starter}).LaunchGoalWorkV0(context.Background(), spec)
	if err != nil || receipt.WorkspaceAuthoritySchemaVersion != orquestagoal.GoalWorkspaceAuthoritySchemaV0 ||
		receipt.IntentManifestRef != spec.IntentManifestRef || receipt.IntentManifestSHA256 != spec.IntentManifestSHA256 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestCodexGoalLauncherGenerationConflictPersisteLifecycleRunningV0(t *testing.T) {
	startErr := errors.New("codex_app_server_tmux_generation_conflict")
	starter := &recordingCodexGoalStarterV0{
		receipt: CodexGoalStartReceiptV0{
			Status:               orquestagoal.GoalStatusRunningV0,
			ExternalGoalRef:      "thread-ref-generation-lifecycle-001",
			RuntimeGenerationRef: "generation-ref-lifecycle-001",
			IssueCode:            "codex_app_server_tmux_generation_conflict",
		},
		err: startErr,
	}
	store := &recordingGoalStateStoreForGenerationTestV0{}
	result, err := orquestagoal.StartGoalWorkV0(context.Background(), orquestagoal.GoalWorkStartRequestV0{
		RunRef: "run-ref-generation-lifecycle-001", Spec: validCodexGoalSpecV0(),
	}, orquestagoal.GoalWorkLifecyclePortsV0{
		Launcher: CodexGoalLauncherV0{Starter: starter}, StateStore: store,
	})
	if !errors.Is(err, startErr) || !starter.called || store.saves != 1 ||
		result.State.Status != orquestagoal.GoalStatusRunningV0 ||
		result.State.LaunchReceipt.RuntimeGenerationRef != "generation-ref-lifecycle-001" ||
		!hasGoalIssueCodeForTestV0(result.State.LaunchReceipt.Issues, "codex_app_server_tmux_generation_conflict_retryable") ||
		hasGoalIssueCodeForTestV0(result.State.LaunchReceipt.Issues, "goal_launch_partial_error") {
		t.Fatalf("result=%+v err=%v saves=%d", result, err, store.saves)
	}
}

func TestCodexGoalLauncherV0FusionaCacheStatusDelBackendV0(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{
		receipt: CodexGoalStartReceiptV0{
			Status:               orquestagoal.GoalStatusAcceptedV0,
			ExternalGoalRef:      "external-goal-ref-cache-001",
			RuntimeGenerationRef: "generation-ref-cache-status-001",
			ContextBudget: orquestagoal.GoalContextBudgetV0{
				CodeContextCacheStatus: "hit",
			},
		},
	}
	launcher := CodexGoalLauncherV0{Starter: starter}

	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.ContextBudget.ContextBudgetTotalBytes <= 0 ||
		receipt.ContextBudget.CodeContextCacheStatus != "hit" {
		t.Fatalf("context budget no fusionado: %+v", receipt.ContextBudget)
	}
}

func TestCodexGoalLauncherV0PreservaIssueCodeDeBackendV0(t *testing.T) {
	starter := &recordingCodexGoalStarterV0{
		receipt: CodexGoalStartReceiptV0{
			IssueCode:    "codex_app_server_control_socket_missing",
			EvidenceRefs: []string{"evidence-ref-codex-app-server-control-socket-missing"},
		},
		err: errors.New("backend unavailable"),
	}
	launcher := CodexGoalLauncherV0{Starter: starter}

	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

	if err == nil ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(receipt.Issues, "codex_app_server_control_socket_missing") ||
		!hasGoalStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-control-socket-missing") {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestCodexGoalLauncherV0ClasificaErrorBackendSinIssueCodeV0(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{
			name: "reset stdio",
			message: `Node.js[2796823]: void node::ResetStdio() at ../src/node.cc:751
Assertion failed: !(err != 0) || (err == -1 && (*__errno_location ()) == 1)`,
			want: "codex_app_server_wrapper_stdio_failed",
		},
		{
			name:    "tmux exited",
			message: "tmux session exited before app-server socket was ready",
			want:    "codex_app_server_tmux_session_exited",
		},
		{
			name:    "bwrap permission",
			message: "bwrap: Can't mkdir parents for /tmp/orquesta-runtime: Permission denied",
			want:    "codex_app_server_permission_denied",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			starter := &recordingCodexGoalStarterV0{
				err: errors.New(tt.message),
			}
			launcher := CodexGoalLauncherV0{Starter: starter}

			receipt, err := launcher.LaunchGoalWorkV0(context.Background(), validCodexGoalSpecV0())

			if err == nil ||
				receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
				!hasGoalIssueCodeForTestV0(receipt.Issues, tt.want) {
				t.Fatalf("receipt=%+v err=%v want_issue=%q", receipt, err, tt.want)
			}
		})
	}
}

func TestBuildCodexGoalObservationRequestV0ValidaRefs(t *testing.T) {
	packet, issues := BuildCodexGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		GoalRef:                         "goal-ref-001",
		ExternalGoalRef:                 "external-goal-ref-001",
		IntentManifestRef:               "intent-manifest-ref-goal-ref-001",
		IntentManifestSHA256:            strings.Repeat("a", 64),
		WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-001"),
		ProviderRef:                     CodexGoalProviderRefV0,
		RuntimeGenerationRef:            "generation-ref-build-observation-001",
	})
	if len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	if packet.SchemaVersion != CodexGoalObservationRequestSchemaV0 ||
		packet.GoalRef != "goal-ref-001" ||
		packet.ExternalGoalRef != "external-goal-ref-001" ||
		packet.IntentManifestRef != "intent-manifest-ref-goal-ref-001" ||
		packet.IntentManifestSHA256 != strings.Repeat("a", 64) ||
		packet.WorkspaceRef != orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-001") ||
		packet.ProviderRef != CodexGoalProviderRefV0 ||
		packet.RuntimeGenerationRef != "generation-ref-build-observation-001" {
		t.Fatalf("packet=%+v", packet)
	}

	_, issues = BuildCodexGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		ExternalGoalRef: "/tmp/external-goal",
	})
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}

	_, issues = BuildCodexGoalObservationRequestV0(orquestagoal.GoalObservationRequestV0{
		GoalRef: "goal-ref-provider-mismatch-001", WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef: orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-provider-mismatch-001"),
		ProviderRef:  "provider-ref-other", RuntimeGenerationRef: "generation-ref-provider-mismatch-001",
	})
	if !hasGoalIssueCodeForTestV0(issues, orquestagoal.ErrGoalWorkspaceProviderMismatchV0) {
		t.Fatalf("provider mismatch issues=%+v", issues)
	}
}

func TestCodexGoalObserverV0LlamaObserverInyectado(t *testing.T) {
	backend := &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         "goal-ref-001",
			ExternalGoalRef: "external-goal-ref-001",
			Summary:         "cierre observado",
			EvidenceRefs:    []string{"evidence-ref-observed"},
			ArtifactPaths:   []string{" docs/orquesta_goal_result_v0.json "},
			RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
				TestRef: "test-ref-goal",
				Status:  "passed",
			}},
		},
	}
	observer := CodexGoalObserverV0{Observer: backend}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:                         "goal-ref-001",
		ExternalGoalRef:                 "external-goal-ref-001",
		IntentManifestRef:               "intent-manifest-ref-goal-ref-001",
		IntentManifestSHA256:            strings.Repeat("a", 64),
		WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-001"),
		ProviderRef:                     CodexGoalProviderRefV0,
		RuntimeGenerationRef:            "generation-ref-observer-propagation-001",
	})

	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !backend.called ||
		backend.lastRequest.GoalRef != "goal-ref-001" ||
		backend.lastRequest.IntentManifestRef != "intent-manifest-ref-goal-ref-001" ||
		backend.lastRequest.IntentManifestSHA256 != strings.Repeat("a", 64) ||
		backend.lastRequest.RuntimeGenerationRef != "generation-ref-observer-propagation-001" ||
		result.Status != orquestagoal.GoalStatusCompleteV0 ||
		result.GoalRef != "goal-ref-001" ||
		result.ExternalGoalRef != "external-goal-ref-001" ||
		len(result.ArtifactPaths) != 1 ||
		result.ArtifactPaths[0] != "docs/orquesta_goal_result_v0.json" ||
		len(result.EvidenceRefs) != 1 {
		t.Fatalf("called=%v request=%+v result=%+v", backend.called, backend.lastRequest, result)
	}
}

func TestCodexGoalObserverV0RechazaObserverAusente(t *testing.T) {
	var observer CodexGoalObserverV0

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, ErrCodexGoalObserverMissingV0) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0RechazaGoalRefCruzado(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:  orquestagoal.GoalStatusCompleteV0,
			GoalRef: "goal-ref-otro",
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueFieldForTestV0(result.Issues, "goal_ref") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0RechazaRefsInvalidasDelBackend(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      "goal-ref-001",
			EvidenceRefs: []string{"/tmp/evidence"},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueFieldForTestV0(result.Issues, "evidence_refs") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0ConservaArtefactosMaterializadosChecklistYRework(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:  orquestagoal.GoalStatusBlockedV0,
			GoalRef: "goal-ref-001",
			MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
				ArtifactRef:  "artifact-ref-tema-001",
				Path:         "temas/tema_001.md",
				ArtifactType: "markdown",
				Status:       orquestagoal.GoalMaterializedArtifactStatusPartialV0,
				EvidenceRefs: []string{"evidence-ref-artifact-partial"},
			}},
			Checklist: orquestagoal.GoalWorkChecklistV0{
				ExpectedRefs:  []string{"check-ref-texto-publicable"},
				MissingRefs:   []string{"check-ref-texto-publicable"},
				EvidenceRefs:  []string{"evidence-ref-checklist"},
				CompletedRefs: []string{"check-ref-fuentes"},
			},
			ReworkPlanRefs: []string{"rework-plan-ref-tema-001"},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err != nil ||
		result.Status != orquestagoal.GoalStatusBlockedV0 ||
		len(result.MaterializedArtifacts) != 1 ||
		result.MaterializedArtifacts[0].Status != orquestagoal.GoalMaterializedArtifactStatusPartialV0 ||
		len(result.Checklist.MissingRefs) != 1 ||
		len(result.ReworkPlanRefs) != 1 ||
		result.ReworkPlanRefs[0] != "rework-plan-ref-tema-001" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func validCodexGoalSpecV0() orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		GoalRef:            "goal-ref-001",
		RequestRef:         "request-ref-001",
		ProjectRef:         "project-ref-orquesta",
		WorkKind:           "idle_self_improvement",
		WorkProfileKind:    "implementation",
		Objective:          "Programar el contrato goal-first.",
		DirectorKind:       orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs:        []orquestagoal.GoalContextRefV0{{Kind: "doc", Ref: "docs/estado_actual_2026-05-17.md", Required: true}},
		RuleRefs:           []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0}},
		WriteSet:           []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}},
		RequiredTests:      []orquestagoal.GoalRequiredTestV0{{TestRef: "test-ref-goal", Command: "go test -count=1 ./modulos/orquesta-goal"}},
		AcceptanceCriteria: []string{"criterio-ref-goal-first"},
		ArtifactContracts:  []orquestagoal.GoalArtifactContractV0{{ArtifactRef: "artifact-ref-goal-summary", ArtifactType: "summary", Required: true}},
		EvidenceRefs:       []string{"evidence-ref-input"},
		Budget:             orquestagoal.GoalBudgetV0{MaxSubgoals: 2},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests:                 true,
			RequireArtifactPaths:                 true,
			RequireMaterializedArtifacts:         true,
			RequireChecklist:                     true,
			RequireReworkPlanForPartialArtifacts: true,
			RequiredEvidenceRefs:                 []string{"evidence-ref-required"},
		},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, PreserveArtifacts: true},
	}
}

type fakeCodexGoalStarterV0 struct{}

func (fakeCodexGoalStarterV0) StartCodexGoalV0(context.Context, CodexGoalStartPacketV0) (CodexGoalStartReceiptV0, error) {
	return CodexGoalStartReceiptV0{}, nil
}

type recordingCodexGoalStarterV0 struct {
	called  bool
	receipt CodexGoalStartReceiptV0
	err     error
}

type recordingGoalStateStoreForGenerationTestV0 struct {
	state orquestagoal.GoalWorkStateV0
	saves int
}

func (store *recordingGoalStateStoreForGenerationTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.state = normalized
	store.saves++
	return nil
}

func (store *recordingGoalStateStoreForGenerationTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	if store.state.RunRef != strings.TrimSpace(runRef) {
		return orquestagoal.GoalWorkStateV0{}, errors.New("goal_state_not_found")
	}
	return store.state, nil
}

func (starter *recordingCodexGoalStarterV0) StartCodexGoalV0(context.Context, CodexGoalStartPacketV0) (CodexGoalStartReceiptV0, error) {
	starter.called = true
	if starter.err != nil {
		return starter.receipt, starter.err
	}
	if starter.receipt.Status != "" || starter.receipt.IssueCode != "" {
		return starter.receipt, nil
	}
	return CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusAcceptedV0,
		ExternalGoalRef: "external-goal-ref-001",
		EvidenceRefs:    []string{"evidence-ref-started"},
	}, nil
}

type recordingCodexGoalObserverV0 struct {
	called      bool
	lastRequest CodexGoalObservationRequestV0
	receipt     CodexGoalObservationReceiptV0
	err         error
}

func (observer *recordingCodexGoalObserverV0) ObserveCodexGoalV0(
	_ context.Context,
	request CodexGoalObservationRequestV0,
) (CodexGoalObservationReceiptV0, error) {
	observer.called = true
	observer.lastRequest = request
	if observer.err != nil {
		return observer.receipt, observer.err
	}
	return observer.receipt, nil
}

func TestCodexGoalObserverV0PropagaErrorBackend(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("backend rejected"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: "goal-ref-001"})

	if err == nil || result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, ErrCodexGoalObservationRejectedV0) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0GenerationConflictExactoEsRetryableSinIssuesV0(t *testing.T) {
	backend := &recordingCodexGoalObserverV0{
		err:     errors.New("generation rotated"),
		receipt: CodexGoalObservationReceiptV0{IssueCode: "codex_app_server_tmux_generation_conflict"},
	}
	result, err := (CodexGoalObserverV0{Observer: backend}).ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: "goal-ref-generation-conflict-001", ExternalGoalRef: "thread-ref-generation-conflict-001",
		WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-generation-conflict-001"),
		ProviderRef:                     CodexGoalProviderRefV0,
		RuntimeGenerationRef:            "generation-ref-generation-conflict-001",
	})
	if err == nil || result.Status != orquestagoal.GoalStatusRunningV0 || len(result.Issues) != 0 || result.ExternalGoalRef != "thread-ref-generation-conflict-001" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0PreservaIssueCodeDeBackendV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("backend unavailable"),
		receipt: CodexGoalObservationReceiptV0{
			IssueCode:       "codex_app_server_control_socket_missing",
			ExternalGoalRef: "external-goal-ref-001",
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		result.ExternalGoalRef != "external-goal-ref-001" ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_control_socket_missing") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0CheckpointStartedNoDerivaIssueDesdeSummaryV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-checkpoint-001",
			ExternalGoalRef: "external-goal-ref-checkpoint-001",
			IssueCode:       "checkpoint_started",
			Summary:         "checkpoint_started; implementacion pendiente",
			ArtifactPaths:   []string{"generated-apps/checkpoint_started.txt"},
			EvidenceRefs:    []string{"evidence-ref-codex-app-server-checkpoint-started"},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-checkpoint-001",
		ExternalGoalRef: "external-goal-ref-checkpoint-001",
	})

	if err != nil ||
		result.Status != orquestagoal.GoalStatusRunningV0 ||
		result.Summary != "checkpoint_started; implementacion pendiente" ||
		!hasGoalIssueCodeForTestV0(result.Issues, "checkpoint_started") ||
		hasGoalIssueCodeForTestV0(result.Issues, "checkpoint_started; implementacion pendiente") ||
		hasGoalIssueCodeForTestV0(result.Issues, "started") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0ClasificaErrorBackendSinIssueCodeV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("codex app-server failed: Operation not permitted (os error 1)"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_operation_not_permitted") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCodexGoalObserverV0ClasificaQuotaFilesystemBackendSinIssueCodeV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		err: errors.New("rollout writer failed: Quota exceeded (os error 122)"),
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         "goal-ref-001",
		ExternalGoalRef: "external-goal-ref-001",
	})

	if err == nil ||
		result.Status != orquestagoal.GoalStatusInvalidV0 ||
		!hasGoalIssueCodeForTestV0(result.Issues, "codex_app_server_storage_quota_exceeded") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func hasGoalIssueCodeForTestV0(issues []orquestagoal.GoalWorkIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hasGoalStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func promptIndexForTestV0(t *testing.T, prompt string, needle string) int {
	t.Helper()
	index := strings.Index(prompt, needle)
	if index < 0 {
		t.Fatalf("prompt no contiene %q:\n%s", needle, prompt)
	}
	return index
}

func hasGoalIssueFieldForTestV0(issues []orquestagoal.GoalWorkIssueV0, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}

func mustMarshalCodexGoalTestV0(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(raw)
}
