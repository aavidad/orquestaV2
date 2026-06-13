package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestExternalBridgeInputLedgerClaimBlocksSecondSubmitV0(t *testing.T) {
	ctx := context.Background()
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	entry := externalBridgeInputLedgerEntryV0{
		Key:            externalBridgeInputLedgerKeyV0("opes", "job-ref-001"),
		ExternalSystem: "opes",
		ExternalJobRef: "job-ref-001",
		ClaimRef:       "claim-ref-001",
		CorrelationID:  "corr-ref-001",
		IdempotencyKey: "idem-ref-001",
		RunRef:         "run-ref-001",
		ChangeRef:      "change-ref-001",
	}

	claimed, acquired, err := ledger.ClaimExternalBridgeInputV0(ctx, entry)
	if err != nil || !acquired || claimed.Status != externalBridgeInputStatusClaimedV0 {
		t.Fatalf("claimed=%+v acquired=%v err=%v", claimed, acquired, err)
	}
	previous, secondAcquired, err := ledger.ClaimExternalBridgeInputV0(ctx, entry)
	if err != nil || secondAcquired || previous.ClaimRef != "claim-ref-001" {
		t.Fatalf("previous=%+v acquired=%v err=%v", previous, secondAcquired, err)
	}
}

func TestExternalBridgeInputLedgerRejectsSubmittedOverwriteV0(t *testing.T) {
	ctx := context.Background()
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := externalBridgeRecordSubmittedInputV0(
		ctx, ledger, "opes", "job-ref-001", "run-ref-001", "change-ref-001",
	); err != nil {
		t.Fatalf("record first: %v", err)
	}
	err = externalBridgeRecordSubmittedInputV0(
		ctx, ledger, "opes", "job-ref-001", "run-ref-002", "change-ref-001",
	)
	if err == nil || err.Error() != "external_bridge_submitted_conflict" {
		t.Fatalf("err=%v", err)
	}
}

func TestRunOPESDrainOnceV0ClaimExisteAntesDeSubmitV0(t *testing.T) {
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	job := opesClaimTestJobV0("job-ref-claim-001")
	opesServer := opesClaimTestOPESServerV0(t, job)
	defer opesServer.Close()

	submitChecks := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		submitChecks++
		entry, found, err := opesBridgeSubmittedLedgerEntryV0(context.Background(), ledger, job)
		if err != nil {
			t.Fatalf("lookup submitted: %v", err)
		}
		if found || entry.Status != "" {
			t.Fatalf("submitted antes de POST: entry=%+v found=%v", entry, found)
		}
		claimed, ok, err := ledger.LookupExternalBridgeInputV0(
			context.Background(), opesBridgeInputLedgerKeyV0(job.ID),
		)
		if err != nil || !ok || claimed.Status != externalBridgeInputStatusClaimedV0 {
			t.Fatalf("claimed=%+v ok=%v err=%v", claimed, ok, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"run_ref": "run-ref-claim-001", "estado": "accepted"})
	}))
	defer orquestaServer.Close()

	summary, err := runOPESDrainOnceV0(context.Background(), opesClaimDrainConfigV0(
		opesServer.URL, orquestaServer.URL, ledger,
	))
	if err != nil || submitChecks != 1 || summary.Submitted != 1 || summary.Results[0].Status != "submitted" {
		t.Fatalf("submitChecks=%d summary=%+v err=%v", submitChecks, summary, err)
	}
}

func TestRunOPESDrainOnceV0ClaimPrevioBloqueaReenvioV0(t *testing.T) {
	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	job := opesClaimTestJobV0("job-ref-claimed-001")
	if _, acquired, err := ledger.ClaimExternalBridgeInputV0(context.Background(), externalBridgeInputLedgerEntryV0{
		Key:            opesBridgeInputLedgerKeyV0(job.ID),
		ExternalSystem: opesBridgeExternalSystemV0,
		ExternalJobRef: job.ID,
		ClaimRef:       opesBridgeInputClaimRefV0(job.ID),
		RunRef:         "run-ref-claimed-001",
		ChangeRef:      "opes-job-job-ref-claimed-001",
	}); err != nil || !acquired {
		t.Fatalf("claim err=%v acquired=%v", err, acquired)
	}
	opesServer := opesClaimTestOPESServerV0(t, job)
	defer opesServer.Close()
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("orquesta no debe recibir POST")
	}))
	defer orquestaServer.Close()

	summary, err := runOPESDrainOnceV0(context.Background(), opesClaimDrainConfigV0(
		opesServer.URL, orquestaServer.URL, ledger,
	))
	if err != nil || summary.Claimed != 1 || summary.Results[0].Status != "claimed" ||
		summary.Errors[0].Code != externalBridgeRecoveryRequiredCodeV0 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}

func TestRunOPESDrainOnceV0ClaimRaceSubmittedSupervisaSinReenviarV0(t *testing.T) {
	job := opesClaimTestJobV0("job-ref-race-submitted-001")
	ledger := &raceSubmittedExternalBridgeInputLedgerV0{
		entry: externalBridgeInputLedgerEntryV0{
			Key:            opesBridgeInputLedgerKeyV0(job.ID),
			ExternalSystem: opesBridgeExternalSystemV0,
			ExternalJobRef: job.ID,
			Status:         externalBridgeInputStatusSubmittedV0,
			RunRef:         "run-ref-race-submitted-001",
			ChangeRef:      "opes-job-job-ref-race-submitted-001",
		},
	}
	opesServer := opesClaimTestOPESServerV0(t, job)
	defer opesServer.Close()
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			supervisions++
			return
		}
		t.Fatalf("orquesta no debe recibir submit, recibido %s %s", r.Method, r.URL.Path)
	}))
	defer orquestaServer.Close()

	summary, err := runOPESDrainOnceV0(context.Background(), opesClaimDrainConfigV0(
		opesServer.URL, orquestaServer.URL, ledger,
	))
	if err != nil ||
		supervisions != 1 ||
		summary.AlreadySubmitted != 1 ||
		summary.Submitted != 0 ||
		summary.Results[0].Status != "already_submitted" ||
		summary.Results[0].SupervisionStatus != "running" {
		t.Fatalf("supervisions=%d summary=%+v err=%v", supervisions, summary, err)
	}
}

func TestRunOPESDrainOnceV0FalloFinalLedgerExigeRecoveryV0(t *testing.T) {
	ledger := &failingSubmitExternalBridgeInputLedgerV0{entries: map[string]externalBridgeInputLedgerEntryV0{}}
	job := opesClaimTestJobV0("job-ref-recovery-001")
	opesServer := opesClaimTestOPESServerV0(t, job)
	defer opesServer.Close()
	externalPosts := 0
	supervisions := 0
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeOrquestaRunSuperviseOKForDrainTestV0(t, w, r) {
			supervisions++
			return
		}
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		externalPosts++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"run_ref": "run-ref-recovery-001", "estado": "accepted"})
	}))
	defer orquestaServer.Close()

	summary, err := runOPESDrainOnceV0(context.Background(), opesClaimDrainConfigV0(
		opesServer.URL, orquestaServer.URL, ledger,
	))
	if err != nil || externalPosts != 1 || supervisions != 1 || summary.Submitted != 1 ||
		summary.Results[0].Status != "recovery_required" ||
		summary.Results[0].SupervisionStatus != "running" ||
		summary.Errors[0].Code != externalBridgeRecoveryRequiredCodeV0 {
		t.Fatalf("externalPosts=%d supervisions=%d summary=%+v err=%v", externalPosts, supervisions, summary, err)
	}
}

func opesClaimTestJobV0(jobRef string) orquestaopesconnector.ExternalJobV0 {
	return orquestaopesconnector.ExternalJobV0{
		ID:             jobRef,
		Type:           "draft_content_block",
		Status:         "pending",
		ExecutionMode:  "external",
		PayloadJSON:    `{"topic_id":"topic-ref-001","title":"Bloque"}`,
		CorrelationID:  "corr-" + jobRef,
		IdempotencyKey: "idem-" + jobRef,
		RequestedBy:    "opes",
	}
}

func opesClaimTestOPESServerV0(
	t *testing.T,
	job orquestaopesconnector.ExternalJobV0,
) *httptest.Server {
	t.Helper()
	return newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]orquestaopesconnector.ExternalJobV0{job})
	}))
}

func opesClaimDrainConfigV0(
	opesBaseURL string,
	orquestaBaseURL string,
	ledger externalBridgeInputLedgerV0,
) opesDrainConfigV0 {
	return opesDrainConfigV0{
		OPESBaseURL:        opesBaseURL,
		OrquestaBaseURL:    orquestaBaseURL,
		Limit:              1,
		JobType:            "draft_content_block",
		HTTPTimeout:        time.Second,
		SuperviseSubmitted: true,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}
}

type failingSubmitExternalBridgeInputLedgerV0 struct {
	entries map[string]externalBridgeInputLedgerEntryV0
}

type raceSubmittedExternalBridgeInputLedgerV0 struct {
	entry externalBridgeInputLedgerEntryV0
}

func (ledger *raceSubmittedExternalBridgeInputLedgerV0) LookupExternalBridgeInputV0(
	context.Context,
	string,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	return externalBridgeInputLedgerEntryV0{}, false, nil
}

func (ledger *raceSubmittedExternalBridgeInputLedgerV0) ClaimExternalBridgeInputV0(
	context.Context,
	externalBridgeInputLedgerEntryV0,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	return ledger.entry, false, nil
}

func (ledger *raceSubmittedExternalBridgeInputLedgerV0) RecordExternalBridgeInputSubmittedV0(
	context.Context,
	externalBridgeInputLedgerEntryV0,
) error {
	return fmt.Errorf("unexpected_submit_record")
}

func (ledger *raceSubmittedExternalBridgeInputLedgerV0) UpsertExternalBridgeInputV0(
	ctx context.Context,
	entry externalBridgeInputLedgerEntryV0,
) error {
	return ledger.RecordExternalBridgeInputSubmittedV0(ctx, entry)
}

func (ledger *failingSubmitExternalBridgeInputLedgerV0) LookupExternalBridgeInputV0(
	_ context.Context,
	key string,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	entry, ok := ledger.entries[key]
	return entry, ok, nil
}

func (ledger *failingSubmitExternalBridgeInputLedgerV0) ClaimExternalBridgeInputV0(
	_ context.Context,
	entry externalBridgeInputLedgerEntryV0,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	entry.Status = externalBridgeInputStatusClaimedV0
	ledger.entries[entry.Key] = entry
	return entry, true, nil
}

func (ledger *failingSubmitExternalBridgeInputLedgerV0) RecordExternalBridgeInputSubmittedV0(
	context.Context,
	externalBridgeInputLedgerEntryV0,
) error {
	return fmt.Errorf("external_bridge_input_ledger_write_error")
}

func (ledger *failingSubmitExternalBridgeInputLedgerV0) UpsertExternalBridgeInputV0(
	ctx context.Context,
	entry externalBridgeInputLedgerEntryV0,
) error {
	return ledger.RecordExternalBridgeInputSubmittedV0(ctx, entry)
}
