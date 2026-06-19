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
	assertAppUnitForTestV0(t, plan, "bootstrap", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, nil, []string{
		"go.mod",
		"AGENTS.md",
		"README.md",
		"docs/contratos.md",
		"docs/tareas.md",
		"docs/pruebas.md",
		"docs/decisiones.md",
	})
	assertAppUnitForTestV0(t, plan, "agenda-core", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-agenda-bootstrap"}, []string{"internal/domain", "internal/application", "internal/ports"})
	assertAppUnitForTestV0(t, plan, "web", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-agenda-bootstrap"}, []string{"web"})
	assertAppUnitForTestV0(t, plan, "api", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-agenda-agenda-core", "ack-agenda-web"}, []string{"internal/adapters/http", "internal/app/bootstrap", "cmd/server"})
	assertAppUnitForTestV0(t, plan, "docs", orquestacoreworkflow.OrchestrationPhaseDocumentacionV0, []string{"ack-agenda-api"}, []string{"README.md"})
	assertAppUnitForTestV0(t, plan, "review", orquestacoreworkflow.OrchestrationPhaseRevisionV0, []string{"ack-agenda-docs"}, []string{"docs/revision.md"})
}

func TestBuildGoAPIWebMicrotaskPlanV0ExigeHexagonalidadEstructural(t *testing.T) {
	plan := mustAppPlanForTestV0(t)

	bootstrap := appUnitByKeyForTestV0(t, plan, "bootstrap")
	assertAppUnitCriteriaContainsForTestV0(t, bootstrap, "AGENTS.md y docs/contratos.md declaran arquitectura hexagonal estricta como condicion de aceptacion.")
	assertAppUnitCriteriaContainsForTestV0(t, bootstrap, "docs/contratos.md separa domain, application, ports, adapters y bootstrap.")

	domain := appUnitByKeyForTestV0(t, plan, "agenda-core")
	assertAppUnitCriteriaContainsForTestV0(t, domain, "Dominio probado sin imports de adapters, HTTP, DB, filesystem, runtime ni UI.")
	assertAppUnitCriteriaContainsForTestV0(t, domain, "Casos de uso dependen de puertos/interfaces, no de repositorios concretos.")

	api := appUnitByKeyForTestV0(t, plan, "api")
	assertAppUnitCriteriaContainsForTestV0(t, api, "Entrypoint bajo cmd/server y composicion bajo internal/app/bootstrap.")
	assertAppUnitCriteriaContainsForTestV0(t, api, "Handlers finos: no construyen repositorios, autenticacion, fixtures ni reglas de negocio.")

	review := appUnitByKeyForTestV0(t, plan, "review")
	assertAppUnitCriteriaContainsForTestV0(t, review, "La revision no acepta la app si dominio/application importan adapters, HTTP, DB, filesystem, runtime o UI.")
	assertAppUnitCriteriaContainsForTestV0(t, review, "La revision no acepta handlers con composicion de repositorios, autenticacion, fixtures o reglas de negocio.")
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
	if !sameStringSetForTestV0(contract.WriteSet, []string{"internal/adapters/http", "internal/app/bootstrap", "cmd/server"}) {
		t.Fatalf("write_set=%v", contract.WriteSet)
	}
	if len(contract.TestsObligatorios) != 1 || contract.TestsObligatorios[0] != "go test ./..." {
		t.Fatalf("tests=%v", contract.TestsObligatorios)
	}
	if !appPlannerStringInSetV0(contract.CriterioCierre, "Handlers finos: no construyen repositorios, autenticacion, fixtures ni reglas de negocio.") {
		t.Fatalf("criterio_cierre=%v", contract.CriterioCierre)
	}
}

func TestWorkProfileForUnitV0PropagaSkillRefsDeclaradas(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "web")
	unit.SkillRefs = []string{"skill-ref-catalogo-declarado-v0"}

	profile, err := WorkProfileForUnitV0(plan, unit)
	if err != nil {
		t.Fatalf("WorkProfileForUnitV0: %v", err)
	}
	if !appPlannerStringInSetV0(profile.SkillRefs, "skill-ref-catalogo-declarado-v0") {
		t.Fatalf("profile skill_refs=%v", profile.SkillRefs)
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
	wantPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
	wantDeps []string,
	wantWriteSet []string,
) {
	t.Helper()
	unit := appUnitByKeyForTestV0(t, plan, key)
	if unit.PhaseID != wantPhase {
		t.Fatalf("%s phase=%s want=%s", key, unit.PhaseID, wantPhase)
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

func assertAppUnitCriteriaContainsForTestV0(t *testing.T, unit AppWorkUnitV0, want string) {
	t.Helper()
	for _, criterion := range unit.AcceptanceCriteria {
		if criterion == want {
			return
		}
	}
	t.Fatalf("%s missing criterion %q in %+v", unit.TaskRef, want, unit.AcceptanceCriteria)
}
