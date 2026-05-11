package orquestaruntime

import orquestacontext "orquesta/modulos/orquesta-context"

func cloneRuntimeFunctionContractV0(in *RuntimeFunctionContractV0) *RuntimeFunctionContractV0 {
	if in == nil {
		return nil
	}
	out := *in
	out.WriteSet = append([]string(nil), in.WriteSet...)
	out.TestsObligatorios = append([]string(nil), in.TestsObligatorios...)
	out.CriterioCierre = append([]string(nil), in.CriterioCierre...)
	return &out
}

func cloneRuntimeCapacityDecisionV0(in *RuntimeCapacityDecisionV0) *RuntimeCapacityDecisionV0 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneRuntimeBindingV0(in *RuntimeBindingV0) *RuntimeBindingV0 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneRuntimeEvidenceRefsV0(in *RuntimeEvidenceRefsV0) *RuntimeEvidenceRefsV0 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneRuntimeContextBundleV0(in *orquestacontext.ContextBundleV0) *orquestacontext.ContextBundleV0 {
	if in == nil {
		return nil
	}
	out := *in
	out.Entries = append([]orquestacontext.ContextBundleEntryV0(nil), in.Entries...)
	out.Issues = append([]orquestacontext.ContextBundleIssueV0(nil), in.Issues...)
	return &out
}

func runtimeLaunchIssuesAsAgentLauncherIssuesV0(
	correlationID string,
	issues []RuntimeLaunchErrorV0,
) []AgentLauncherInboundErrorV0 {
	out := make([]AgentLauncherInboundErrorV0, 0, len(issues))
	for _, issue := range issues {
		field := issue.Field
		if field != "" {
			field = "runtime_launch_request." + field
		}
		out = append(out, AgentLauncherInboundErrorV0{
			Code:          AgentLauncherRuntimeLaunchInvalidaV0,
			MessageKey:    "orquesta.runtime.agent_launcher." + string(AgentLauncherRuntimeLaunchInvalidaV0),
			Field:         field,
			Retryable:     issue.Retryable,
			CorrelationID: firstNonEmptyV0(issue.CorrelationID, correlationID),
			Evidence:      []string{string(issue.Code)},
		})
	}
	return out
}

func correlateAgentLauncherIssuesV0(
	correlationID string,
	issues []AgentLauncherInboundErrorV0,
) []AgentLauncherInboundErrorV0 {
	for i := range issues {
		if issues[i].CorrelationID == "" {
			issues[i].CorrelationID = correlationID
		}
	}
	return issues
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
