package orquestaappplanner

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const defaultAppPlanContextTargetModuleV0 = "orquesta-app-planner"

type AppPlanContextBundleResolverV0 struct {
	Plan              AppMicrotaskPlanV0
	TargetModule      string
	CapacityDecisions orquestaruntime.CapacityDecisionResolverV0
}

var _ orquestaruntime.ContextBundleResolverV0 = AppPlanContextBundleResolverV0{}

func (resolver AppPlanContextBundleResolverV0) ResolveContextBundleV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (*orquestacontext.ContextBundleV0, error) {
	if issues := orquestaruntime.ValidateAgentLauncherInboundV0(inbound); len(issues) > 0 {
		return nil, AppPlannerIssueV0{Field: issues[0].Field}
	}
	if err := validateAppMicrotaskPlanV0(resolver.Plan); err != nil {
		return nil, err
	}
	unit, ok := findAppPlanUnitByTaskRefV0(resolver.Plan, inbound.Payload.TaskRef)
	if !ok {
		return nil, AppPlannerIssueV0{Field: "task_ref"}
	}
	capacityLevel, err := resolver.contextCapacityLevelV0(unit, inbound.Payload.CapacityRequestRef)
	if err != nil {
		return nil, err
	}
	bundle := orquestacontext.BuildContextBundleV0(orquestacontext.ContextBundleRequestV0{
		SchemaVersion: orquestacontext.ContextBundleRequestSchemaVersionV0,
		BundleRef:     "context-bundle-" + unit.TaskRef,
		WorkOrderRef:  unit.TaskRef,
		TargetModule:  resolver.contextTargetModuleV0(),
		Phase:         string(unit.PhaseID),
		TaskKind:      unit.Role,
		Objective:     unit.Summary,
		CapacityLevel: capacityLevel,
		WriteSet:      unit.WriteSet,
		ContractRefs: []string{
			"contract-" + unit.TaskRef,
			"AppMicrotaskPlanV0",
		},
		CrossModuleRefs: []string{
			"orquesta-runtime:RuntimeLaunchRequestV0",
			"orquesta-core-workflow:RequestAgent",
		},
		EvidenceRefs:  appPlanContextEvidenceRefsV0(resolver.Plan, unit, inbound.Payload.EvidenceRefs),
		MaxEntries:    24,
		MaxTotalBytes: 24000,
	})
	return &bundle, nil
}

func (resolver AppPlanContextBundleResolverV0) contextTargetModuleV0() string {
	if resolver.TargetModule != "" {
		return resolver.TargetModule
	}
	return defaultAppPlanContextTargetModuleV0
}

func (resolver AppPlanContextBundleResolverV0) contextCapacityLevelV0(
	unit AppWorkUnitV0,
	capacityRequestRef string,
) (string, error) {
	if resolver.CapacityDecisions == nil {
		return string(unit.Capacity), nil
	}
	decision, err := resolver.CapacityDecisions.ResolveCapacityDecisionV0(capacityRequestRef)
	if err != nil {
		return "", err
	}
	if decision == nil || decision.NivelCapacidad == "" {
		return string(unit.Capacity), nil
	}
	return decision.NivelCapacidad, nil
}

func appPlanContextEvidenceRefsV0(
	plan AppMicrotaskPlanV0,
	unit AppWorkUnitV0,
	refs []string,
) []string {
	values := append([]string(nil), plan.EvidenceRefs...)
	values = append(values, unit.EvidenceRefs...)
	values = append(values, refs...)
	return compactAppPlannerStringsV0(values)
}
