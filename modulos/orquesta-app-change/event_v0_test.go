package orquestaappchange

import (
	"context"
	"testing"
)

func TestReceiveAppChangeIntentEventV0RegistraSolicitudDeCambio(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()
	notifier := &fakeAppChangeNotifierV0{}

	result, err := ReceiveAppChangeIntentEventV0(
		context.Background(),
		validAppChangeIntentEventForTestV0(),
		AppChangePortsV0{Store: store, DirectorNotifier: notifier},
	)
	if err != nil {
		t.Fatalf("ReceiveAppChangeIntentEventV0: %v", err)
	}
	if result.Status != AppChangeStatusAcceptedV0 ||
		result.ChangeRef != "change-ref-mid-001" ||
		result.DirectorQuestionRef != "question-ref-app-change-change-ref-mid-001" {
		t.Fatalf("result=%+v", result)
	}
	records, err := store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-ref-agenda-001"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 1 ||
		records[0].Request.UserIntent != "Quiero modificar la vista semanal." ||
		records[0].ReceivedAt != "2026-05-10T10:20:00Z" {
		t.Fatalf("records=%+v", records)
	}
}

func TestReceiveAppChangeIntentEventV0DerivaChangeRefDesdeEvento(t *testing.T) {
	event := validAppChangeIntentEventForTestV0()
	event.ChangeRef = ""
	event.EventID = "event-mid-002"

	result, err := ReceiveAppChangeIntentEventV0(
		context.Background(),
		event,
		AppChangePortsV0{Store: NewInMemoryAppChangeStoreV0(), DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("ReceiveAppChangeIntentEventV0: %v", err)
	}
	if result.Status != AppChangeStatusAcceptedV0 ||
		result.ChangeRef != "change-ref-event-mid-002" {
		t.Fatalf("result=%+v", result)
	}
}

func TestReceiveAppChangeIntentEventV0InvalidoNoTocaPuertos(t *testing.T) {
	store := &fakeAppChangeStoreV0{}
	notifier := &fakeAppChangeNotifierV0{}
	event := validAppChangeIntentEventForTestV0()
	event.EventID = ""
	event.ChangeRef = ""
	event.UserIntent = ""

	result, err := ReceiveAppChangeIntentEventV0(
		context.Background(),
		event,
		AppChangePortsV0{Store: store, DirectorNotifier: notifier},
	)
	if err != nil {
		t.Fatalf("ReceiveAppChangeIntentEventV0: %v", err)
	}
	if result.Status != AppChangeStatusInvalidV0 || len(result.Issues) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if store.calls != 0 || notifier.calls != 0 {
		t.Fatalf("puertos tocados store=%d notifier=%d", store.calls, notifier.calls)
	}
}

func validAppChangeIntentEventForTestV0() AppChangeIntentEventV0 {
	return AppChangeIntentEventV0{
		SchemaVersion:      AppChangeIntentEventSchemaV0,
		EventID:            "event-mid-001",
		Source:             "orquesta-web",
		OccurredAt:         "2026-05-10T10:20:00Z",
		RunRef:             "run-ref-agenda-001",
		AppRef:             "app-ref-agenda",
		ChangeRef:          "change-ref-mid-001",
		Locale:             "es-ES",
		UserIntent:         "Quiero modificar la vista semanal.",
		TargetArea:         "web",
		CurrentStateRefs:   []string{"delivery-ref-web-001"},
		AcceptanceCriteria: []string{"vista semanal modificada"},
		AllowedWriteSet:    []string{"web/agenda"},
	}
}
