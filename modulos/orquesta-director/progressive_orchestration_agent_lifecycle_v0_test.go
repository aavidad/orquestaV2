package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func (h *progressiveHarnessV0) registerAgentStarted(agentRef string, launchRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewRegisterAgentStartedCommandV0(
		h.meta("started-"+agentRef),
		orquestacoreworkflow.RegisterAgentStartedCommandPayloadV0{
			AgentRequestID: agentRef,
			LaunchRef:      launchRef,
			AckRef:         "ack-ref-" + agentRef,
			ReadinessRef:   "readiness-ref-" + agentRef,
			EvidenceRefs:   []string{"evidence-ref-agent-started-001"},
		},
	)
	if err != nil {
		h.t.Fatalf("NewRegisterAgentStartedCommandV0: %v", err)
	}
	return cmd
}
