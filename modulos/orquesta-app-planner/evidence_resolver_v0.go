package orquestaappplanner

import orquestaruntime "orquesta/modulos/orquesta-runtime"

type AppPlanLaunchEvidenceResolverV0 struct {
	Plan AppMicrotaskPlanV0
}

var _ orquestaruntime.LaunchEvidenceResolverV0 = AppPlanLaunchEvidenceResolverV0{}

func (resolver AppPlanLaunchEvidenceResolverV0) ResolveLaunchEvidenceV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (*orquestaruntime.RuntimeEvidenceRefsV0, error) {
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
	return &orquestaruntime.RuntimeEvidenceRefsV0{
		MailboxRef:    "mailbox-" + unit.AgentRequestID,
		AckRef:        unit.DeliveryRef,
		ReadinessRef:  "readiness-" + unit.AgentRequestID,
		CheckpointRef: "checkpoint-" + unit.AgentRequestID,
	}, nil
}
