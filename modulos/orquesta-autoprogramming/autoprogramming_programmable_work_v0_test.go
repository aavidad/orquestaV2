package orquestaautoprogramming

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildAutoprogrammingProgrammableWorkV0BuildsWorkflowTaskForSingleGroup(t *testing.T) {
	request := validAutoprogrammingRequestV0(nil)

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Groups) != 1 || len(result.Work.Profiles) != 1 || len(result.Work.Tasks) != 1 {
		t.Fatalf("work=%+v", result.Work)
	}
	task := result.Work.Tasks[0]
	if task.WorkProfileKind != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%q", task.WorkProfileKind)
	}
	if !reflect.DeepEqual(task.WriteSet, result.Work.Groups[0].WriteSet) ||
		!reflect.DeepEqual(task.WriteSet, []string{
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0.go",
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0_test.go",
		}) {
		t.Fatalf("write_set=%v", task.WriteSet)
	}
	if !reflect.DeepEqual(task.RequiredTests, request.RequiredTests) {
		t.Fatalf("required_tests=%v want %v", task.RequiredTests, request.RequiredTests)
	}
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "worktree_ref:"+request.WorktreeRef)
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "branch_ref:"+request.BranchRef)
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-autoprogramming-a")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-autoprogramming-b")
}

func TestBuildAutoprogrammingProgrammableWorkV0SetsTenBySixDelegationBudget(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= AutoprogrammingRequestDefaultMaxTaskRefsV0; i++ {
			area := fmt.Sprintf("parent-wave-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-parent-wave-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contrato.go",
			)
		}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Tasks) != AutoprogrammingRequestDefaultMaxTaskRefsV0 {
		t.Fatalf("tasks=%d", len(result.Work.Tasks))
	}
	for _, task := range result.Work.Tasks {
		if task.MaxDelegationDepth != AutoprogrammingDefaultMaxDelegationDepthV0 ||
			task.MaxChildAgents != AutoprogrammingDefaultMaxSubagentsPerAgentV0 ||
			task.MaxSubagentsPerAgent != AutoprogrammingDefaultMaxSubagentsPerAgentV0 ||
			task.MaxRecursiveAgents != AutoprogrammingDefaultMaxRecursiveAgentsV0 {
			t.Fatalf("delegation budget inesperado: %+v", task)
		}
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0PreservaDelegationBudgetExplicito(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.MaxDelegationDepth = 2
		request.MaxSubagentsPerAgent = 4
		request.MaxRecursiveAgents = 40
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	if task.MaxDelegationDepth != 2 ||
		task.MaxChildAgents != 4 ||
		task.MaxSubagentsPerAgent != 4 ||
		task.MaxRecursiveAgents != 40 {
		t.Fatalf("delegation budget explicito no preservado: %+v", task)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0TransportaContratoExplicitoDeTarea(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "task-ref-explicita-001",
			Area:               "Autoprogramming",
			Title:              "Cambio acotado 01",
			Objective:          "Transportar objetivo y contexto sin inferir por task_ref.",
			Context:            []string{"Paquete de agente ya trae criterios y reglas compactas."},
			ContextRefs:        []string{"doc-ref:autoprog-t03"},
			AcceptanceCriteria: []string{"objetivo, contexto y criterios llegan al worker"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-autoprogramming -run TestBuildAutoprogramming"},
			CompactRules:       []string{"tratar worktree_ref y branch_ref como refs opacas"},
		}}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/autoprogramming_programmable_work_v0.go",
		}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	if task.Title != "Cambio acotado 01" ||
		!stringsContainForAutoprogrammingTestV0(task.Summary, "Transportar objetivo") ||
		!stringsContainForAutoprogrammingTestV0(task.Summary, "Autoprogramacion acotada") {
		t.Fatalf("task summary/title=%+v", task)
	}
	for _, want := range []string{
		"Objetivo task-ref-explicita-001: Transportar objetivo",
		"Contexto task-ref-explicita-001: Paquete de agente",
		"Regla compacta task-ref-explicita-001: tratar worktree_ref",
	} {
		if !stringsContainForAutoprogrammingTestV0(task.Summary+"\n"+result.Work.Profiles[0].Objective, want) {
			t.Fatalf("objective no contiene %q: %+v", want, result.Work.Profiles[0])
		}
	}
	for _, want := range []string{
		"Criterio task-ref-explicita-001: objetivo, contexto y criterios llegan al worker",
		"Regla compacta task-ref-explicita-001: tratar worktree_ref",
	} {
		if !stringsSliceContainsSubstringForAutoprogrammingTestV0(task.AcceptanceCriteria, want) {
			t.Fatalf("criteria no contiene %q: %v", want, task.AcceptanceCriteria)
		}
	}
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-explicita-001")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "doc-ref:autoprog-t03")
	if !stringsSliceContainsForAutoprogrammingTestV0(
		task.RequiredTests,
		"go test -count=1 ./modulos/orquesta-autoprogramming -run TestBuildAutoprogramming",
	) {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0NormalizaContextRefsNoCompactos(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks[0].ContextRefs = []string{
			"doc-ref:autoprog-t29",
			"backlog_input:write_set:`modulos/orquesta-runtime`+`cmd/orquesta-server`",
			"backlog_output:tests:go test -count=1 ./cmd/orquesta-server",
		}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "doc-ref:autoprog-t29")
	assertAutoprogrammingContextRefPrefixV0(t, task.ContextRefs, "context_ref:backlog-input-")
	assertAutoprogrammingContextRefPrefixV0(t, task.ContextRefs, "context_ref:backlog-output-")
	for _, ref := range task.ContextRefs {
		if strings.ContainsAny(ref, " /\\\t\r\n") {
			t.Fatalf("context_ref no compacto: %q refs=%v", ref, task.ContextRefs)
		}
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0CompactaTextoLargoParaWorkflowTask(t *testing.T) {
	longObjective := strings.Repeat("iterar pensar tareas agentes pruebas revision cierre ", 20)
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:   "task-ref-texto-largo-001",
			Area:      "Autoprogramming",
			Title:     strings.Repeat("Ciclo residente ", 20),
			Objective: longObjective,
			Context: []string{
				strings.Repeat("contexto operativo con runtime provider db sql oauth docker tmux token budget ", 10),
				strings.Repeat("hexagonal puro y refs opacas sin producto dentro del nucleo ", 10),
			},
			ContextRefs: []string{"doc-ref:autoprogramacion-pendientes"},
			AcceptanceCriteria: []string{
				strings.Repeat("criterio de cierre con pruebas reales evidencia durable y cola residente ", 10),
				strings.Repeat("criterio de reparacion con followups y self repair sin tirar todo ", 10),
			},
			CompactRules: []string{
				strings.Repeat("regla compacta caveman y reutilizar codigo existente ", 10),
			},
		}}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/autoprogramming_task_contract_v0.go"}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	if len([]rune(task.Title)) > 160 || len([]rune(task.Summary)) > 600 {
		t.Fatalf("task text no compactado title=%d summary=%d task=%+v", len([]rune(task.Title)), len([]rune(task.Summary)), task)
	}
	for _, criterion := range task.AcceptanceCriteria {
		if len([]rune(criterion)) > 600 {
			t.Fatalf("criterion demasiado largo len=%d value=%q", len([]rune(criterion)), criterion)
		}
	}
	payload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if len(payload) > 4096 {
		t.Fatalf("payload demasiado grande: %d %s", len(payload), payload)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0CompactaBacklogScanParaWorkflowTask(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequestRef = "request-ref-autoprogramming-backlog-t45-autoprogramming-go-file-line-budget-baseline-fd1e0db4"
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:   "task-ref-backlog-scan-large-001",
			Area:      "Autoprogramming",
			Title:     "Automejora en segundo plano",
			Objective: "Corregir de forma general un patron detectado en backlog documental con criterios extensos.",
			Context: []string{
				"failure_summary: convertir el limite de 300 lineas por fichero Go en contrato medible",
				"backlog_input:write_set:`modulos/orquesta-autoprogramming`+`modulos/orquesta-runtime-worktree`+`modulos/orquesta-app-codex-stack`+`cmd/orquesta-server`",
			},
			ContextRefs: []string{
				"backlog-doc-autoprogramacion-2026-05-23",
				"backlog_input:tests:go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server",
			},
			AcceptanceCriteria: []string{
				"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS por defecto dispara tras 60 segundos sin ejecuciones",
				"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 desactiva automejora idle",
				"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE",
				"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
				"usar evidencia del fallo y corregir la causa general si es posible",
				"crear una linea base de ficheros Go historicos por encima de 300 lineas y no bloquear por piezas existentes",
				"en modo estricto futuro una entrega nueva no debe cerrar completed si agranda ficheros sin particionar",
				"la medicion debe salir del snapshot real cuando exista y el ACK no debe decidirlo solo",
				"file_too_large puede seguir como advisory en modo legacy hasta tener matriz externa suficiente",
			},
			CompactRules: []string{
				"comunicacion compacta",
				"un agente padre por tarea; subagentes hasta 6 si ayudan",
				"si aparece otro hueco general, registrarlo como nueva automejora y seguir",
			},
		}}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming",
			"modulos/orquesta-runtime-worktree",
			"modulos/orquesta-app-codex-stack",
			"cmd/orquesta-server",
			"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
			"docs/rail_errors_observados_2026-05-23.md",
			"docs/duplicaciones_railes_pendientes_2026-05-24.md",
		}
		request.RequiredTests = []string{
			"go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server",
		}
		request.BacklogScan = AutoprogrammingBacklogScanV0{
			Epoch:           "backlog-scan-epoch-5f7d86dffafa",
			ReservationRefs: []string{"reservation-ref-backlog-scan-doc-merge-1a2b3c4d5e6f"},
			Documents: []AutoprogrammingBacklogDocumentV0{
				{Path: "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md", StartLine: 2115, SHA256: strings.Repeat("a", 64), SectionRef: "t45-autoprogramming-go-file-line-budget-baseline"},
				{Path: "docs/rail_errors_observados_2026-05-23.md", StartLine: 2115, SHA256: strings.Repeat("b", 64), SectionRef: "t45-autoprogramming-go-file-line-budget-baseline"},
				{Path: "docs/duplicaciones_railes_pendientes_2026-05-24.md", StartLine: 2115, SHA256: strings.Repeat("c", 64), SectionRef: "t45-autoprogramming-go-file-line-budget-baseline"},
			},
		}
		request.MaxWriteSetEntries = len(request.WriteSet)
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	payload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if len(payload) > 4096 {
		t.Fatalf("payload demasiado grande: %d %s", len(payload), payload)
	}
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-5f7d86dffafa")
	assertAutoprogrammingContextRefPrefixV0(t, task.ContextRefs, "context_ref:backlog-input-")
}

func TestEnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0PreservaMarkerDirectorOperativoV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequestRef = "request-ref-autoprogramming-marker-director-operativo-001"
		request.Tasks[0].ContextRefs = append([]string{
			"backlog-doc-autoprogramacion-2026-05-23",
			"backlog_section:t11-rails-blandos-y-falsos-positivos",
			"backlog_line:705",
			"backlog_input:write_set_modulos_orquesta_runtime_codex",
			"backlog_output:tests_go_test_count_1_all",
			"context_ref:extra-001",
			"context_ref:extra-002",
			"context_ref:extra-003",
			"context_ref:extra-004",
			"context_ref:extra-005",
			"context_ref:extra-006",
			"context_ref:extra-007",
			"context_ref:extra-008",
			"context_ref:extra-009",
			"context_ref:extra-010",
			"context_ref:extra-011",
			"context_ref:extra-012",
			"context_ref:extra-013",
			"context_ref:extra-014",
			"context_ref:extra-015",
		}, "operational_director.task_source:autoprogramming")
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "operational_director.task_source:autoprogramming")
	if len(task.ContextRefs) > autoprogrammingWorkflowTaskContextRefsMaxV0+1 {
		t.Fatalf("context_refs no compactadas: %d refs=%v", len(task.ContextRefs), task.ContextRefs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0PartitionsWriteSetByGroupArea(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-groups", Area: "Task Group"},
			{TaskRef: "task-ref-review", Area: "Review Gate"},
		}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go",
			"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go",
		}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Tasks) != 2 {
		t.Fatalf("tasks=%d", len(result.Work.Tasks))
	}
	gotByArea := map[string][]string{}
	for _, group := range result.Work.Groups {
		gotByArea[group.Area] = group.Task.WriteSet
	}
	assertStringsEqualV0(t, gotByArea["task-group"], []string{
		"modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go",
	})
	assertStringsEqualV0(t, gotByArea["review-gate"], []string{
		"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go",
	})
	assertNoWriteSetOverlapV0(t, result.Work.Tasks)
}

func TestBuildAutoprogrammingProgrammableWorkV0ConstruyeDiezPadresConLimitesExplicitos(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= 10; i++ {
			area := fmt.Sprintf("parent-wave-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-parent-wave-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contrato.go",
			)
		}
		request.MaxTaskRefs = 10
		request.MaxAreas = 10
		request.MaxWriteSetEntries = 10
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Groups) != 10 ||
		len(result.Work.Profiles) != 10 ||
		len(result.Work.Tasks) != 10 ||
		len(result.Work.Partition.Steps) != 10 {
		t.Fatalf("work=%+v", result.Work)
	}
	assertNoWriteSetOverlapV0(t, result.Work.Tasks)
	for i, task := range result.Work.Tasks {
		wantPath := fmt.Sprintf("modulos/orquesta-autoprogramming/parent-wave-%02d/contrato.go", i+1)
		if !stringsSliceContainsForAutoprogrammingTestV0(task.WriteSet, wantPath) {
			t.Fatalf("task %d write_set=%v want %s", i, task.WriteSet, wantPath)
		}
		assertAutoprogrammingContextRefV0(t, task.ContextRefs,
			fmt.Sprintf("source_task_ref:task-ref-parent-wave-%02d", i+1))
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0ConservaWriteSetNoParticionableComoReparable(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-group-a", Area: "Task Group"},
			{TaskRef: "task-ref-review-a", Area: "Review Gate"},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/README.md"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Partition.Repairs) != 1 ||
		result.Work.Partition.Repairs[0].Code != "write_set_unassigned_shared" {
		t.Fatalf("repairs=%+v", result.Work.Partition.Repairs)
	}
	for _, group := range result.Work.Groups {
		if !stringsSliceContainsForAutoprogrammingTestV0(group.WriteSet, "modulos/orquesta-autoprogramming/README.md") {
			t.Fatalf("group %s write_set=%v", group.Area, group.WriteSet)
		}
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0SequencesRepairableWriteSetOverlap(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-api", Area: "API"},
			{TaskRef: "task-ref-api-client", Area: "API Client"},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/api_client_handler.go"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Partition.Repairs) != 1 ||
		result.Work.Partition.Repairs[0].Code != "write_set_overlap_sequenced" {
		t.Fatalf("repairs=%+v", result.Work.Partition.Repairs)
	}
	if len(result.Work.Groups) != 2 || len(result.Work.Groups[1].Task.DependsOn) != 1 {
		t.Fatalf("groups=%+v", result.Work.Groups)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0NoConfundeAreaComoSubcadena(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-api", Area: "API"},
			{TaskRef: "task-ref-capital", Area: "Capital"},
		}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/api/client.go",
			"modulos/orquesta-autoprogramming/capital/report.go",
		}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	gotByArea := map[string][]string{}
	for _, group := range result.Work.Groups {
		gotByArea[group.Area] = group.Task.WriteSet
	}
	assertStringsEqualV0(t, gotByArea["api"], []string{
		"modulos/orquesta-autoprogramming/api/client.go",
	})
	assertStringsEqualV0(t, gotByArea["capital"], []string{
		"modulos/orquesta-autoprogramming/capital/report.go",
	})
}

func assertAutoprogrammingContextRefV0(t *testing.T, refs []string, want string) {
	t.Helper()
	for _, ref := range refs {
		if ref == want {
			return
		}
	}
	t.Fatalf("context ref %q no encontrado en %v", want, refs)
}

func assertAutoprogrammingContextRefPrefixV0(t *testing.T, refs []string, prefix string) {
	t.Helper()
	for _, ref := range refs {
		if strings.HasPrefix(ref, prefix) {
			return
		}
	}
	t.Fatalf("context ref prefix %q no encontrado en %v", prefix, refs)
}

func assertStringsEqualV0(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func assertNoWriteSetOverlapV0(
	t *testing.T,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) {
	t.Helper()
	seen := map[string]string{}
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			if owner := seen[path]; owner != "" {
				t.Fatalf("write_set path %q reused by %s and %s", path, owner, task.TaskID)
			}
			seen[path] = task.TaskID
		}
	}
}

func stringsContainForAutoprogrammingTestV0(value string, want string) bool {
	return strings.Contains(value, want)
}

func stringsSliceContainsForAutoprogrammingTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringsSliceContainsSubstringForAutoprogrammingTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
