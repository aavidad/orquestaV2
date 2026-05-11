package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestStartAppDirectorV0ConsumesDecisionFileSourceAndStartsWorkflowAgent(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	request := validDecisionFileStartAppDirectorRequestForTestV0()
	decisionPath := serviceWriteDecisionFileForTestV0(t, request.RunRef)
	descriptorProvider := &serviceDecisionFileDescriptorProviderForTestV0{
		Descriptors: []orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0{{
			DescriptorRef: "descriptor-ref-app-director-file-source-001",
			RunID:         request.RunRef,
			Path:          decisionPath,
		}},
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:       store,
			EventSink:      sink,
			OutboxLedger:   ledger,
			DeliverySource: serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource: orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0{
				DescriptorProvider: descriptorProvider,
				Reader:             orquestadirectoragentfilesource.OSDirectorAgentDecisionFileReaderV0{},
			},
			DirectorTaskStore: taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if descriptorProvider.Calls == 0 {
		t.Fatalf("decision file source no fue consultado")
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	taskRef := "task-ref-agenda-autonomy-001"
	if !serviceStringInSetV0(result.Run.Tasks, taskRef) {
		t.Fatalf("tasks=%v missing=%s", result.Run.Tasks, taskRef)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), request.RunRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TaskID != taskRef {
		t.Fatalf("stored_tasks=%+v", tasks)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
		pendingRefs := servicePendingOutboxRefsForTestV0(t, ledger, result.Run.RunID)
		t.Fatalf("started_agents=%v missing=%s pending=%v", result.Run.StartedAgents, agentRef, pendingRefs)
	}
}

func validDecisionFileStartAppDirectorRequestForTestV0() StartAppDirectorRequestV0 {
	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = "run-app-director-service-file-source-001"
	request.ProjectRef = "project-app-director-service-file-source-001"
	request.CorrelationID = "corr-app-director-service-file-source-001"
	request.AppSpecRequest.RequestID = "request-ref-app-director-service-file-source-001"
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8
	return request
}

func serviceWriteDecisionFileForTestV0(t *testing.T, runRef string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "director-decisions.json")
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions: serviceDirectorPlanDecisionsForTestV0(orquestacoreworkflow.OrchestrationRunV0{
			RunID:       runRef,
			Brainstorms: []string{"brainstorm-agenda-director"},
		}),
	})
	if err != nil {
		t.Fatalf("Marshal decision file: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

type serviceDecisionFileDescriptorProviderForTestV0 struct {
	Descriptors []orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0
	Calls       int
}

func (provider *serviceDecisionFileDescriptorProviderForTestV0) ListDirectorAgentDecisionFilesV0(
	ctx context.Context,
	_ orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) ([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider.Calls++
	if provider.Calls > 1 {
		return nil, nil
	}
	return append([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0(nil), provider.Descriptors...), nil
}
