package orquestaappplanner

import (
	"fmt"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
)

type AppPlannerIssueV0 struct {
	Field string `json:"field"`
}

func (issue AppPlannerIssueV0) Error() string {
	return "app_planner_invalido: " + issue.Field
}

func validateAppPlanRequestV0(request AppPlanRequestV0) error {
	if strings.TrimSpace(request.RunRef) == "" {
		return AppPlannerIssueV0{Field: "run_ref"}
	}
	if strings.TrimSpace(request.AppRef) == "" {
		return AppPlannerIssueV0{Field: "app_ref"}
	}
	if strings.TrimSpace(request.AppName) == "" {
		return AppPlannerIssueV0{Field: "app_name"}
	}
	if !request.API && !request.Web {
		return AppPlannerIssueV0{Field: "surfaces"}
	}
	if request.Scale != AppPlanScaleStandardV0 && request.Scale != AppPlanScaleLargeV0 {
		return AppPlannerIssueV0{Field: "scale"}
	}
	return nil
}

func validateAppMicrotaskPlanV0(plan AppMicrotaskPlanV0) error {
	if plan.SchemaVersion != AppMicrotaskPlanSchemaVersionV0 {
		return AppPlannerIssueV0{Field: "schema_version"}
	}
	if strings.TrimSpace(plan.RunRef) == "" || strings.TrimSpace(plan.AppRef) == "" {
		return AppPlannerIssueV0{Field: "refs"}
	}
	if len(plan.Units) == 0 {
		return AppPlannerIssueV0{Field: "units"}
	}
	for index, unit := range plan.Units {
		if err := validateAppWorkUnitV0(unit); err != nil {
			return AppPlannerIssueV0{Field: fmt.Sprintf("units[%d].%s", index, appPlannerIssueFieldV0(err))}
		}
	}
	return nil
}

func appPlannerIssueFieldV0(err error) string {
	if issue, ok := err.(AppPlannerIssueV0); ok {
		return issue.Field
	}
	return err.Error()
}

func validateAppWorkUnitV0(unit AppWorkUnitV0) error {
	required := map[string]string{
		"task_ref":         unit.TaskRef,
		"claim_ref":        unit.ClaimRef,
		"agent_request_id": unit.AgentRequestID,
		"delivery_ref":     unit.DeliveryRef,
		"role":             unit.Role,
		"title":            unit.Title,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return AppPlannerIssueV0{Field: field}
		}
	}
	if len(unit.WriteSet) == 0 || len(unit.AcceptanceCriteria) == 0 {
		return AppPlannerIssueV0{Field: "work"}
	}
	for _, value := range unit.WriteSet {
		if _, issues := orquestacoreconcurrency.NewScopeRefV0(value); len(issues) > 0 {
			return AppPlannerIssueV0{Field: "write_set"}
		}
	}
	return nil
}
