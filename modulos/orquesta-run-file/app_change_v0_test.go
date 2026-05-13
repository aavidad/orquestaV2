package orquestarunfile

import (
	"context"
	"errors"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRunFileStoreAppChangePersistsAfterRecreateAndReplacesV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	first := appChangeRecordForRunFileTestV0(" run-a ", " change-1 ")

	if err := store.SaveAppChangeRequestV0(ctx, first); err != nil {
		t.Fatalf("SaveAppChangeRequestV0 first: %v", err)
	}
	first.Request.AcceptanceCriteria[0] = "mutated-input"
	if err := store.SaveAppChangeRequestV0(
		ctx,
		appChangeRecordForRunFileTestV0("run-b", "change-2"),
	); err != nil {
		t.Fatalf("SaveAppChangeRequestV0 second: %v", err)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	records, err := reopened.ListAppChangeRecordsV0(ctx, orquestaappchange.AppChangeRecordFilterV0{
		RunRef: " run-a ",
	})
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 1 ||
		records[0].Request.RunRef != "run-a" ||
		records[0].Request.ChangeRef != "change-1" ||
		records[0].Request.RequestID != "change-1" ||
		records[0].Request.AcceptanceCriteria[0] != "criterio-original" ||
		records[0].Request.ExternalWork == nil ||
		records[0].Request.ExternalWork.JobRef != "job-ref-opes-001" ||
		records[0].Request.ExternalWork.InputFields[0].Name != "topic_id" ||
		records[0].Request.ExternalWork.InputFields[2].ValueJSON == nil {
		t.Fatalf("records=%+v", records)
	}

	records[0].Request.AcceptanceCriteria[0] = "mutated-output"
	records[0].Request.ExternalWork.InputFields[0].Value = "mutated-topic"
	records, err = reopened.ListAppChangeRecordsV0(ctx, orquestaappchange.AppChangeRecordFilterV0{
		RunRef: "run-a",
	})
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0 again: %v", err)
	}
	if records[0].Request.AcceptanceCriteria[0] != "criterio-original" ||
		records[0].Request.ExternalWork.InputFields[0].Value != "topic-ref-opes-001" {
		t.Fatalf("record leaked mutable slices: %+v", records[0])
	}

	updated := appChangeRecordForRunFileTestV0("run-a", "change-1")
	updated.Request.UserIntent = "Actualizar la vista semanal con filtros."
	if err := reopened.SaveAppChangeRequestV0(ctx, updated); err != nil {
		t.Fatalf("SaveAppChangeRequestV0 updated: %v", err)
	}
	reopened = mustNewRunFileStoreV0(t, dir)
	records, err = reopened.ListAppChangeRecordsV0(ctx, orquestaappchange.AppChangeRecordFilterV0{
		RunRef: "run-a",
	})
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0 after replace: %v", err)
	}
	if len(records) != 1 || records[0].Request.UserIntent != updated.Request.UserIntent {
		t.Fatalf("records after replace=%+v", records)
	}
}

func TestRunFileStoreAppChangeRespectsCanceledContextV0(t *testing.T) {
	store := mustNewRunFileStoreV0(t, t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.SaveAppChangeRequestV0(ctx, appChangeRecordForRunFileTestV0("run-a", "change-1"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("save err=%v", err)
	}
	_, err = store.ListAppChangeRecordsV0(ctx, orquestaappchange.AppChangeRecordFilterV0{RunRef: "run-a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("list err=%v", err)
	}
}

func appChangeRecordForRunFileTestV0(
	runRef string,
	changeRef string,
) orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RunRef:             runRef,
			AppRef:             " app-ref ",
			ChangeRef:          changeRef,
			UserIntent:         " Cambiar agenda ",
			TargetArea:         " web ",
			CurrentStateRefs:   []string{" state-1 ", "state-1"},
			AcceptanceCriteria: []string{" criterio-original ", "criterio-original"},
			AllowedWriteSet:    []string{" modulos/app/main.go "},
			MetadataRefs:       []string{" meta-1 "},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef:    " opes ",
				JobRef:        " job-ref-opes-001 ",
				InterfaceRefs: []string{" opes-rest-v0 ", "opes-rest-v0"},
				WorkKind:      " draft_content_block ",
				WorkRefs:      []string{" topic-ref-opes-001 ", "topic-ref-opes-001"},
				InputFields: []orquestadomainwork.DomainWorkFieldV0{
					{Name: " topic_id ", Value: " topic-ref-opes-001 "},
					{Name: " source_refs ", Values: []string{" BOE-A-001 ", "BOE-A-001"}},
					{Name: " block_position ", ValueJSON: []byte(` { "block_order": 1 } `)},
				},
			},
		},
		ReceivedAt:  " 2026-05-12T09:00:00Z ",
		RequestedBy: " user ",
	}
}
