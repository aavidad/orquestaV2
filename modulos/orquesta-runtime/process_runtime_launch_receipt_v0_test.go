package orquestaruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestProcessRuntimeConnectorV0ExponeReciboLaunchIOYEnvRedactado(t *testing.T) {
	connector := NewProcessRuntimeConnectorV0()
	req := processRuntimeLaunchRequestForTestV0(t, "exit")

	launched, err := connector.LaunchV0(context.Background(), req)
	requireNoProcessRuntimeErrorV0(t, err)
	waitForProcessRuntimeStatusV0(t, connector, launched.ProcessRef, ProcessRuntimeStoppedV0)

	assertProcessRuntimeLaunchReceiptV0(t, launched, req)
	if len(launched.LaunchReceipt.EnvRefs) != 1 ||
		!strings.HasPrefix(launched.LaunchReceipt.EnvRefs[0], "env-ref-process-runtime-redacted-count-") {
		t.Fatalf("env refs no redactadas: %+v", launched.LaunchReceipt)
	}
}

func assertProcessRuntimeLaunchReceiptV0(
	t *testing.T,
	snapshot ProcessRuntimeSnapshotV0,
	req ProcessRuntimeLaunchRequestV0,
) {
	t.Helper()
	if snapshot.LaunchReceipt == nil {
		t.Fatalf("launch_receipt requerido: %+v", snapshot)
	}
	if snapshot.LaunchReceipt.IO.StdoutPolicy != ProcessRuntimeLaunchIOPolicyDiscardV0 ||
		snapshot.LaunchReceipt.IO.StderrPolicy != ProcessRuntimeLaunchIOPolicyDiscardV0 ||
		!snapshot.LaunchReceipt.IO.OutputRedacted {
		t.Fatalf("io policy inesperada: %+v", snapshot.LaunchReceipt.IO)
	}
	if len(snapshot.LaunchReceiptRefs) == 0 {
		t.Fatalf("launch_receipt_refs vacias: %+v", snapshot)
	}
	raw, err := json.Marshal(snapshot.LaunchReceipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, req.CommandPath) ||
		strings.Contains(text, req.WorkingDir) ||
		strings.Contains(text, processRuntimeTestChildEnvV0) {
		t.Fatalf("receipt filtra detalle operacional: %s", text)
	}
}
