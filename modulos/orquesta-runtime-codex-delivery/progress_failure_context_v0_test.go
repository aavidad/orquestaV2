package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressReportWithProcessFailureContextV0NoClasificaCapacidadPorTexto(t *testing.T) {
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
		got.Summary != "Proceso detenido sin ACK." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("report=%+v", got)
	}
	if got.BudgetStatus != "" || got.BudgetReason != "" ||
		stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("texto de capacidad no debe clasificar presupuesto: %+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0ClasificaCuotaEstructurada(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{},"quota":{"status":"exhausted"}}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage accounting: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()
	descriptor := CodexReceiptDescriptorV0{
		AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
	}

	got := codexProgressReportWithProcessFailureContextV0(descriptor, report)

	if got.BudgetStatus != orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		got.BudgetReason != "Cuota externa agotada antes de ACK." ||
		!got.DecisionRequired ||
		got.Summary != "Cuota externa agotada antes de ACK; cerrar agente y replanificar automaticamente cuando haya capacidad." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-provider-quota-exhausted") ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-capacity-limited") {
		t.Fatalf("report cuota estructurada inesperado=%+v", got)
	}
	if gotClass := codexProgressFailureClassFromDescriptorV0(descriptor); gotClass != codexProgressFailureProviderQuotaExhaustedV0 {
		t.Fatalf("failure class=%s want=%s", gotClass, codexProgressFailureProviderQuotaExhaustedV0)
	}
	if strings.Contains(got.Summary, dir) || strings.Contains(got.BudgetReason, dir) {
		t.Fatalf("report filtra ruta local: %+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func TestCodexProgressReportWithProcessFailureContextV0NoClasificaAuthPorTextoLibre(t *testing.T) {
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
		got.Summary != "Proceso detenido sin ACK." ||
		!stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("report no_ack inesperado=%+v", got)
	}
	if got.BudgetStatus != "" || got.BudgetReason != "" ||
		stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-auth-config-blocker") ||
		stringInCodexDeliverySetV0(got.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("texto libre no debe clasificar auth/capacity: %+v", got)
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
			name:    "capacity_text_does_not_override_interrupted",
			content: "turn interrupted\nlow remaining capacity for this model\n",
			want:    codexProgressFailureInterruptedV0,
		},
		{
			name:    "auth_text_does_not_block",
			content: "ERROR 401 token_invalidated selected service is at capacity",
			want:    codexProgressFailureNoACKV0,
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

func TestCodexProgressReadFailureLogV0LeeSoloTailAcotado(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0)
	prefix := "capacity\n" + strings.Repeat("padding\n", 10*1024)
	suffix := "turn was interrupted\n"
	if err := os.WriteFile(path, []byte(prefix+suffix), 0o600); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	got, ok := codexProgressReadFailureLogV0(path)
	if !ok {
		t.Fatalf("failure log no leido")
	}
	if len(got) > maxCodexProgressFailureLogBytesV0 {
		t.Fatalf("tail len=%d max=%d", len(got), maxCodexProgressFailureLogBytesV0)
	}
	if strings.Contains(got, "capacity") || !strings.Contains(got, "turn was interrupted") {
		t.Fatalf("tail inesperado")
	}
}

func TestCodexProgressFailureClassFromDescriptorV0ClasificaDesdeTailSinPrefijoGrande(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("capacity\n"+strings.Repeat("padding\n", 10*1024)+"turn interrupted\n"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	got := codexProgressFailureClassFromDescriptorV0(CodexReceiptDescriptorV0{
		AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
	})
	if got != codexProgressFailureInterruptedV0 {
		t.Fatalf("failure class=%s want=%s", got, codexProgressFailureInterruptedV0)
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
