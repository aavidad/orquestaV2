package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func TestMCPRequestAppChangeV0DelegaEnCasoDeUso(t *testing.T) {
	store := &fakeMCPAppChangeStoreV0{}
	notifier := &fakeMCPAppChangeNotifierV0{
		result: orquestaappchange.AppChangeDirectorNotificationV0{
			DirectorQuestionRef: "question-ref-change-mcp-001",
			EvidenceRefs:        []string{"change-ref-mcp-001"},
		},
	}
	executor := NewMCPRequestAppChangeToolExecutorV0(orquestaappchange.AppChangePortsV0{
		Store:            store,
		DirectorNotifier: notifier,
	})

	result, err := executor.Execute(context.Background(), MCPRequestAppChangeToolInputV0{
		RequestID:     "req-change-mcp-001",
		CorrelationID: "corr-change-mcp-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			RunRef:     "run-ref-change-mcp-001",
			AppRef:     "app-ref-agenda",
			ChangeRef:  "change-ref-mcp-001",
			UserIntent: "Modificar la web para anadir vista semanal.",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRequestAppChangeEstadoOKV0 ||
		result.DirectorQuestionRef != "question-ref-change-mcp-001" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "change-ref-mcp-001") ||
		store.saved.Request.RequestID != "req-change-mcp-001" ||
		notifier.notified.Request.CorrelationID != "corr-change-mcp-001" {
		t.Fatalf("result=%+v store=%+v notifier=%+v", result, store.saved, notifier.notified)
	}
}

func TestMCPRequestAppChangeDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPRequestAppChangeDescriptorV0()
	if descriptor.Name != MCPRequestAppChangeToolNameV0 ||
		descriptor.ResourceURI != MCPRequestAppChangeResourceURIV0 ||
		!strings.Contains(descriptor.Output, "evidence_refs?") ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
}

type fakeMCPAppChangeStoreV0 struct {
	saved orquestaappchange.AppChangeRecordV0
}

func (store *fakeMCPAppChangeStoreV0) SaveAppChangeRequestV0(
	_ context.Context,
	record orquestaappchange.AppChangeRecordV0,
) error {
	store.saved = record
	return nil
}

type fakeMCPAppChangeNotifierV0 struct {
	notified orquestaappchange.AppChangeRecordV0
	result   orquestaappchange.AppChangeDirectorNotificationV0
}

func (notifier *fakeMCPAppChangeNotifierV0) NotifyAppChangeRequestedV0(
	_ context.Context,
	record orquestaappchange.AppChangeRecordV0,
) (orquestaappchange.AppChangeDirectorNotificationV0, error) {
	notifier.notified = record
	return notifier.result, nil
}
