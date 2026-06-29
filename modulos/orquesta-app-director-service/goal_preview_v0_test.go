package orquestaappdirectorservice

import (
	"context"
	"reflect"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestBuildStartAppDirectorGoalWorkPreviewV0CompilaGoalSinPuertos(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = ""

	preview, err := BuildStartAppDirectorGoalWorkPreviewV0(context.Background(), request)

	if err != nil {
		t.Fatalf("BuildStartAppDirectorGoalWorkPreviewV0: %v", err)
	}
	if preview.Status != StartAppDirectorStatusPreviewReadyV0 ||
		preview.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 ||
		preview.GoalSpec.GoalRef == "" ||
		preview.GoalSpec.RunRef != preview.Run.RunID {
		t.Fatalf("preview=%+v", preview)
	}
	if len(preview.WriteSet) != 1 || preview.WriteSet[0] != "generated-apps/agenda" {
		t.Fatalf("write_set=%v goal=%+v", preview.WriteSet, preview.GoalSpec.WriteSet)
	}
	if len(preview.RequiredTests) != 1 || preview.Estimate.TokenBudget == 0 || preview.Estimate.CostTier == "" {
		t.Fatalf("tests=%v estimate=%+v", preview.RequiredTests, preview.Estimate)
	}
	if issues := orquestagoal.ValidateGoalWorkSpecV0(preview.GoalSpec); len(issues) > 0 {
		t.Fatalf("goal spec invalido: %+v", issues)
	}
	hasDeepManualCriterion := false
	for _, criterion := range preview.GoalSpec.AcceptanceCriteria {
		if criterion == "Documentacion: usuario=true; desarrollo=true; sistemas=true; profundidad=profunda. Si profundidad=profunda, entregar manuales de usuario, desarrollo y sistemas con flujos, comandos, criterios de aceptacion y operacion." {
			hasDeepManualCriterion = true
			break
		}
	}
	if !hasDeepManualCriterion {
		t.Fatalf("goal spec no transporta manual profundo: %+v", preview.GoalSpec.AcceptanceCriteria)
	}
}

func TestBuildStartAppDirectorGoalWorkPreviewV0CoincideConSpecLanzada(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	preview, err := BuildStartAppDirectorGoalWorkPreviewV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildStartAppDirectorGoalWorkPreviewV0: %v", err)
	}
	_, err = StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:             store,
		EventSink:            sink,
		GoalLauncher:         launcher,
		GoalObserver:         serviceGoalObserverForTestV0{},
		GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:       newServiceGoalStateStoreForTestV0(),
	})
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("launcher specs=%d", len(launcher.specs))
	}
	if !reflect.DeepEqual(preview.GoalSpec, launcher.specs[0]) {
		t.Fatalf("preview spec no coincide\npreview=%+v\nlaunched=%+v", preview.GoalSpec, launcher.specs[0])
	}
}

func TestBuildStartAppDirectorGoalWorkPreviewV0DevuelveIssuesSinLanzar(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.AppSpecRequest.Locale = "locale invalido"

	preview, err := BuildStartAppDirectorGoalWorkPreviewV0(context.Background(), request)

	if err != nil {
		t.Fatalf("request invalida no debe ser error de transporte: %v", err)
	}
	if preview.Status != StartAppDirectorStatusInvalidV0 || len(preview.ValidationIssues) == 0 {
		t.Fatalf("preview=%+v", preview)
	}
	if preview.Run.RunID != "" || preview.GoalSpec.GoalRef != "" {
		t.Fatalf("preview invalido no debe preparar run ni goal: %+v", preview)
	}
}
