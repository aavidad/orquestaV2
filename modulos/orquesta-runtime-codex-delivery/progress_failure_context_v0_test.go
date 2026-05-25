package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressReportWithProcessFailureContextV0ClasificaCuotaSinFiltrarProveedor(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("ERROR: You've hit your usage limit. Try again later."),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
		report,
	)

	if !got.DecisionRequired ||
		got.BudgetStatus != orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		got.BudgetReason != "Aviso de capacidad externa antes de ACK." ||
		got.Summary != "Proceso detenido por aviso de capacidad externa; requiere relevo." {
		t.Fatalf("report=%+v", got)
	}
	if !stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("evidence refs sin capacity_warning compacto: %+v", got.EvidenceRefs)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0ClasificaAuthInvalidaComoBloqueoRecuperable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("ERROR 401 token_invalidated refresh_token_reused"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
		report,
	)

	if !got.DecisionRequired ||
		got.Summary != "Autenticacion externa invalida; requiere reautorizacion del operador." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-auth-config-blocker") {
		t.Fatalf("report=%+v", got)
	}
	if got.BudgetStatus != "" || got.BudgetReason != "" {
		t.Fatalf("auth invalida no debe confundirse con capacidad: %+v", got)
	}
	if stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-capacity-warning") ||
		stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("auth invalida no debe mezclar evidencias de capacidad/no_ack: %+v", got.EvidenceRefs)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0ClasificaNoACKSinLogs(t *testing.T) {
	report := validCodexProgressFailureReportForTestV0()

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath: filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
		report,
	)

	if !got.DecisionRequired ||
		got.Summary != "Proceso detenido sin ACK." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("report=%+v", got)
	}
	if got.BudgetStatus != "" || got.BudgetReason != "" {
		t.Fatalf("no_ack no debe marcar presupuesto: %+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0NoACKConArtefactoRequiereValidacion(t *testing.T) {
	projectDir := t.TempDir()
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"external/opes/topic"}
	if err := os.MkdirAll(filepath.Join(projectDir, "external/opes"), 0o700); err != nil {
		t.Fatalf("mkdir write-set: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, "external/opes/topic"),
		[]byte(`{"artifact_type":"topic_expansion_package"}`),
		0o600,
	); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath:        filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0),
			ProjectWorkDir: projectDir,
			Spec:           spec,
		},
		report,
	)

	if !got.DecisionRequired ||
		got.Summary != "Proceso detenido sin ACK pero con artefacto en write-set; requiere validacion." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-artifact-without-ack") {
		t.Fatalf("report=%+v", got)
	}
	if got.BudgetStatus != "" || got.BudgetReason != "" {
		t.Fatalf("artefacto sin ACK no debe marcar presupuesto: %+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0StalledConArtefactoRequiereValidacion(t *testing.T) {
	projectDir := t.TempDir()
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"external/opes/topic"}
	if err := os.MkdirAll(filepath.Join(projectDir, "external/opes"), 0o700); err != nil {
		t.Fatalf("mkdir write-set: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, "external/opes/topic"),
		[]byte(`{"artifact_type":"topic_expansion_package"}`),
		0o600,
	); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()
	report.Status = orquestaruntime.AgentStalledV0

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath:        filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0),
			ProjectWorkDir: projectDir,
			Spec:           spec,
		},
		report,
	)

	if !got.DecisionRequired ||
		got.Summary != "Proceso sin ACK pero con artefacto en write-set; requiere validacion antes de replanificar." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-artifact-without-ack") {
		t.Fatalf("report=%+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressFailureClassFromDescriptorV0ClasificaSenalesCompactas(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    codexProgressFailureClassV0
	}{
		{
			name:    "interrupted",
			content: "turn interrupted\n42 tokens used\n",
			want:    codexProgressFailureInterruptedV0,
		},
		{
			name:    "capacity_warning_priority",
			content: "turn interrupted\nlow remaining capacity for this model\n",
			want:    codexProgressFailureCapacityWarningV0,
		},
		{
			name:    "auth_invalid_priority",
			content: "ERROR 401 token_invalidated selected service is at capacity",
			want:    codexProgressFailureAuthInvalidV0,
		},
		{
			name:    "no_ack_without_signal",
			content: "",
			want:    codexProgressFailureNoACKV0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != "" {
				if err := os.WriteFile(
					filepath.Join(dir, orquestaruntimecodex.CodexLastMessageFileNameV0),
					[]byte(tt.content),
					0o600,
				); err != nil {
					t.Fatalf("write last message: %v", err)
				}
			}
			got := codexProgressFailureClassFromDescriptorV0(CodexReceiptDescriptorV0{
				AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
			})
			if got != tt.want {
				t.Fatalf("failure class=%s want=%s", got, tt.want)
			}
		})
	}
}

func validCodexProgressFailureReportForTestV0() orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:        "agent-progress-report-ref-quota-001",
		RunID:           "run-ref-quota-001",
		AgentRequestID:  "agent-ref-quota-001",
		Status:          orquestaruntime.AgentStoppedV0,
		Summary:         "Proceso observado como parado.",
		EvidenceRefs:    []string{"evidence-ref-quota-001"},
		NoProgressTicks: 1,
	}
}
