package orquestaappchange

import (
	"context"
	"errors"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestInMemoryAppChangeStoreV0GuardaYListaPorRunRef(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()

	mustSaveAppChangeRecordV0(t, store, appChangeRecordForStoreTestV0("run-a", "change-1"))
	mustSaveAppChangeRecordV0(t, store, appChangeRecordForStoreTestV0("run-b", "change-2"))
	mustSaveAppChangeRecordV0(t, store, appChangeRecordForStoreTestV0("run-a", "change-3"))

	records, err := store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-a"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%+v", records)
	}
	if records[0].Request.ChangeRef != "change-1" || records[1].Request.ChangeRef != "change-3" {
		t.Fatalf("orden inesperado: %+v", records)
	}
}

func TestInMemoryAppChangeStoreV0SirveAlCasoDeUso(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()

	result, err := RequestAppChangeV0(
		context.Background(),
		validAppChangeRequestForTestV0(),
		AppChangePortsV0{Store: store, DirectorNotifier: &fakeAppChangeNotifierV0{}},
	)
	if err != nil {
		t.Fatalf("RequestAppChangeV0: %v", err)
	}
	if result.Status != AppChangeStatusAcceptedV0 {
		t.Fatalf("result=%+v", result)
	}

	records, err := store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-ref-agenda-001"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 1 || records[0].Request.ChangeRef != "change-ref-001" {
		t.Fatalf("records=%+v", records)
	}
}

func TestInMemoryAppChangeStoreV0ReemplazaMismaSolicitud(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()
	first := appChangeRecordForStoreTestV0("run-a", "change-1")
	updated := first
	updated.Request.UserIntent = "Actualizar la vista semanal con filtros."

	mustSaveAppChangeRecordV0(t, store, first)
	mustSaveAppChangeRecordV0(t, store, updated)

	records, err := store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-a"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 1 || records[0].Request.UserIntent != updated.Request.UserIntent {
		t.Fatalf("records=%+v", records)
	}
}

func TestInMemoryAppChangeStoreV0UsaCopiasDefensivas(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()
	record := appChangeRecordForStoreTestV0("run-a", "change-1")
	record.Request.AcceptanceCriteria = []string{"criterio-original"}
	record.Request.RequiredTests = []string{"test-original"}
	record.Request.ExternalWork.RequiredTests = []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "domain-test-original",
	}}

	mustSaveAppChangeRecordV0(t, store, record)
	record.Request.AcceptanceCriteria[0] = "criterio-mutado"
	record.Request.RequiredTests[0] = "test-mutado"
	record.Request.ExternalWork.RequiredTests[0].TestRef = "domain-test-mutado"

	records, err := store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-a"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	records[0].Request.AcceptanceCriteria[0] = "criterio-mutado-desde-listado"
	records[0].Request.RequiredTests[0] = "test-mutado-desde-listado"
	records[0].Request.ExternalWork.RequiredTests[0].TestRef = "domain-test-mutado-desde-listado"

	records, err = store.ListAppChangeRecordsV0(
		context.Background(),
		AppChangeRecordFilterV0{RunRef: "run-a"},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if records[0].Request.AcceptanceCriteria[0] != "criterio-original" ||
		records[0].Request.RequiredTests[0] != "test-original" ||
		records[0].Request.ExternalWork.RequiredTests[0].TestRef != "domain-test-original" {
		t.Fatalf("record mutado: %+v", records[0])
	}
}

func TestInMemoryAppChangeStoreV0RespetaContextoCancelado(t *testing.T) {
	store := NewInMemoryAppChangeStoreV0()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.SaveAppChangeRequestV0(ctx, appChangeRecordForStoreTestV0("run-a", "change-1"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("save err=%v", err)
	}
	_, err = store.ListAppChangeRecordsV0(ctx, AppChangeRecordFilterV0{RunRef: "run-a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("list err=%v", err)
	}
}

func mustSaveAppChangeRecordV0(
	t *testing.T,
	store AppChangeStorePortV0,
	record AppChangeRecordV0,
) {
	t.Helper()
	if err := store.SaveAppChangeRequestV0(context.Background(), record); err != nil {
		t.Fatalf("SaveAppChangeRequestV0: %v", err)
	}
}

func appChangeRecordForStoreTestV0(runRef string, changeRef string) AppChangeRecordV0 {
	request := validAppChangeRequestForTestV0()
	request.RunRef = runRef
	request.ChangeRef = changeRef
	return AppChangeRecordV0{
		Request:     request,
		RequestedBy: AppChangeDefaultRequestedByV0,
	}
}

var _ AppChangeRecordStorePortV0 = (*InMemoryAppChangeStoreV0)(nil)
