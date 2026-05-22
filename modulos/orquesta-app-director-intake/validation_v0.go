package orquestaappdirectorintake

import (
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validatePrepareAppDirectorIntakeRequestV0(
	request PrepareAppDirectorInputRequestV0,
) error {
	if strings.TrimSpace(request.RunRef) == "" {
		return AppDirectorIntakeIssueV0{Field: "run_ref"}
	}
	if strings.TrimSpace(request.ProjectRef) == "" {
		return AppDirectorIntakeIssueV0{Field: "project_ref"}
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		return AppDirectorIntakeIssueV0{Field: "occurred_at"}
	}
	return validateDirectorIntakeAppSpecV0(request.AppSpec)
}

func validateDirectorIntakeAppSpecV0(spec AppDirectorInputSpecV0) error {
	if strings.TrimSpace(spec.SchemaVersion) != AppDirectorInputSpecSchemaVersionV0 {
		return AppDirectorIntakeIssueV0{Field: "app_spec.schema_version"}
	}
	if strings.TrimSpace(spec.SpecID) == "" {
		return AppDirectorIntakeIssueV0{Field: "app_spec.spec_id"}
	}
	if strings.TrimSpace(spec.App.Slug) == "" {
		return AppDirectorIntakeIssueV0{Field: "app_spec.app.slug"}
	}
	if strings.TrimSpace(spec.CreatedAt) == "" {
		return AppDirectorIntakeIssueV0{Field: "app_spec.created_at"}
	}
	if strings.TrimSpace(spec.Validation.Estado) != "valida" {
		return AppDirectorIntakeIssueV0{Field: "app_spec.validation.estado"}
	}
	return nil
}

func validateAppDirectorTaskV0(task AppDirectorTaskV0) error {
	required := map[string]string{
		"task_ref":         task.TaskRef,
		"topic_ref":        task.TopicRef,
		"brainstorm_ref":   task.BrainstormRef,
		"claim_ref":        task.ClaimRef,
		"capacity_ref":     task.CapacityRef,
		"agent_request_id": task.AgentRequestID,
		"role":             task.Role,
		"summary":          task.Summary,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return AppDirectorIntakeIssueV0{Field: "director_task." + field}
		}
	}
	if task.PhaseID != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		return AppDirectorIntakeIssueV0{Field: "director_task.phase_id"}
	}
	if task.Capacity != orquestacoreworkflow.OrchestrationCapacityHighV0 &&
		task.Capacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		return AppDirectorIntakeIssueV0{Field: "director_task.capacity"}
	}
	if len(task.WriteSet) == 0 {
		return AppDirectorIntakeIssueV0{Field: "director_task.write_set"}
	}
	if _, issues := orquestacoreconcurrency.NormalizeScopeRefsV0(task.WriteSet); len(issues) > 0 {
		return AppDirectorIntakeIssueV0{Field: "director_task.write_set"}
	}
	if len(task.EvidenceRefs) == 0 {
		return AppDirectorIntakeIssueV0{Field: "director_task.evidence_refs"}
	}
	return nil
}

func validateAppDirectorTasksV0(tasks []AppDirectorTaskV0) error {
	if len(tasks) == 0 {
		return AppDirectorIntakeIssueV0{Field: "director_tasks"}
	}
	seen := map[string]bool{}
	for _, task := range tasks {
		if err := validateAppDirectorTaskV0(task); err != nil {
			return err
		}
		for _, ref := range []string{task.TaskRef, task.BrainstormRef, task.ClaimRef, task.CapacityRef, task.AgentRequestID} {
			if seen[ref] {
				return AppDirectorIntakeIssueV0{Field: "director_tasks.refs"}
			}
			seen[ref] = true
		}
	}
	return nil
}
