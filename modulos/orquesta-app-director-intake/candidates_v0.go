package orquestaappdirectorintake

import (
	"context"
	"fmt"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

type AppDirectorCandidateProviderV0 struct {
	Task        AppDirectorTaskV0
	Tasks       []AppDirectorTaskV0
	RequestedBy string
}

const maxAppDirectorWorkCandidatesPerTickV0 = 4

var _ orquestacionnucleoapp.CandidateProviderPortV0 = AppDirectorCandidateProviderV0{}

func (provider AppDirectorCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
) (orquestacionnucleoapp.SchedulerCandidateSetV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
	}
	tasks := provider.tasksV0()
	if err := validateAppDirectorTasksV0(tasks); err != nil {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
	}
	workCandidates := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, len(tasks))
	evidenceRefs := []string{"evidence-ref-app-director-candidate-v0"}
	for _, task := range tasks {
		if !provider.taskReadyV0(request.Run, task) {
			continue
		}
		candidate, err := provider.workCandidateV0(request, task)
		if err != nil {
			return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
		}
		workCandidates = append(workCandidates, candidate)
		evidenceRefs = append(evidenceRefs, task.EvidenceRefs...)
		if len(workCandidates) == maxAppDirectorWorkCandidatesPerTickV0 {
			break
		}
	}
	if len(workCandidates) == 0 {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{},
			EvidenceRefs:   []string{"evidence-ref-app-director-noop-v0"},
		}, nil
	}
	return orquestacionnucleoapp.SchedulerCandidateSetV0{
		WorkCandidates: workCandidates,
		EvidenceRefs:   compactDirectorIntakeStringsV0(evidenceRefs),
	}, nil
}

func (provider AppDirectorCandidateProviderV0) taskReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task AppDirectorTaskV0,
) bool {
	return run.CurrentPhase == task.PhaseID &&
		directorIntakeStringInSetV0(run.Brainstorms, task.BrainstormRef) &&
		!directorIntakeStringInSetV0(run.Agents, task.AgentRequestID) &&
		!directorIntakeStringInSetV0(run.StartedAgents, task.AgentRequestID) &&
		!directorIntakeStringInSetV0(run.FailedAgents, task.AgentRequestID) &&
		!directorIntakeStringInSetV0(run.StoppedAgents, task.AgentRequestID)
}

func (provider AppDirectorCandidateProviderV0) workCandidateV0(
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
	task AppDirectorTaskV0,
) (orquestadirectorscheduler.SchedulableWorkCandidateV0, error) {
	claim, err := claimForDirectorTaskV0(task, request.Run.RunID)
	if err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, err
	}
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-" + task.TaskRef,
		SubjectClaimRefs: []string{task.ClaimRef},
		Claims:           []orquestacoreconcurrency.WorksetClaimV0{claim},
		CapacityCandidate: &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
			CommandMeta: provider.commandMetaV0(request, task, "capacity"),
			Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
				CapacityRequestID:          task.CapacityRef,
				PhaseID:                    string(task.PhaseID),
				TaskRef:                    task.TaskRef,
				ReasonCode:                 "app_director_ready",
				Summary:                    "Capacidad para director de app.",
				MinimumRecommendedCapacity: task.Capacity,
				EvidenceRefs:               task.EvidenceRefs,
			},
		},
		AgentCandidate: &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
			ClaimRef:    task.ClaimRef,
			CommandMeta: provider.commandMetaV0(request, task, "agent"),
			Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
				AgentRequestID:     task.AgentRequestID,
				PhaseID:            string(task.PhaseID),
				TaskRef:            task.TaskRef,
				CapacityRequestRef: task.CapacityRef,
				Role:               task.Role,
				Summary:            task.Summary,
				EvidenceRefs:       task.EvidenceRefs,
			},
		},
		GateCommandMeta:  provider.commandMetaV0(request, task, "gate"),
		GateEvidenceRefs: task.EvidenceRefs,
		EvidenceRefs:     task.EvidenceRefs,
	}, nil
}

func claimForDirectorTaskV0(
	task AppDirectorTaskV0,
	runRef string,
) (orquestacoreconcurrency.WorksetClaimV0, error) {
	refs, issues := orquestacoreconcurrency.NormalizeScopeRefsV0(task.WriteSet)
	if len(issues) > 0 {
		return orquestacoreconcurrency.WorksetClaimV0{}, fmt.Errorf("write_set invalido")
	}
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       task.ClaimRef,
		RunRef:         runRef,
		TaskRef:        task.TaskRef,
		GroupRef:       "group-" + task.TaskRef,
		AgentRequestID: task.AgentRequestID,
		WriteSet:       refs,
		EvidenceRefs:   task.EvidenceRefs,
	}, nil
}

func (provider AppDirectorCandidateProviderV0) commandMetaV0(
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
	task AppDirectorTaskV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref := safeDirectorIntakeRefPartV0(kind + "-" + task.TaskRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-director-" + ref,
		RunID:          request.Run.RunID,
		IdempotencyKey: "idem-app-director-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    provider.requestedByV0(),
		OccurredAt:     request.OccurredAt,
	}
}

func (provider AppDirectorCandidateProviderV0) tasksV0() []AppDirectorTaskV0 {
	if len(provider.Tasks) > 0 {
		return append([]AppDirectorTaskV0(nil), provider.Tasks...)
	}
	return []AppDirectorTaskV0{provider.Task}
}

func (provider AppDirectorCandidateProviderV0) requestedByV0() string {
	if provider.RequestedBy != "" {
		return provider.RequestedBy
	}
	return "orquesta-app-director-intake"
}

func directorIntakeStringInSetV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
