package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexReceiptDeliveryLoopV0SmokeCodexRealAppOptIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_APP_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_APP_SMOKE=1 para crear una app con Codex real")
	}
	cfg := codexRealSmokeConfigForTestV0(t)
	cfg.Timeout = 240 * time.Second
	spec := codexRealAppSmokeSpecForTestV0("agent-ref-real-app-smoke-001", "task-ref-real-app-smoke-001")
	runRef := "run-ref-real-app-smoke-001"
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	run := codexDeliveryLoopRunForTestV0(t, runRef, spec.AgentPacket.Task.TaskRef)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	processRegistry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	service := codexRealSmokeServiceForTestV0(store, sink, ledger, receiptStore, runRef, spec)
	dispatchers := codexRealSmokeDispatchersForTestV0(
		store,
		sink,
		ledger,
		receiptStore,
		processRegistry,
		processRuntime,
		cfg,
		spec,
	)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	result, err := codexRealSmokeRunLoopForTestV0(ctx, service, runRef, dispatchers)
	if err != nil {
		t.Fatalf("launch loop app: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.StartedAgents, spec.RequestID) {
		t.Fatalf("started_agents=%v", result.Run.StartedAgents)
	}
	defer codexRealSmokeStopProcessForTestV0(t, processRuntime, processRegistry, runRef, spec.RequestID)

	if err := codexRealSmokeWaitForAckV0(ctx, cfg.RuntimeWorkDir, spec); err != nil {
		t.Fatalf("ack app no validado: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	result, err = codexRealSmokeRunLoopForTestV0(ctx, service, runRef, dispatchers)
	if err != nil {
		t.Fatalf("delivery loop app: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.Deliveries, spec.AgentPacket.DeliveryRefs.AckRef) {
		t.Fatalf("deliveries=%v", result.Run.Deliveries)
	}
	codexRealAppSmokeVerifyProjectV0(t, cfg.ProjectWorkDir, codexRealAppSmokeExpectedFilesV0())
}

func codexRealAppSmokeSpecForTestV0(agentRef string, taskRef string) orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	spec.AgentPacket.TargetModule = "agenda-go-api-web-smoke"
	spec.AgentPacket.Task.Title = "Crear mini agenda Go con API REST y web"
	spec.AgentPacket.Task.Objective = strings.Join([]string{
		"Crea una app Go minima de agenda con API REST usando net/http.",
		"Debe compilar con go test ./... y servir una web estatica simple.",
		"Para este smoke no uses base de datos, dependencias externas ni paquetes innecesarios.",
		"Si necesitas simplificar, concentra el servidor en server/main.go.",
	}, " ")
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"server",
		"internal",
		"web",
		"README.md",
	}
	spec.AgentPacket.Task.RequiredTests = []string{"go test ./..."}
	spec.AgentPacket.Task.DoneCriteria = []string{
		"API REST minima de agenda.",
		"Web estatica simple.",
		"`go test ./...` ejecutado y declarado en el ACK.",
		"agent_ack.json escrito con status completed.",
	}
	spec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-real-app-smoke-001"
	spec.AgentPacket.DeliveryRefs.MailboxRef = "mailbox-ref-real-app-smoke-001"
	spec.AgentPacket.DeliveryRefs.ReadinessRef = "readiness-ref-real-app-smoke-001"
	return spec
}

func codexRealAppSmokeExpectedFilesV0() []string {
	return []string{
		"go.mod",
		"server/main.go",
		"web/index.html",
		"README.md",
	}
}

func codexRealAppSmokeVerifyProjectV0(t *testing.T, projectDir string, expectedFiles []string) {
	t.Helper()
	for _, path := range expectedFiles {
		if _, err := os.Stat(filepath.Join(projectDir, path)); err != nil {
			t.Fatalf("artifact missing %s: %v", path, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... app: %v\n%s", err, codexRealSmokeTruncateV0(string(output)))
	}
}
