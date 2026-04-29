/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/rpclocal"
)

func TestPrepareCLISinResumeNativoDevuelvePromptContinuidad(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "codex1",
		ProyectoSlug: "autofirmav2",
		ProyectoRuta: "/tmp/autofirmav2",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			ArgsJSON:     `["chat"]`,
			EnvJSON:      `{"A":"1"}`,
			MetadataJSON: `{"familia":"openai","reanudable":true}`,
		},
		Resume: ResumeContext{
			ExternalSessionID:  "sess-1",
			ResumenContinuidad: "seguir con la firma",
			Branch:             "feature/x",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %s", plan.Modo)
	}
	if plan.NativeResume {
		t.Fatalf("no debía activar resume nativo")
	}
	if !strings.Contains(plan.ContinuityPrompt, "seguir con la firma") {
		t.Fatalf("prompt de continuidad inesperado: %s", plan.ContinuityPrompt)
	}
}

func TestPrepareCLICodexAsignaPerfilOperativoPersistentePorDefecto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.PerfilOperativo != "persistente" {
		t.Fatalf("perfil_operativo inesperado: %+v", plan)
	}
}

func TestEsConectorFamiliaOllamaReconocePoolLocal(t *testing.T) {
	if !EsConectorFamiliaOllama("ollama_pool_local", "") {
		t.Fatal("ollama_pool_local deberia pertenecer a la familia ollama")
	}
	if !EsConectorFamiliaOllama("", "ollama") {
		t.Fatal("comando ollama deberia pertenecer a la familia ollama")
	}
	if EsConectorFamiliaOllama("codex-cli", "codex") {
		t.Fatal("codex-cli no deberia pertenecer a la familia ollama")
	}
}

func TestPreparePoolLocalOllamaGeneraPlanRemotoAPI(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemma1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gemma4:26b",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:         "ollama_pool_local",
			Transporte:   "api",
			Comando:      "ollama",
			MetadataJSON: `{"endpoint":"http://127.0.0.1:18081","launch_path":"/api/runtime/ollama-pool/launch","input_path":"/api/runtime/ollama-pool/input","status_path":"/api/runtime/ollama-pool/status","stop_path":"/api/runtime/ollama-pool/stop","default_task_profile":"implementacion","default_reasoning_effort":"high"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare pool local ollama: %v", err)
	}
	if plan.Transporte != "api" || plan.Driver != "api" {
		t.Fatalf("transporte/driver inesperado: %+v", plan)
	}
	if plan.Modo != "launch" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.RemoteConfigJSON == "" {
		t.Fatalf("faltaba remote_config_json: %+v", plan)
	}
	if !strings.Contains(plan.RemoteConfigJSON, `"endpoint":"http://127.0.0.1:18081"`) {
		t.Fatalf("remote_config_json inesperado: %s", plan.RemoteConfigJSON)
	}
	if plan.PerfilTarea != "implementacion" || plan.Razonamiento != "high" {
		t.Fatalf("perfil/razonamiento inesperados: %+v", plan)
	}
}

func TestPreparePoolLocalOllamaRespetaWorkingDirDeResume(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemma1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "ollama_pool_local",
			Transporte:   "api",
			Comando:      "ollama",
			MetadataJSON: `{"endpoint":"http://127.0.0.1:18081","launch_path":"/api/runtime/ollama-pool/launch","input_path":"/api/runtime/ollama-pool/input","status_path":"/api/runtime/ollama-pool/status","stop_path":"/api/runtime/ollama-pool/stop"}`,
		},
		Resume: ResumeContext{
			CWD:    "/tmp/orquestador/.orquesta-worktrees/orquestador-gemma1",
			Branch: "orq-orquestador-gemma1",
		},
	})
	if err != nil {
		t.Fatalf("prepare pool local ollama con resume: %v", err)
	}
	if plan.WorkingDir != "/tmp/orquestador/.orquesta-worktrees/orquestador-gemma1" {
		t.Fatalf("working dir inesperado: %+v", plan)
	}
	if plan.WorktreePath != "/tmp/orquestador/.orquesta-worktrees/orquestador-gemma1" {
		t.Fatalf("worktree path inesperado: %+v", plan)
	}
}

func TestPreparePoolLocalOllamaUsaRutasCanonicasPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_SERVER_URL", "http://127.0.0.1:17669")
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemma1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gemma4:26b",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:         "ollama_pool_local",
			Transporte:   "api",
			Comando:      "ollama",
			MetadataJSON: `{"pool_compartido":true,"default_task_profile":"implementacion","default_reasoning_effort":"high"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare pool local ollama por defecto: %v", err)
	}
	if !strings.Contains(plan.RemoteConfigJSON, `"endpoint":"http://127.0.0.1:17669"`) {
		t.Fatalf("endpoint inesperado: %s", plan.RemoteConfigJSON)
	}
	for _, expected := range []string{
		`"launch_path":"/api/runtime/ollama-pool/launch"`,
		`"input_path":"/api/runtime/ollama-pool/input"`,
		`"status_path":"/api/runtime/ollama-pool/status"`,
		`"stop_path":"/api/runtime/ollama-pool/stop"`,
	} {
		if !strings.Contains(plan.RemoteConfigJSON, expected) {
			t.Fatalf("remote_config_json sin %s: %s", expected, plan.RemoteConfigJSON)
		}
	}
}

func TestPrepareCLISinResumeNativoCompactaResumePayloadEnPrompt(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir con control plane",
			ResumePayloadJSON: `{
				"bootstrap_prompt":"Bootstrap de Orquesta para Codex2.",
				"checkpoint":{"id":9,"kind":"stop"},
				"mailbox":[
					{"id":1,"kind":"instruction","payload":{"texto":"revisa el gate y sigue"}},
					{"id":2}
				],
				"project_context":{"proyecto":{"slug":"orquestador"}},
				"governance_catalog":{"hash":"abc123"},
				"persist":"ok"
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.Contains(plan.ContinuityPrompt, "Bootstrap de Orquesta para Codex2.") || strings.Contains(plan.ContinuityPrompt, `"checkpoint"`) {
		t.Fatalf("el prompt de continuidad no deberia incrustar el JSON bruto: %s", plan.ContinuityPrompt)
	}
	for _, token := range []string{"checkpoint#9(stop)", "mailbox=2", "project_context", "governance_catalog", "persist"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta resumen de resume payload %s en: %s", token, plan.ContinuityPrompt)
		}
	}
	if !strings.Contains(plan.ContinuityPrompt, "[instruction") {
		t.Fatalf("falta resumen compacto de mailbox en continuity prompt: %s", plan.ContinuityPrompt)
	}
}

func TestPrepareCLISinResumeNativoCompactaArtifactsVersionadosEnPrompt(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir con control plane",
			ResumePayloadJSON: `{
				"artifacts":[
					{"artifact_id":"patch-1","scope":"task","kind":"patch","version":3,"path":"artifacts/patches/patch-1.diff"},
					{"artifact_id":"tests-1","scope":"task","kind":"test_output","version":2,"blob_ref":"blob://tests-1"},
					{"artifact_id":"tx-1","scope":"session","kind":"transcript","version":1,"path":"artifacts/transcripts/tx-1.txt"}
				]
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for _, token := range []string{"artifacts=3", "patch@v3", "test_output@v2", "transcript@v1"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta resumen de artifacts %s en: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLICodexCompactaContinuityPromptParaRuntimeNoInteractivo(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "texto largo que no deberia pasar completo al prompt de lanzamiento",
			Branch:             "orq-orquestador-codex7",
			ResumePayloadJSON:  `{"checkpoint":{"id":33155,"kind":"stop"},"mailbox":[{"id":1,"kind":"nudge","payload":{"accion":"continuar_trabajo","tarea_id":411,"texto":"toma tarea asignada y sigue"}}],"project_context":{},"governance_catalog":{}}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.Contains(plan.ContinuityPrompt, "texto largo que no deberia pasar completo") {
		t.Fatalf("continuity prompt demasiado verboso: %s", plan.ContinuityPrompt)
	}
	for _, token := range []string{"Retoma el trabajo actual desde Orquesta.", "Proyecto: orquestador.", "Rama: orq-orquestador-codex7.", "Activa $caveman. Salida minima: hecho, tests, riesgos/bloqueos; nada mas.", "checkpoint#33155", "mailbox=1", "accion=continuar_trabajo", "tarea#411", "toma tarea asignada y sigue"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt: %s", token, plan.ContinuityPrompt)
		}
	}
	for _, token := range []string{
		"Ignora cualquier conversación vieja que no coincida con la tarea activa o el mailbox actual.",
		"No abras frentes nuevos ni reescribas código fuera del alcance inmediato.",
		"No hagas broad scans ni releas contexto global si mailbox/tarea ya acotan el slice.",
	} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt endurecido: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLICodexProvisionaPerfilCompartido(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USE_PROFILE_WRAPPER", "true")
	t.Setenv("CODEX_MULTI_BASE", "/tmp/codex-perfiles")

	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare codex provisioned profile: %v", err)
	}
	if got, want := plan.Comando, filepath.Clean("/tmp/codex-perfiles/bin/codex-perfil"); got != want {
		t.Fatalf("wrapper codex inesperado: got=%q want=%q", got, want)
	}
	if len(plan.Args) == 0 || plan.Args[0] != "Codex2" {
		t.Fatalf("args codex sin perfil canonico: %+v", plan.Args)
	}
	if plan.ProvisionedProfile == nil {
		t.Fatalf("faltaba provisioned profile: %+v", plan)
	}
	if got, want := plan.ProvisionedProfile.Strategy, "shared_clone"; got != want {
		t.Fatalf("strategy inesperada: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["HOME"], filepath.Clean("/tmp/codex-perfiles/homes/Codex2"); got != want {
		t.Fatalf("HOME provisionado inesperado: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["XDG_CONFIG_HOME"], filepath.Clean("/tmp/codex-perfiles/homes/Codex2/.config"); got != want {
		t.Fatalf("XDG_CONFIG_HOME inesperado: got=%q want=%q", got, want)
	}
}

func TestPrepareCLIClaudeProvisionaPerfilCompartido(t *testing.T) {
	t.Setenv("ORQUESTA_ANTHROPIC_PROFILES_BASE", "/tmp/claude-perfiles")

	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Claude1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "claude-code",
			Transporte:   "cli",
			Comando:      "claude-code",
			MetadataJSON: `{"familia":"anthropic","profile_strategy":"shared_clone"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare claude provisioned profile: %v", err)
	}
	if plan.ProvisionedProfile == nil {
		t.Fatalf("faltaba provisioned profile: %+v", plan)
	}
	if got, want := plan.ProvisionedProfile.Family, "anthropic"; got != want {
		t.Fatalf("family inesperada: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["HOME"], filepath.Clean("/tmp/claude-perfiles/homes/Claude1"); got != want {
		t.Fatalf("HOME provisionado inesperado: got=%q want=%q", got, want)
	}
	if got := plan.Env["ORQUESTA_PROFILE_STRATEGY"]; got != "shared_clone" {
		t.Fatalf("strategy env inesperada: %q", got)
	}
}

func TestPrepareCLIGeminiProvisionaPerfilDedicadoConTemplates(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemini1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "gemini-cli",
			Transporte: "cli",
			Comando:    "gemini",
			EnvJSON:    `{"GEMINI_HOME":"{{profile_home}}","GEMINI_PROFILE":"{{profile_name}}"}`,
			MetadataJSON: `{
				"familia":"google",
				"profile_strategy":"dedicated_profile",
				"profiles_base_dir":"/srv/orquesta/gemini"
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare gemini provisioned profile: %v", err)
	}
	if plan.ProvisionedProfile == nil {
		t.Fatalf("faltaba provisioned profile: %+v", plan)
	}
	if got, want := plan.ProvisionedProfile.Strategy, "dedicated_profile"; got != want {
		t.Fatalf("strategy inesperada: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["HOME"], filepath.Clean("/srv/orquesta/gemini/homes/Gemini1"); got != want {
		t.Fatalf("HOME provisionado inesperado: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["GEMINI_HOME"], filepath.Clean("/srv/orquesta/gemini/homes/Gemini1"); got != want {
		t.Fatalf("template GEMINI_HOME inesperado: got=%q want=%q", got, want)
	}
	if got, want := plan.Env["GEMINI_PROFILE"], "Gemini1"; got != want {
		t.Fatalf("template GEMINI_PROFILE inesperado: got=%q want=%q", got, want)
	}
}

func TestPrepareCLICodexPriorizaSliceEjecutableEnPipelineLocalContinuityPrompt(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex9",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir frente activo",
			Branch:             "orq-orquestador-codex9",
			ResumePayloadJSON: `{
				"mailbox":[
					{
						"id":1,
						"kind":"pipeline_local",
						"payload":{
							"source":"pipeline_local",
							"accion":"continuar_trabajo",
							"tarea_objetivo_id":585,
							"carril":"premium_worktree",
							"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go","runtimeagente/driver.go"],
							"instruction":"FASE: implementacion\nACCION: continuar_trabajo\nTAREA: #585 Micro-refactorización cíclica del control plane\nSimbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente y helpers inmediatos del mismo slice.\nTests minimos del slice: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1"
						}
					}
				]
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for _, token := range []string{
		"mailbox=1",
		"tarea#585",
		"carril=premium_worktree",
		"write_set=cmd/controlplane_support.go, db/controlplane_entities.go, runtimeagente/driver.go",
		"Simbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente",
		"Tests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
	} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt priorizado: %s", token, plan.ContinuityPrompt)
		}
	}
	if strings.Contains(plan.ContinuityPrompt, `"write_set"`) {
		t.Fatalf("el prompt no deberia incrustar JSON bruto: %s", plan.ContinuityPrompt)
	}
	for _, token := range []string{"project_context", "governance_catalog", "adopted_context"} {
		if strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("el continuity prompt priorizado no deberia duplicar %q: %s", token, plan.ContinuityPrompt)
		}
	}
	for _, token := range []string{
		"Proyecto: orquestador.",
		"Rama: orq-orquestador-codex9.",
		"Ignora cualquier conversación vieja que no coincida con la tarea activa o el mailbox actual.",
		"No abras frentes nuevos ni reescribas código fuera del alcance inmediato.",
		"Empieza por la tarea asignada y consulta Orquesta antes de desviarte.",
	} {
		if strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("el continuity prompt priorizado no deberia duplicar guardrail %q: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLICodexPriorizaPipelineLocalAunqueNoSeaLaPrimeraMailbox(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex9",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir frente activo",
			Branch:             "orq-orquestador-codex9",
			ResumePayloadJSON: `{
				"mailbox":[
					{"id":7,"kind":"instruction","payload":{"texto":"ruido previo"}},
					{
						"id":8,
						"kind":"pipeline_local",
						"payload":{
							"source":"pipeline_local",
							"accion":"continuar_trabajo",
							"tarea_objetivo_id":585,
							"carril":"premium_worktree",
							"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go","runtimeagente/driver.go"],
							"instruction":"FASE: implementacion\nACCION: continuar_trabajo\nTAREA: #585 Micro-refactorización cíclica del control plane\nSimbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente y helpers inmediatos del mismo slice."
						}
					}
				]
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for _, token := range []string{
		"mailbox=2",
		"tarea#585",
		"carril=premium_worktree",
		"Simbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente",
	} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt priorizado: %s", token, plan.ContinuityPrompt)
		}
	}
	for _, token := range []string{
		"Proyecto: orquestador.",
		"Rama: orq-orquestador-codex9.",
		"Ignora cualquier conversación vieja que no coincida con la tarea activa o el mailbox actual.",
	} {
		if strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("no deberia degradar a prompt generico por mailbox inicial no priorizada, token=%q prompt=%s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLIGenericoResumePayloadResumeMailboxCompacto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Claude7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "claude-code",
			Transporte: "cli",
			Comando:    "claude-code",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir trabajo actual",
			Branch:             "orq-orquestador-claude7",
			ResumePayloadJSON:  `{"mailbox":[{"id":2,"kind":"instruction","payload":{"tarea_id":528,"texto":"cambia solo internal/controlruntime/tmux_monitor.go"}}]}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for _, token := range []string{"Resume payload:", "mailbox=1", "instruction", "tarea#528", "cambia solo internal/controlruntime/tmux_monitor.go"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt generico: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLIGenericoResumePayloadOcultaBootstrapPromptRuido(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Claude8",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "claude-code",
			Transporte: "cli",
			Comando:    "claude-code",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir trabajo actual",
			Branch:             "orq-orquestador-claude8",
			ResumePayloadJSON:  `{"bootstrap_prompt":"Bootstrap de Orquesta para Claude8.","checkpoint":{"id":14,"kind":"stop"},"mailbox":[{"id":2,"kind":"instruction","payload":{"texto":"cambia solo runtimeagente/driver.go"}}]}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for _, token := range []string{"Resume payload:", "checkpoint#14(stop)", "mailbox=1", "instruction", "cambia solo runtimeagente/driver.go"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt generico: %s", token, plan.ContinuityPrompt)
		}
	}
	for _, token := range []string{"bootstrap_prompt", "Bootstrap de Orquesta para Claude8."} {
		if strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("continuity prompt generico no deberia filtrar ruido de bootstrap %q: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLIOllamaNoReanudaContextoNoReanudable(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "OllamaLocal",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "qwen2.5-coder:7b",
		Conector: ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			MetadataJSON: `{"reanudable":false}`,
		},
		Resume: ResumeContext{
			ResumenContinuidad: "texto largo que no deberia pasar completo al prompt de continuidad",
			Branch:             "orq-orquestador-ollama",
			ResumePayloadJSON:  `{"checkpoint":{"id":33155,"kind":"stop"},"mailbox":[{"id":1}],"project_context":{},"governance_catalog":{}}`,
			CWD:                "/tmp/orquestador/.orquesta-worktrees/orq-ollama",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "launch" {
		t.Fatalf("ollama no deberia entrar en resume: %+v", plan)
	}
	if plan.NativeResume {
		t.Fatalf("ollama no deberia activar resume nativo: %+v", plan)
	}
	if strings.TrimSpace(plan.ContinuityPrompt) != "" {
		t.Fatalf("ollama no deberia heredar continuity prompt: %q", plan.ContinuityPrompt)
	}
	if strings.TrimSpace(plan.BootstrapPrompt) == "" {
		t.Fatalf("ollama deberia recibir bootstrap minimo al arrancar: %+v", plan)
	}
	if !strings.Contains(plan.BootstrapPrompt, "PROTOCOLO_ORQUESTA_MICRO") {
		t.Fatalf("bootstrap minimo inesperado: %q", plan.BootstrapPrompt)
	}
	if plan.WorkingDir != "/tmp/orquestador/.orquesta-worktrees/orq-ollama" {
		t.Fatalf("deberia conservar cwd de trabajo aunque no reanude: %s", plan.WorkingDir)
	}
}

func TestSanitizarResumeParaConectorNoBorraPayloadPersistido(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"perfil_ejecucion":{"modelo":"gemma4:26b"}}`,
	})
	if resume.ExternalSessionID != "" || resume.ResumenContinuidad != "" {
		t.Fatalf("el conector no reanudable debe limpiar continuidad de proveedor: %+v", resume)
	}
	if !strings.Contains(resume.ResumePayloadJSON, "gemma4:26b") {
		t.Fatalf("el payload persistido no deberia perderse: %s", resume.ResumePayloadJSON)
	}
}

func TestSanitizarResumeParaConectorPoolCompartidoConservaResumenBreve(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama_pool_local",
		Transporte:   "api",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false,"pool_compartido":true}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir con helper ya extraido",
		ResumePayloadJSON:  `{"perfil_ejecucion":{"modelo":"gemma4:26b","perfil_tarea":"implementacion"}}`,
	})
	if resume.ExternalSessionID != "" {
		t.Fatalf("el pool compartido no debe conservar continuidad de proveedor: %+v", resume)
	}
	if got := resume.ResumenContinuidad; got != "seguir con helper ya extraido" {
		t.Fatalf("el pool compartido debe conservar resumen breve: %+v", resume)
	}
	if !strings.Contains(resume.ResumePayloadJSON, "gemma4:26b") {
		t.Fatalf("el payload persistido no deberia perderse: %s", resume.ResumePayloadJSON)
	}
}

func TestSanitizarResumeParaConectorConservaSoloPerfilCuandoPayloadMezclaContextoReanudable(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"external_session_id":"sess-1","checkpoint_kind":"handoff_prepare","perfil_ejecucion":{"modelo":"gemma4:26b","perfil_tarea":"implementacion"}}`,
	})
	if resume.ExternalSessionID != "" || resume.ResumenContinuidad != "" {
		t.Fatalf("el conector no reanudable debe limpiar continuidad de proveedor: %+v", resume)
	}
	if !strings.Contains(resume.ResumePayloadJSON, `"perfil_ejecucion"`) || !strings.Contains(resume.ResumePayloadJSON, "gemma4:26b") {
		t.Fatalf("deberia conservar solo perfil_ejecucion util: %s", resume.ResumePayloadJSON)
	}
	if strings.Contains(resume.ResumePayloadJSON, "checkpoint_kind") || strings.Contains(resume.ResumePayloadJSON, "external_session_id") {
		t.Fatalf("deberia purgar contexto reanudable legado del payload: %s", resume.ResumePayloadJSON)
	}
}

func TestSanitizarResumeParaConectorConservaHintsOperativosDePerfil(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"external_session_id":"sess-1","checkpoint_kind":"handoff_prepare","perfil_ejecucion":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","perfil_operativo":"qa-heavy","driver":"tmux_cli_session","transport":"tmux","mailbox_delivery_mode":"session_resume","worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex1","tmux_session":"orq-codex1","tmux_pane_id":"%7","can_send_input":false}}`,
	})
	if resume.ExternalSessionID != "" || resume.ResumenContinuidad != "" {
		t.Fatalf("el conector no reanudable debe limpiar continuidad de proveedor: %+v", resume)
	}
	for _, token := range []string{
		`"execution_profile":"qa-heavy"`,
		`"perfil_operativo":"qa-heavy"`,
		`"driver":"tmux_cli_session"`,
		`"transport":"tmux"`,
		`"mailbox_delivery_mode":"session_resume"`,
		`"worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex1"`,
		`"tmux_session":"orq-codex1"`,
		`"tmux_pane_id":"%7"`,
		`"can_send_input":false`,
	} {
		if !strings.Contains(resume.ResumePayloadJSON, token) {
			t.Fatalf("faltaba hint operativo %s en payload saneado: %s", token, resume.ResumePayloadJSON)
		}
	}
	if strings.Contains(resume.ResumePayloadJSON, "checkpoint_kind") || strings.Contains(resume.ResumePayloadJSON, "external_session_id") {
		t.Fatalf("deberia purgar contexto reanudable legado del payload: %s", resume.ResumePayloadJSON)
	}
}

func TestSanitizarResumeParaConectorAceptaExecutionProfileCanonico(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"external_session_id":"sess-1","checkpoint_kind":"handoff_prepare","perfil_ejecucion":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","execution_profile":"persistente","driver":"tmux_cli_session","transport":"tmux","mailbox_delivery_mode":"session_resume","worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex2","tmux_session":"orq-codex2","tmux_pane_id":"%8","can_send_input":false}}`,
	})
	for _, token := range []string{
		`"execution_profile":"persistente"`,
		`"perfil_operativo":"persistente"`,
		`"driver":"tmux_cli_session"`,
		`"transport":"tmux"`,
		`"mailbox_delivery_mode":"session_resume"`,
		`"worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex2"`,
		`"tmux_session":"orq-codex2"`,
		`"tmux_pane_id":"%8"`,
		`"can_send_input":false`,
	} {
		if !strings.Contains(resume.ResumePayloadJSON, token) {
			t.Fatalf("faltaba hint operativo %s en payload saneado: %s", token, resume.ResumePayloadJSON)
		}
	}
	if strings.Contains(resume.ResumePayloadJSON, "checkpoint_kind") || strings.Contains(resume.ResumePayloadJSON, "external_session_id") {
		t.Fatalf("deberia purgar contexto reanudable legado del payload: %s", resume.ResumePayloadJSON)
	}
}

func TestSanitizarResumeParaConectorConservaArtifactsVersionados(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"external_session_id":"sess-1","checkpoint_kind":"handoff_prepare","artifacts":[{"artifact_id":"patch-1","scope":"task","kind":"patch","version":3,"path":"artifacts/patches/patch-1.diff"}],"artifacts_ref":["handoff:patch-1"],"perfil_ejecucion":{"modelo":"gemma4:26b","perfil_tarea":"implementacion"}}`,
	})
	if resume.ExternalSessionID != "" || resume.ResumenContinuidad != "" {
		t.Fatalf("el conector no reanudable debe limpiar continuidad de proveedor: %+v", resume)
	}
	for _, token := range []string{`"artifacts"`, `"artifact_id":"patch-1"`, `"artifacts_ref":["handoff:patch-1"]`, `"perfil_ejecucion"`} {
		if !strings.Contains(resume.ResumePayloadJSON, token) {
			t.Fatalf("faltaba %s en payload saneado: %s", token, resume.ResumePayloadJSON)
		}
	}
	if strings.Contains(resume.ResumePayloadJSON, "checkpoint_kind") || strings.Contains(resume.ResumePayloadJSON, "external_session_id") {
		t.Fatalf("deberia purgar contexto reanudable legado del payload: %s", resume.ResumePayloadJSON)
	}
}

func TestResumePayloadTieneContextoReanudableIgnoraArtifactsVersionados(t *testing.T) {
	if resumePayloadTieneContextoReanudable(`{"artifacts":[{"artifact_id":"patch-1","kind":"patch","version":3,"path":"artifacts/patches/patch-1.diff"}],"artifacts_ref":["handoff:patch-1"],"perfil_ejecucion":{"modelo":"gpt-5.5"}}`) {
		t.Fatal("artifacts versionados no deberian activar contexto reanudable por si solos")
	}
}

func TestModeloCompatibleConConectorPremiumNoAceptaModeloLocal(t *testing.T) {
	cases := []struct {
		name     string
		conector ConnectorConfig
		modelo   string
		want     bool
	}{
		{
			name: "codex rechaza qwen local",
			conector: ConnectorConfig{
				Slug:         "codex-cli",
				Transporte:   "cli",
				Comando:      "codex",
				MetadataJSON: `{"familia":"openai"}`,
			},
			modelo: "qwen2.5-coder:14b",
			want:   false,
		},
		{
			name: "codex acepta gpt",
			conector: ConnectorConfig{
				Slug:         "codex-cli",
				Transporte:   "cli",
				Comando:      "codex",
				MetadataJSON: `{"familia":"openai"}`,
			},
			modelo: "gpt-5.4",
			want:   true,
		},
		{
			name: "claude rechaza gemma",
			conector: ConnectorConfig{
				Slug:         "claude-code",
				Transporte:   "cli",
				Comando:      "claude-code",
				MetadataJSON: `{"familia":"anthropic"}`,
			},
			modelo: "gemma4:26b",
			want:   false,
		},
		{
			name: "claude sin metadata familia rechaza qwen por comando",
			conector: ConnectorConfig{
				Slug:       "claude-code",
				Transporte: "cli",
				Comando:    "claude-code",
			},
			modelo: "qwen2.5-coder:14b",
			want:   false,
		},
		{
			name: "gemini acepta gemini",
			conector: ConnectorConfig{
				Slug:         "gemini-cli",
				Transporte:   "cli",
				Comando:      "gemini",
				MetadataJSON: `{"familia":"google"}`,
			},
			modelo: "gemini-2.5-pro",
			want:   true,
		},
		{
			name: "gemini sin metadata familia rechaza qwen por comando",
			conector: ConnectorConfig{
				Slug:       "gemini-cli",
				Transporte: "cli",
				Comando:    "gemini",
			},
			modelo: "qwen2.5-coder:14b",
			want:   false,
		},
		{
			name: "ollama conserva modelo local",
			conector: ConnectorConfig{
				Slug:       "ollama-cli",
				Transporte: "cli",
				Comando:    "ollama",
			},
			modelo: "qwen2.5-coder:14b",
			want:   true,
		},
		{
			name: "ollama rechaza gpt remoto",
			conector: ConnectorConfig{
				Slug:       "ollama-cli",
				Transporte: "cli",
				Comando:    "ollama",
			},
			modelo: "gpt-5.4",
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ModeloCompatibleConConector(tc.conector, tc.modelo)
			if got != tc.want {
				t.Fatalf("ModeloCompatibleConConector(%q)=%t want=%t", tc.modelo, got, tc.want)
			}
		})
	}
}

func TestPrepareRemotoSoloUsaResumeConSesionExternaReal(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "CodexRemote",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "http://127.0.0.1:8080",
			MetadataJSON: `{"launch_path":"/launch","resume_path":"/resume"}`,
		},
		Resume: ResumeContext{
			ResumePayloadJSON:  `{"checkpoint":{"id":7}}`,
			ResumenContinuidad: "seguir desde checkpoint",
		},
	})
	if err != nil {
		t.Fatalf("prepare remoto: %v", err)
	}
	if plan.Modo != "launch" {
		t.Fatalf("el remoto no debe usar /resume sin external_session_id: %+v", plan)
	}
	if strings.TrimSpace(plan.ContinuityPrompt) != "" {
		t.Fatalf("el remoto no deberia fabricar continuity prompt sin sesión externa: %q", plan.ContinuityPrompt)
	}
}

func TestPrepareCLIConHintsActivaResumeNativo(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "claude1",
		ProyectoSlug: "conta",
		ProyectoRuta: "/tmp/conta",
		Conector: ConnectorConfig{
			Slug:         "claude-code",
			Transporte:   "cli",
			Comando:      "claude",
			MetadataJSON: `{"resume_subcommand":"resume","cwd_flag":"--cwd","session_id_flag":"--session"}`,
		},
		Resume: ResumeContext{
			ExternalSessionID: "abc-123",
			CWD:               "/tmp/conta",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !plan.NativeResume {
		t.Fatalf("debía activar resume nativo")
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'resume'") || !strings.Contains(rendered, "'--session'") || !strings.Contains(rendered, "'abc-123'") {
		t.Fatalf("comando inesperado: %s", rendered)
	}
}

func TestPrepareCLINormalizaReasoningLegacyDeCodexCLI(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		PerfilTarea:  "orquestacion",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"model_flag":"--model","reasoning_flag":"--reasoning-effort"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modelo != "gpt-5.4" || plan.Razonamiento != "high" || plan.PerfilTarea != "orquestacion" {
		t.Fatalf("perfil inesperado: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'--model'") || !strings.Contains(rendered, "'gpt-5.4'") {
		t.Fatalf("comando sin modelo esperado: %s", rendered)
	}
	if strings.Contains(rendered, "'--reasoning-effort'") {
		t.Fatalf("comando no deberia usar flag legacy de razonamiento: %s", rendered)
	}
	if !strings.Contains(rendered, "'-c'") || !strings.Contains(rendered, `'model_reasoning_effort="high"'`) {
		t.Fatalf("comando inesperado: %s", rendered)
	}
}

func TestPrepareCLIPropagaRazonamientoPorConfigKey(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"model_flag":"--model","reasoning_config_key":"model_reasoning_effort"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'--model'") || !strings.Contains(rendered, "'gpt-5.4'") {
		t.Fatalf("comando sin modelo esperado: %s", rendered)
	}
	if !strings.Contains(rendered, "'-c'") || !strings.Contains(rendered, `'model_reasoning_effort="high"'`) {
		t.Fatalf("comando sin override de razonamiento esperado: %s", rendered)
	}
}

func TestPrepareCLIExpandePlaceholdersDeConector(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex4",
		Rol:          "programador",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "/tmp/codex-perfil",
			ArgsJSON:   `["{{agent}}","--modo","{{task_profile}}"]`,
			EnvJSON:    `{"ORQUESTA_AGENTE":"{{agent}}","ORQUESTA_PROYECTO":"{{project_slug}}","ORQUESTA_MODELO":"{{model}}","ORQUESTA_BIN":"{{orquesta_executable}}","PATH":"{{orquesta_bin_dir}}:{{host_path}}"}`,
		},
		Resume: ResumeContext{
			Branch: "feature/control-plane",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Comando != "/tmp/codex-perfil" {
		t.Fatalf("comando inesperado: %s", plan.Comando)
	}
	if len(plan.Args) != 7 || plan.Args[0] != "Codex4" || plan.Args[2] != "implementacion" {
		t.Fatalf("args inesperados: %+v", plan.Args)
	}
	if plan.Args[3] != "--sandbox" || plan.Args[4] != "danger-full-access" || plan.Args[5] != "--ask-for-approval" || plan.Args[6] != "never" {
		t.Fatalf("faltan flags operativas por defecto: %+v", plan.Args)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if plan.Env["ORQUESTA_AGENTE"] != "Codex4" || plan.Env["ORQUESTA_PROYECTO"] != "orquestador" || plan.Env["ORQUESTA_MODELO"] != "gpt-5.4" {
		t.Fatalf("env inesperado: %+v", plan.Env)
	}
	if plan.Env["ORQUESTA_BIN"] != exe {
		t.Fatalf("orquesta bin inesperado: %+v", plan.Env)
	}
	wantPathPrefix := filepath.Dir(plan.Comando) + string(os.PathListSeparator)
	if !strings.Contains(plan.Env["PATH"], os.Getenv("PATH")) || !strings.HasPrefix(plan.Env["PATH"], wantPathPrefix) {
		t.Fatalf("PATH expandido inesperado: %+v", plan.Env)
	}
}

func TestPrepareCLIResuelveComandoRelativoContraProyecto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex8",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/srv/orquesta",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "../codex-perfiles/bin/codex-perfil",
			ArgsJSON:   `["{{agent}}"]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, "/srv/codex-perfiles/bin/codex-perfil"; got != want {
		t.Fatalf("comando relativo inesperado: got=%s want=%s", got, want)
	}
	if got := plan.Env["ORQUESTA_TERMINAL_BACKEND"]; got != "tmux" {
		t.Fatalf("backend codex-perfil inesperado: %+v", plan.Env)
	}
}

func TestPrepareCLICodexWrapperInyectaCODEXBINConPathPobre(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USE_PROFILE_WRAPPER", "1")
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin")

	projectPath := filepath.Join(tmp, "srv", "orquesta")
	wrapperPath := filepath.Join(tmp, "srv", "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	codexPath := filepath.Join(home, ".nvm", "versions", "node", "v20.19.2", "bin", "codex")
	if err := os.MkdirAll(filepath.Dir(codexPath), 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}
	if err := os.WriteFile(codexPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write codex: %v", err)
	}

	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: projectPath,
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, wrapperPath; got != want {
		t.Fatalf("wrapper inesperado: got=%q want=%q", got, want)
	}
	if got, want := strings.TrimSpace(plan.Env["CODEX_BIN"]), codexPath; got != want {
		t.Fatalf("CODEX_BIN inesperado: got=%q want=%q env=%+v", got, want, plan.Env)
	}
	if !strings.Contains(plan.Env["PATH"], filepath.Dir(codexPath)) {
		t.Fatalf("PATH deberia incluir el binario de codex: %+v", plan.Env)
	}
	if got := strings.TrimSpace(plan.Env["ORQUESTA_TERMINAL_BACKEND"]); got != "tmux" {
		t.Fatalf("backend inesperado para codex-perfil: %+v", plan.Env)
	}
}

func TestPrepareCLIOllamaUsaModeloPosicional(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Ollama1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			ArgsJSON:     `["run"]`,
			MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","model_positional":true,"launch_prompt_transport":"post_start"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modelo != "qwen2.5-coder:7b" {
		t.Fatalf("modelo inesperado: %+v", plan)
	}
	if strings.TrimSpace(plan.BootstrapPrompt) == "" {
		t.Fatalf("ollama deberia exponer bootstrap minimo: %+v", plan)
	}
	if plan.LaunchPromptMode != "post_start" {
		t.Fatalf("ollama deberia usar post_start para bootstrap: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'run'") || !strings.Contains(rendered, "'qwen2.5-coder:7b'") {
		t.Fatalf("comando ollama inesperado: %s", rendered)
	}
}

func TestPrepareCLIOllamaModeloGeneralUsaBootstrapNatural(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemma1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gemma4:27b",
		Conector: ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			ArgsJSON:     `["run"]`,
			MetadataJSON: `{"launch_prompt_transport":"post_start"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.Contains(plan.BootstrapPrompt, "PROTOCOLO_ORQUESTA_MICRO") {
		t.Fatalf("gemma no deberia recibir bootstrap de wire protocol: %q", plan.BootstrapPrompt)
	}
	for _, token := range []string{"worker local de Orquesta", "ACK-ESPERA", "BLOQUEO:"} {
		if !strings.Contains(plan.BootstrapPrompt, token) {
			t.Fatalf("faltaba %q en bootstrap natural: %q", token, plan.BootstrapPrompt)
		}
	}
}

func TestAplicarDefaultsConectorNoPisaPoliticaResuelta(t *testing.T) {
	perfil, modelo, razonamiento, err := AplicarDefaultsConector(
		ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","default_task_profile":"implementacion"}`,
		},
		"",
		"",
		"",
		"correccion",
		"gemma4:26b",
		"high",
	)
	if err != nil {
		t.Fatalf("AplicarDefaultsConector: %v", err)
	}
	if perfil != "correccion" || modelo != "gemma4:26b" || razonamiento != "high" {
		t.Fatalf("los defaults del conector no deberian pisar una politica ya resuelta: perfil=%q modelo=%q razonamiento=%q", perfil, modelo, razonamiento)
	}
}

func TestPrepareCLICodexSimpleUsaWrapperCanonicoYPerfil(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USE_PROFILE_WRAPPER", "1")
	tmp := t.TempDir()
	projectPath := filepath.Join(tmp, "orquesta")
	wrapperPath := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex8",
		ProyectoSlug: "orquestador",
		ProyectoRuta: projectPath,
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, wrapperPath; got != want {
		t.Fatalf("comando canonico inesperado: got=%s want=%s", got, want)
	}
	if len(plan.Args) != 5 {
		t.Fatalf("args inesperados: %+v", plan.Args)
	}
	if plan.Args[0] != "Codex8" {
		t.Fatalf("el wrapper codex debe recibir el perfil como primer argumento: %+v", plan.Args)
	}
}

func TestPrepareCLICodexSimpleUsaCodexGlobalPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USE_PROFILE_WRAPPER", "")
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "CodexGlobal1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
			ArgsJSON:   `[]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	want := "codex"
	if resolved, err := exec.LookPath("codex"); err == nil && strings.TrimSpace(resolved) != "" {
		want = resolved
	}
	if got := plan.Comando; got != want {
		t.Fatalf("comando inesperado: got=%q want=%q", got, want)
	}
	if len(plan.Args) > 0 && plan.Args[0] == "CodexGlobal1" {
		t.Fatalf("codex global no deberia recibir perfil como primer argumento: %+v", plan.Args)
	}
}

func TestApplyLaunchPromptMetadataEmbebePromptInicial(t *testing.T) {
	plan := &LaunchPlan{
		Comando:          "codex",
		Args:             []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt:  "Bootstrap de Orquesta para Codex2.",
		ContinuityPrompt: "Retoma el contexto del proyecto demo.",
		Env:              map[string]string{},
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_positional":true}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if !plan.LaunchPromptEmbedded {
		t.Fatalf("el plan deberia marcar el prompt inicial como embebido: %+v", plan)
	}
	if got := plan.Args[len(plan.Args)-1]; got != "Bootstrap de Orquesta para Codex2.\n\nRetoma el contexto del proyecto demo." {
		t.Fatalf("prompt final inesperado: %q", got)
	}

	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'Bootstrap de Orquesta para Codex2.\n\nRetoma el contexto del proyecto demo.'") {
		t.Fatalf("comando renderizado sin prompt inicial citado: %s", rendered)
	}
}

func TestApplyLaunchPromptMetadataPostStartNoEmbebePromptInicial(t *testing.T) {
	plan := &LaunchPlan{
		Comando:          "codex",
		Args:             []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt:  "Bootstrap de Orquesta para Codex2.",
		ContinuityPrompt: "Retoma el contexto del proyecto demo.",
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_transport":"post_start","launch_prompt_delay_ms":1000}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.LaunchPromptEmbedded {
		t.Fatalf("el plan no deberia marcar el prompt inicial como embebido: %+v", plan)
	}
	if plan.LaunchPromptMode != "post_start" {
		t.Fatalf("modo de prompt inesperado: %+v", plan)
	}
	if plan.LaunchPromptDelayMS != 1000 {
		t.Fatalf("delay de prompt inesperado: %+v", plan)
	}
	if len(plan.Args) != 3 {
		t.Fatalf("args inesperados tras post_start: %+v", plan.Args)
	}
}

func TestApplyLaunchPromptMetadataExcesoTamanoCaeAPostStart(t *testing.T) {
	plan := &LaunchPlan{
		Comando:         "codex",
		Args:            []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt: strings.Repeat("a", 32),
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_positional":true,"launch_prompt_max_bytes":8}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.LaunchPromptEmbedded {
		t.Fatalf("el plan no deberia embeber un prompt sobredimensionado: %+v", plan)
	}
	if plan.LaunchPromptMode != "post_start" {
		t.Fatalf("modo de prompt inesperado: %+v", plan)
	}
	if len(plan.Args) != 3 {
		t.Fatalf("args inesperados tras fallback por tamano: %+v", plan.Args)
	}
}

func TestNormalizeConnectorMetadataCodexPreservaPromptEmbebido(t *testing.T) {
	normalized := normalizeConnectorMetadata(ConnectorConfig{
		Slug:    "codex-cli",
		Comando: "codex",
	}, map[string]any{
		"launch_prompt_positional": true,
	})
	if got := stringMetadata(normalized, "launch_prompt_transport"); got != "" {
		t.Fatalf("codex no deberia forzar post_start cuando el conector ya soporta prompt embebido: %q", got)
	}
	if !boolMetadata(normalized, "launch_prompt_positional") {
		t.Fatalf("codex deberia preservar launch_prompt_positional: %+v", normalized)
	}
	if !hasMetadataKey(normalized, "can_send_input") || boolMetadata(normalized, "can_send_input") {
		t.Fatalf("codex deberia seguir con can_send_input=false por defecto: %+v", normalized)
	}
	if !hasMetadataKey(normalized, "can_run_slash_commands") || boolMetadata(normalized, "can_run_slash_commands") {
		t.Fatalf("codex no deberia anunciar slash commands por defecto en el plan canonico: %+v", normalized)
	}
}

func TestPrepareRemoteGeneraContratoMinimoDeAdaptador(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "https://runtime.example.test",
			MetadataJSON: `{"resume_path":"/sessions/resume","status_path":"/sessions/status","input_path":"/sessions/input","auth_header":"Authorization","auth_token_env":"ORQUESTA_REMOTE_TOKEN","auth_mode":"oauth","preserve_external_session_on_stop":true,"requires_human_reauth":true,"timeout_ms":2500,"control_retry_count":4,"control_retry_backoff_ms":150,"control_retry_statuses":[408,429,503]}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.TrimSpace(plan.RemoteConfigJSON) == "" {
		t.Fatalf("faltaba remote_config_json: %+v", plan)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(plan.RemoteConfigJSON), &cfg); err != nil {
		t.Fatalf("parse remote config: %v", err)
	}
	if cfg["endpoint"] != "https://runtime.example.test" {
		t.Fatalf("endpoint inesperado: %+v", cfg)
	}
	if cfg["resume_path"] != "/sessions/resume" || cfg["status_path"] != "/sessions/status" || cfg["input_path"] != "/sessions/input" {
		t.Fatalf("paths inesperados: %+v", cfg)
	}
	if cfg["auth_mode"] != "oauth" || cfg["preserve_external_session_on_stop"] != true || cfg["requires_human_reauth"] != true {
		t.Fatalf("auth/persistencia inesperadas: %+v", cfg)
	}
	if cfg["timeout_ms"] != float64(2500) {
		t.Fatalf("timeout inesperado: %+v", cfg)
	}
	if cfg["control_retry_count"] != float64(4) || cfg["control_retry_backoff_ms"] != float64(150) {
		t.Fatalf("retries inesperados: %+v", cfg)
	}
}

func TestPrepareRemotePermiteDesactivarCapacidadesPorMetadata(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "https://runtime.example.test",
			MetadataJSON: `{"pause_path":"","input_path":"","can_pause":false,"can_send_input":false,"can_stop":false}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(plan.RemoteConfigJSON), &cfg); err != nil {
		t.Fatalf("parse remote config: %v", err)
	}
	if cfg["pause_path"] != "" || cfg["input_path"] != "" {
		t.Fatalf("paths remotos inesperados: %+v", cfg)
	}
	if cfg["can_pause"] != false || cfg["can_send_input"] != false || cfg["can_stop"] != false {
		t.Fatalf("capacidades remotas inesperadas: %+v", cfg)
	}
}

func TestPrepareCLIPropagaCanSendInputDesdeMetadata(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"can_send_input":false}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("el plan CLI deberia propagar can_send_input=false desde metadata: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("modo mailbox inesperado: %+v", plan)
	}
}

func TestPrepareCLICodexDesactivaSendInputInteractivoPorDefecto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli deberia desactivar input interactivo por defecto: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex-cli deberia dejar mailbox en bootstrap_only: %+v", plan)
	}
	rendered := RenderCommand(plan)
	for _, token := range []string{"'--sandbox' 'danger-full-access'", "'--ask-for-approval' 'never'"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("codex-cli deberia arrancar con permisos reales por defecto; falta %s en %s", token, rendered)
		}
	}
}

func TestPrepareCLICodexResumeUsaSessionResumeCuandoTieneSesionExterna(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ExternalSessionID: "sess-codex-7",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli deberia seguir sin input interactivo en resume: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("codex-cli resume deberia usar session_resume: %+v", plan)
	}
}

func TestPrepareCLICodexResumeCorrigeInteractiveASessionResumeSinSendInput(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"can_send_input":false,"mailbox_delivery_mode":"interactive"}`,
		},
		Resume: ResumeContext{
			ExternalSessionID: "sess-codex-7",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli resume no deberia habilitar input interactivo aqui: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("codex-cli resume deberia corregir interactive a session_resume cuando hay sesion externa sin send_input: %+v", plan)
	}
}

func TestPrepareCLICodexResumeCorrigeCoordinatedRestartASessionResumeSinSendInput(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		},
		Resume: ResumeContext{
			ExternalSessionID: "sess-codex-7",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli resume no deberia habilitar input interactivo aqui: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("codex-cli resume deberia corregir coordinated_restart a session_resume cuando hay sesion externa sin send_input: %+v", plan)
	}
}

func TestPrepareCLICodexResumeCorrigeBootstrapOnlyASessionResumeSinSendInput(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		},
		Resume: ResumeContext{
			ExternalSessionID: "sess-codex-7",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli resume no deberia habilitar input interactivo aqui: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("codex-cli resume deberia corregir bootstrap_only a session_resume cuando hay sesion externa sin send_input: %+v", plan)
	}
}

func TestPrepareCLIGeminiEmbebePromptInteractivo(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemini1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "gemini-cli",
			Transporte:   "cli",
			Comando:      "gemini",
			ArgsJSON:     `["--yolo"]`,
			MetadataJSON: `{"familia":"google","reanudable":true,"launch_prompt_flag":"--prompt-interactive","model_flag":"--model","can_send_input":false}`,
		},
		Modelo: "gemini-2.5-flash-lite",
		Resume: ResumeContext{
			ResumenContinuidad: "Tarea activa: corregir merge helper",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_flag":"--prompt-interactive","model_flag":"--model","can_send_input":false}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("gemini-cli deberia quedar sin input interactivo directo: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("gemini-cli deberia usar bootstrap_only: %+v", plan)
	}
	if !plan.LaunchPromptEmbedded {
		t.Fatalf("gemini-cli deberia embeber el prompt inicial: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'--model' 'gemini-2.5-flash-lite'") {
		t.Fatalf("gemini-cli deberia propagar --model: %s", rendered)
	}
	if !strings.Contains(rendered, "'--prompt-interactive'") {
		t.Fatalf("gemini-cli deberia usar --prompt-interactive: %s", rendered)
	}
	if !strings.Contains(rendered, "Tarea activa: corregir merge helper") {
		t.Fatalf("gemini-cli deberia incluir continuidad en el prompt: %s", rendered)
	}
}

func TestPrepareCLIGeminiPrependaDirectorioDelComandoAlPATH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/bin")
	geminiBin := filepath.Join(home, ".nvm", "versions", "node", "v99.0.0", "bin")
	if err := os.MkdirAll(geminiBin, 0o755); err != nil {
		t.Fatalf("mkdir gemini bin: %v", err)
	}
	geminiPath := filepath.Join(geminiBin, "gemini")
	if err := os.WriteFile(geminiPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write gemini stub: %v", err)
	}
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Gemini1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "gemini-cli",
			Transporte: "cli",
			Comando:    "gemini",
			ArgsJSON:   `["--yolo"]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, geminiPath; got != want {
		t.Fatalf("comando resuelto inesperado: got=%q want=%q", got, want)
	}
	if got := plan.Env["PATH"]; !strings.HasPrefix(got, geminiBin+string(os.PathListSeparator)) {
		t.Fatalf("PATH deberia comenzar por el directorio del comando resuelto: %q", got)
	}
}

func TestPrepareCLICodexInyectaHerramientasBaseEnPATH(t *testing.T) {
	goPath := "/usr/local/go/bin/go"
	if _, err := os.Stat(goPath); err != nil {
		t.Skip("go no disponible en la ruta canonica esperada")
	}
	t.Setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/bin")
	projectPath := t.TempDir()
	wrapperDir := filepath.Join(projectPath, "..", "codex-perfiles", "bin")
	wrapperPath := filepath.Join(wrapperDir, "codex-perfil")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: projectPath,
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    wrapperPath,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := plan.Env["PATH"]; !strings.Contains(got, filepath.Dir(goPath)) {
		t.Fatalf("PATH deberia incluir la toolchain de go: %q", got)
	}
}

func TestPrepareCLIInyectaEntornoBaseOrquesta(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "orquesta-localrpc.json")
	if err := rpclocal.SaveState(statePath, &rpclocal.State{Addr: "127.0.0.1:17669"}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	t.Setenv("ORQUESTA_SERVER_INFO", statePath)
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: t.TempDir(),
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "/tmp/codex-perfil",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.TrimSpace(plan.Env["ORQUESTA_SERVER_URL"]); got != "http://127.0.0.1:17669" {
		t.Fatalf("ORQUESTA_SERVER_URL inesperado: %+v", plan.Env)
	}
	if got := strings.TrimSpace(plan.Env["ORQUESTA_BIN"]); got == "" {
		t.Fatalf("ORQUESTA_BIN deberia inyectarse por defecto: %+v", plan.Env)
	}
}

func TestPrepareCLIIgnoraEnvResidualYUsaServerCanonico(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "orquesta-localrpc.json")
	if err := rpclocal.SaveState(statePath, &rpclocal.State{Addr: "127.0.0.1:17652"}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	t.Setenv("ORQUESTA_SERVER_INFO", statePath)
	t.Setenv("ORQUESTA_SERVER_URL", "http://127.0.0.1:16543")
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex3",
		ProyectoSlug: "orquestador",
		ProyectoRuta: t.TempDir(),
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "/tmp/codex-perfil",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.TrimSpace(plan.Env["ORQUESTA_SERVER_URL"]); got != "http://127.0.0.1:17652" {
		t.Fatalf("ORQUESTA_SERVER_URL deberia salir del server canonico, got=%q env=%+v", got, plan.Env)
	}
}

func TestPrepareCLISobrescribeServerURLLocalStaleDesdeConnectorEnv(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "orquesta-localrpc.json")
	if err := rpclocal.SaveState(statePath, &rpclocal.State{Addr: "127.0.0.1:17652"}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	t.Setenv("ORQUESTA_SERVER_INFO", statePath)
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex3",
		ProyectoSlug: "orquestador",
		ProyectoRuta: t.TempDir(),
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "/tmp/codex-perfil",
			EnvJSON:    `{"ORQUESTA_SERVER_URL":"http://127.0.0.1:16543"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := strings.TrimSpace(plan.Env["ORQUESTA_SERVER_URL"]); got != "http://127.0.0.1:17652" {
		t.Fatalf("ORQUESTA_SERVER_URL deberia corregir env stale del conector, got=%q env=%+v", got, plan.Env)
	}
}

func TestPrepareCLIClaudeEmbebePromptPosicional(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Claude1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "claude-code",
			Transporte:   "cli",
			Comando:      "claude-code",
			ArgsJSON:     `["--dangerously-skip-permissions","--permission-mode","bypassPermissions","--add-dir","{{working_dir}}","--session-id","{{session_uuid}}"]`,
			MetadataJSON: `{"familia":"anthropic","reanudable":true,"launch_prompt_positional":true,"can_send_input":false}`,
		},
		Resume: ResumeContext{
			ResumenContinuidad: "Tarea activa: corregir merge helper",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_positional":true,"can_send_input":false}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("claude-code deberia quedar sin input interactivo directo: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("claude-code deberia usar bootstrap_only: %+v", plan)
	}
	if !plan.LaunchPromptEmbedded {
		t.Fatalf("claude-code deberia embeber el prompt inicial: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "Tarea activa: corregir merge helper") {
		t.Fatalf("claude-code deberia incluir continuidad en el prompt: %s", rendered)
	}
	if strings.Contains(rendered, "'--bare'") {
		t.Fatalf("claude-code no deberia usar --bare cuando dependemos de OAuth/keychain: %s", rendered)
	}
}

func TestApplyLocalControlHintsNilMetadataNoCambiaPlan(t *testing.T) {
	plan := &LaunchPlan{MailboxDeliveryMode: MailboxDeliveryInteractive}
	applied := applyLocalControlHints(plan, nil)
	if applied {
		t.Fatal("nil metadata deberia devolver applied=false")
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryInteractive {
		t.Fatalf("nil metadata no deberia cambiar el modo existente: %s", plan.MailboxDeliveryMode)
	}
	if plan.CanSendInput != nil {
		t.Fatalf("nil metadata no deberia tocar can_send_input: %v", plan.CanSendInput)
	}
}

func TestApplyLocalControlHintsNilPlanDevuelveFalse(t *testing.T) {
	applied := applyLocalControlHints(nil, map[string]any{"can_send_input": true})
	if applied {
		t.Fatal("plan nil deberia devolver applied=false")
	}
}

func TestApplyLocalControlHintsAplicaAmbosKeysCuandoPresentes(t *testing.T) {
	plan := &LaunchPlan{}
	applied := applyLocalControlHints(plan, map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": MailboxDeliverySessionResume,
	})
	if !applied {
		t.Fatal("deberia devolver applied=true cuando se aplican hints")
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("can_send_input esperado false: %v", plan.CanSendInput)
	}
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("mailbox_delivery_mode inesperado: %s", plan.MailboxDeliveryMode)
	}
}

func TestApplyLocalControlHintsCanSendInputFalsoUsaBootstrapOnlyPorDefecto(t *testing.T) {
	plan := &LaunchPlan{}
	applied := applyLocalControlHints(plan, map[string]any{"can_send_input": false})
	if !applied {
		t.Fatal("deberia devolver applied=true cuando can_send_input esta presente")
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("can_send_input esperado false: %v", plan.CanSendInput)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("sin modo explicito y can_send_input=false deberia defaultear a bootstrap_only: %s", plan.MailboxDeliveryMode)
	}
}

func TestApplyLocalControlHintsModoPorMetadataNoUsaDefault(t *testing.T) {
	plan := &LaunchPlan{}
	applied := applyLocalControlHints(plan, map[string]any{
		"mailbox_delivery_mode": MailboxDeliveryCoordinatedRestart,
	})
	if !applied {
		t.Fatal("deberia devolver applied=true cuando mailbox_delivery_mode esta presente")
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryCoordinatedRestart {
		t.Fatalf("modo inesperado: %s", plan.MailboxDeliveryMode)
	}
	if plan.CanSendInput != nil {
		t.Fatalf("can_send_input no deberia tocarse si no esta en metadata: %v", plan.CanSendInput)
	}
}

func TestApplyLocalControlHintsModoDesconocidoUsaDefault(t *testing.T) {
	plan := &LaunchPlan{}
	applied := applyLocalControlHints(plan, map[string]any{
		"mailbox_delivery_mode": "modo_invalido",
	})
	if applied {
		t.Fatal("modo desconocido no deberia contar como hint aplicado")
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryInteractive {
		t.Fatalf("modo desconocido deberia caer al default interactive (can_send_input nil): %s", plan.MailboxDeliveryMode)
	}
}

func TestConectorPermiteResume(t *testing.T) {
	// nil metadata → siempre permite
	if !conectorPermiteResume(nil) {
		t.Error("nil metadata deberia permitir resume")
	}
	// sin clave "reanudable" → permite por defecto
	if !conectorPermiteResume(map[string]any{"otro": "valor"}) {
		t.Error("ausencia de clave reanudable deberia permitir resume")
	}
	// reanudable=true → permite
	if !conectorPermiteResume(map[string]any{"reanudable": true}) {
		t.Error("reanudable=true deberia permitir resume")
	}
	// reanudable=false → no permite
	if conectorPermiteResume(map[string]any{"reanudable": false}) {
		t.Error("reanudable=false deberia bloquear resume")
	}
}

func TestResumeActivaModoLocal(t *testing.T) {
	cases := []struct {
		name   string
		resume ResumeContext
		want   bool
	}{
		{"vacio", ResumeContext{}, false},
		{"solo external session", ResumeContext{ExternalSessionID: "sess-1"}, true},
		{"solo resumen continuidad", ResumeContext{ResumenContinuidad: "continua tarea"}, true},
		{"ambos presentes", ResumeContext{ExternalSessionID: "sess-2", ResumenContinuidad: "continua"}, true},
		{"solo espacios en blanco", ResumeContext{ExternalSessionID: "   ", ResumenContinuidad: "  "}, false},
	}
	for _, c := range cases {
		got := resumeActivaModoLocal(c.resume)
		if got != c.want {
			t.Errorf("[%s] resumeActivaModoLocal = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestResumeActivaModoRemoto(t *testing.T) {
	cases := []struct {
		name   string
		resume ResumeContext
		want   bool
	}{
		{"vacio", ResumeContext{}, false},
		{"con external session", ResumeContext{ExternalSessionID: "sess-remote"}, true},
		{"solo resumen sin sesion externa", ResumeContext{ResumenContinuidad: "continua"}, false},
		{"espacios en blanco", ResumeContext{ExternalSessionID: "   "}, false},
	}
	for _, c := range cases {
		got := resumeActivaModoRemoto(c.resume)
		if got != c.want {
			t.Errorf("[%s] resumeActivaModoRemoto = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestNormalizeMailboxDeliveryModeCaseInsensitive(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"interactive", MailboxDeliveryInteractive},
		{"INTERACTIVE", MailboxDeliveryInteractive},
		{"Interactive", MailboxDeliveryInteractive},
		{"bootstrap_only", MailboxDeliveryBootstrapOnly},
		{"BOOTSTRAP_ONLY", MailboxDeliveryBootstrapOnly},
		{"coordinated_restart", MailboxDeliveryCoordinatedRestart},
		{"COORDINATED_RESTART", MailboxDeliveryCoordinatedRestart},
		{"session_resume", MailboxDeliverySessionResume},
		{"SESSION_RESUME", MailboxDeliverySessionResume},
		{"Session_Resume", MailboxDeliverySessionResume},
		{"unknown", ""},
		{"", ""},
		{"  session_resume  ", MailboxDeliverySessionResume},
	}
	for _, c := range cases {
		got := NormalizeMailboxDeliveryMode(c.raw)
		if got != c.want {
			t.Errorf("NormalizeMailboxDeliveryMode(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestPermiteFallbackInteractivoSessionResumePipelineLocal(t *testing.T) {
	got := PermiteFallbackInteractivoSessionResumePipelineLocal(
		"pipeline_local",
		"",
		"cli",
		"process",
		`{"driver":"process_pty_cli","stdin_path":"/tmp/codex.stdin","mailbox_delivery_mode":"session_resume"}`,
		`{"mailbox_delivery_mode":"session_resume"}`,
	)
	if !got {
		t.Fatal("pipeline_local process_pty_cli con stdin_path y session_resume deberia permitir fallback interactivo")
	}
}

func TestPermiteFallbackInteractivoSessionResumePipelineLocalLegacyStdinRawPath(t *testing.T) {
	got := PermiteFallbackInteractivoSessionResumePipelineLocal(
		"pipeline_local",
		"",
		"cli",
		"process",
		`{"driver":"process_pty_cli","stdin_raw_path":"/tmp/codex.stdin.raw","mailbox_delivery_mode":"session_resume"}`,
		`{"mailbox_delivery_mode":"session_resume"}`,
	)
	if !got {
		t.Fatal("stdin_raw_path legacy tambien deberia habilitar fallback interactivo")
	}
}

func TestPermiteFallbackInteractivoSessionResumePipelineLocalSoloSupervisorRef(t *testing.T) {
	got := PermiteFallbackInteractivoSessionResumePipelineLocal(
		"pipeline_local",
		"",
		"cli",
		"process",
		`{"driver":"process_pty_cli","supervisor_ref":"codex1-supervisor","mailbox_delivery_mode":"session_resume"}`,
		`{"mailbox_delivery_mode":"session_resume"}`,
	)
	if !got {
		t.Fatal("supervisor_ref sin stdin_path tambien deberia habilitar fallback interactivo")
	}
}

func TestPermiteFallbackInteractivoSessionResumePipelineLocalBloqueaSesionExterna(t *testing.T) {
	got := PermiteFallbackInteractivoSessionResumePipelineLocal(
		"pipeline_local",
		"sess-1",
		"cli",
		"process",
		`{"driver":"process_pty_cli","stdin_path":"/tmp/codex.stdin","mailbox_delivery_mode":"session_resume"}`,
		`{"mailbox_delivery_mode":"session_resume"}`,
	)
	if got {
		t.Fatal("con external_session_id no deberia permitir fallback interactivo local")
	}
}

func TestDefaultMailboxDeliveryModePorCanSendInput(t *testing.T) {
	f := false
	tr := true
	cases := []struct {
		canSendInput *bool
		want         string
	}{
		{nil, MailboxDeliveryInteractive},
		{&tr, MailboxDeliveryInteractive},
		{&f, MailboxDeliveryBootstrapOnly},
	}
	for _, c := range cases {
		got := defaultMailboxDeliveryMode(c.canSendInput)
		if got != c.want {
			t.Errorf("defaultMailboxDeliveryMode(%v) = %q, want %q", c.canSendInput, got, c.want)
		}
	}
}

func TestApplyResumeDeliveryHintsNoCorrigeConSendInputActivo(t *testing.T) {
	tr := true
	plan := &LaunchPlan{
		MailboxDeliveryMode: MailboxDeliveryInteractive,
		CanSendInput:        &tr,
	}
	applyResumeDeliveryHints(plan, LaunchRequest{
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{ExternalSessionID: "sess-active"},
	})
	if plan.MailboxDeliveryMode != MailboxDeliveryInteractive {
		t.Fatalf("no deberia corregir mailbox_delivery_mode cuando can_send_input es true: %s", plan.MailboxDeliveryMode)
	}
}

func TestApplyResumeDeliveryHintsNoCorrigeSinSesionExterna(t *testing.T) {
	f := false
	plan := &LaunchPlan{
		MailboxDeliveryMode: MailboxDeliveryBootstrapOnly,
		CanSendInput:        &f,
	}
	applyResumeDeliveryHints(plan, LaunchRequest{
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{ExternalSessionID: ""},
	})
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("no deberia corregir mailbox_delivery_mode sin external_session_id: %s", plan.MailboxDeliveryMode)
	}
}

func TestApplyResumeDeliveryHintsNoCorrigeConectorNoCodex(t *testing.T) {
	f := false
	plan := &LaunchPlan{
		MailboxDeliveryMode: MailboxDeliveryBootstrapOnly,
		CanSendInput:        &f,
	}
	applyResumeDeliveryHints(plan, LaunchRequest{
		Conector: ConnectorConfig{
			Slug:       "gemini-cli",
			Transporte: "cli",
			Comando:    "gemini",
		},
		Resume: ResumeContext{ExternalSessionID: "sess-gemini"},
	})
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("no deberia corregir mailbox_delivery_mode para conector no-codex: %s", plan.MailboxDeliveryMode)
	}
}

func TestApplyResumeDeliveryHintsNoCorrigeSessionResumePrevio(t *testing.T) {
	f := false
	plan := &LaunchPlan{
		MailboxDeliveryMode: MailboxDeliverySessionResume,
		CanSendInput:        &f,
	}
	applyResumeDeliveryHints(plan, LaunchRequest{
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{ExternalSessionID: "sess-codex-ya-resume"},
	})
	if plan.MailboxDeliveryMode != MailboxDeliverySessionResume {
		t.Fatalf("no deberia modificar session_resume ya establecido: %s", plan.MailboxDeliveryMode)
	}
}

func TestPrepareCLICodexRespetaSandboxYApprovalExplicitos(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
			ArgsJSON:   `["--sandbox","workspace-write","--ask-for-approval","on-request"]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	rendered := RenderCommand(plan)
	if strings.Count(rendered, "'--sandbox'") != 1 || strings.Count(rendered, "'--ask-for-approval'") != 1 {
		t.Fatalf("no deberia duplicar sandbox/approval cuando ya vienen definidos: %s", rendered)
	}
	if !strings.Contains(rendered, "'workspace-write'") || !strings.Contains(rendered, "'on-request'") {
		t.Fatalf("deberia respetar sandbox/approval explicitos: %s", rendered)
	}
}
