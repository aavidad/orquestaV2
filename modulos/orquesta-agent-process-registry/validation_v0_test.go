package orquestaagentprocessregistry

import (
	"errors"
	"testing"
)

func TestAgentProcessRegistryRecordV0NormalizaYValidaRefsOpacas(t *testing.T) {
	record := NormalizeAgentProcessRegistryRecordV0(AgentProcessRegistryRecordV0{
		RunID:          " run-agent-process-001 ",
		AgentRequestID: " agent-process-001 ",
		ProcessRef:     " process-ref-001 ",
		SessionRef:     " session-ref-001 ",
		LaunchRef:      " launch-ref-001 ",
		ReadinessRef:   " readiness-ref-001 ",
		EvidenceRefs:   []string{" evidence-ref-001 ", "evidence-ref-001"},
	})

	if err := ValidateAgentProcessRegistryRecordV0(record); err != nil {
		t.Fatalf("validate record: %v", err)
	}
	if record.RunID != "run-agent-process-001" ||
		len(record.EvidenceRefs) != 1 ||
		record.EvidenceRefs[0] != "evidence-ref-001" {
		t.Fatalf("record normalizado=%+v", record)
	}
}

func TestAgentProcessRegistryRecordV0RechazaDetallesOperativos(t *testing.T) {
	record := validRegistryRecordForTestV0()
	record.ProcessRef = "process://raw"

	err := ValidateAgentProcessRegistryRecordV0(record)

	var publicErr ErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("error=%T %v", err, err)
	}
	if publicErr.Field != "agent_process.process_ref" {
		t.Fatalf("field=%s", publicErr.Field)
	}
}

func validRegistryRecordForTestV0() AgentProcessRegistryRecordV0 {
	return AgentProcessRegistryRecordV0{
		RunID:          "run-agent-process-001",
		AgentRequestID: "agent-process-001",
		ProcessRef:     "process-ref-001",
		SessionRef:     "session-ref-001",
		LaunchRef:      "launch-ref-001",
		ReadinessRef:   "readiness-ref-001",
	}
}
