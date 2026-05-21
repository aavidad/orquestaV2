package orquestaappplanner

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func WorkProfileForUnitV0(plan AppMicrotaskPlanV0, unit AppWorkUnitV0) (orquestacoreworkflow.WorkProfileV0, error) {
	if err := validateAppMicrotaskPlanV0(plan); err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, err
	}
	if err := validateAppWorkUnitV0(unit); err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, err
	}
	planUnit, ok := findAppPlanUnitByTaskRefV0(plan, unit.TaskRef)
	if !ok {
		return orquestacoreworkflow.WorkProfileV0{}, AppPlannerIssueV0{Field: "task_ref"}
	}
	if planUnit.DeliveryRef != unit.DeliveryRef {
		return orquestacoreworkflow.WorkProfileV0{}, AppPlannerIssueV0{Field: "delivery_ref"}
	}
	dependsOn, err := appPlanTaskRefsForDeliveryDepsV0(plan, unit)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, err
	}
	profile, err := orquestacoreworkflow.NewWorkProfileV0(orquestacoreworkflow.WorkProfileV0{
		SchemaVersion:        orquestacoreworkflow.WorkProfileSchemaVersionV0,
		ProfileRef:           "profile-" + unit.TaskRef,
		ProfileKind:          appWorkProfileKindForUnitV0(unit),
		TaskRef:              unit.TaskRef,
		RunRef:               plan.RunRef,
		PhaseID:              unit.PhaseID,
		Title:                unit.Title,
		Objective:            unit.Summary,
		Summary:              unit.Summary,
		ScopeRefs:            append([]string(nil), unit.WriteSet...),
		AcceptanceCriteria:   append([]string(nil), unit.AcceptanceCriteria...),
		RequiredTests:        append([]string(nil), unit.RequiredTests...),
		DependsOn:            dependsOn,
		FunctionContractRefs: appPlanFunctionContractRefsForUnitV0(unit),
	})
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, err
	}
	return profile, nil
}

func WorkflowTaskForUnitV0(plan AppMicrotaskPlanV0, unit AppWorkUnitV0) (orquestacoreworkflow.WorkflowTaskV0, error) {
	profile, err := WorkProfileForUnitV0(plan, unit)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, err
	}
	return orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(profile)
}

func appWorkProfileKindForUnitV0(unit AppWorkUnitV0) orquestacoreworkflow.WorkProfileKindV0 {
	if unit.WorkProfileKind != "" {
		return orquestacoreworkflow.NormalizeWorkProfileKindV0(unit.WorkProfileKind)
	}
	return appWorkProfileKindForRoleV0(unit.Role)
}

func appPlanFunctionContractRefsForUnitV0(unit AppWorkUnitV0) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	return []orquestacoreworkflow.WorkflowFunctionContractRefV0{
		{ContractRef: "contract-" + unit.TaskRef},
	}
}

func appPlanTaskRefsForDeliveryDepsV0(plan AppMicrotaskPlanV0, unit AppWorkUnitV0) ([]string, error) {
	if len(unit.DependsOnDeliveries) == 0 {
		return nil, nil
	}
	byDeliveryRef := make(map[string]string, len(plan.Units))
	for _, candidate := range plan.Units {
		byDeliveryRef[candidate.DeliveryRef] = candidate.TaskRef
	}
	taskRefs := make([]string, 0, len(unit.DependsOnDeliveries))
	seen := map[string]bool{}
	for _, deliveryRef := range unit.DependsOnDeliveries {
		deliveryRef = strings.TrimSpace(deliveryRef)
		taskRef := byDeliveryRef[deliveryRef]
		if taskRef == "" {
			return nil, AppPlannerIssueV0{Field: "depends_on_deliveries"}
		}
		if seen[taskRef] {
			continue
		}
		seen[taskRef] = true
		taskRefs = append(taskRefs, taskRef)
	}
	return taskRefs, nil
}
