package orquestaappplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildGoAPIWebMicrotaskPlanV0DivideAppEnCortesPequenos(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	if plan.SchemaVersion != AppMicrotaskPlanSchemaVersionV0 {
		t.Fatalf("schema=%s", plan.SchemaVersion)
	}
	if len(plan.Units) != 6 {
		t.Fatalf("units=%d", len(plan.Units))
	}
	assertAppUnitForTestV0(t, plan, "bootstrap", nil, []string{
		"go.mod",
		"AGENTS.md",
		"README.md",
		"docs/contratos.md",
		"docs/tareas.md",
		"docs/pruebas.md",
		"docs/decisiones.md",
	})
	assertAppUnitForTestV0(t, plan, "agenda-core", []string{"ack-agenda-bootstrap"}, []string{"internal/agenda"})
	assertAppUnitForTestV0(t, plan, "web", []string{"ack-agenda-bootstrap"}, []string{"web"})
	assertAppUnitForTestV0(t, plan, "api", []string{"ack-agenda-agenda-core", "ack-agenda-web"}, []string{"cmd/server"})
	assertAppUnitForTestV0(t, plan, "docs", []string{"ack-agenda-api"}, []string{"README.md"})
	assertAppUnitForTestV0(t, plan, "review", []string{"ack-agenda-docs"}, []string{"docs/revision.md"})
}

func TestRuntimeFunctionContractForUnitV0ConservaCorteVerificable(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "api")

	contract := RuntimeFunctionContractForUnitV0(unit)
	if contract.ContractRef == "" ||
		contract.Titulo != unit.Title ||
		contract.Objetivo != unit.Summary {
		t.Fatalf("contract=%+v unit=%+v", contract, unit)
	}
	if len(contract.WriteSet) != 1 || contract.WriteSet[0] != "cmd/server" {
		t.Fatalf("write_set=%v", contract.WriteSet)
	}
	if len(contract.TestsObligatorios) != 1 || contract.TestsObligatorios[0] != "go test ./..." {
		t.Fatalf("tests=%v", contract.TestsObligatorios)
	}
}

func mustAppPlanForTestV0(t *testing.T) AppMicrotaskPlanV0 {
	t.Helper()
	plan, err := BuildGoAPIWebMicrotaskPlanV0(AppPlanRequestV0{
		RunRef:  "run-ref-app-001",
		AppRef:  "agenda",
		AppName: "Agenda",
		API:     true,
		Web:     true,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0: %v", err)
	}
	return plan
}

func assertAppUnitForTestV0(
	t *testing.T,
	plan AppMicrotaskPlanV0,
	key string,
	wantDeps []string,
	wantWriteSet []string,
) {
	t.Helper()
	unit := appUnitByKeyForTestV0(t, plan, key)
	if unit.PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("%s phase=%s", key, unit.PhaseID)
	}
	if !sameStringSetForTestV0(unit.DependsOnDeliveries, wantDeps) {
		t.Fatalf("%s deps=%v want=%v", key, unit.DependsOnDeliveries, wantDeps)
	}
	if !sameStringSetForTestV0(unit.WriteSet, wantWriteSet) {
		t.Fatalf("%s write_set=%v want=%v", key, unit.WriteSet, wantWriteSet)
	}
	if len(unit.AcceptanceCriteria) == 0 || unit.AgentRequestID == "" || unit.DeliveryRef == "" {
		t.Fatalf("%s incompleta: %+v", key, unit)
	}
}

func appUnitByKeyForTestV0(t *testing.T, plan AppMicrotaskPlanV0, key string) AppWorkUnitV0 {
	t.Helper()
	wantTask := taskRefV0(AppPlanRequestV0{AppRef: plan.AppRef}, key)
	for _, unit := range plan.Units {
		if unit.TaskRef == wantTask {
			return unit
		}
	}
	t.Fatalf("unit %s no encontrada", key)
	return AppWorkUnitV0{}
}

func sameStringSetForTestV0(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
