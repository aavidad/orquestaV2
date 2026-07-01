package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0HealthcheckExigeReadinessOperativa(t *testing.T) {
	withGuardianFakeReadinessHTTPClientV0(t, "not_ready")
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	candidate := writeGuardianFakeCandidateScriptV0(t, dir, "not_ready")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		CandidateBin:   candidate,
		CandidateAddr:  "127.0.0.1:1",
		BuildCommand:   "true",
		Promote:        true,
		SkipHealth:     false,
		HealthTimeout:  2 * time.Second,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || result.Phase != "healthcheck" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
	public := publicGuardianResultV0(config, result)
	if !containsStringForGuardianTestV0(public.ReasonCodes, guardianCandidateReadinessNotReadyV0) ||
		len(public.Commands) == 0 ||
		public.Commands[len(public.Commands)-1].ReasonCode != guardianCandidateReadinessNotReadyV0 {
		t.Fatalf("public=%+v", public)
	}
}

func TestGuardianV0HealthcheckReadinessListaPermitePromocion(t *testing.T) {
	withGuardianFakeReadinessHTTPClientV0(t, "ready")
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	candidate := writeGuardianFakeCandidateScriptV0(t, dir, "ready")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		CandidateBin:   candidate,
		CandidateAddr:  "127.0.0.1:1",
		BuildCommand:   "true",
		Promote:        false,
		SkipHealth:     false,
		HealthTimeout:  2 * time.Second,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 27, 12, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusPromotedV0 || result.Phase != "verified" {
		t.Fatalf("result=%+v", result)
	}
	if !containsStringForGuardianTestV0(result.EvidenceRefs, "evidence-ref-guardian-candidate-readiness-passed") {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	if !containsStringForGuardianTestV0(result.EvidenceRefs, "evidence-ref-guardian-candidate-process-stop-confirmed") {
		t.Fatalf("evidence_refs sin stop confirmado: %+v", result.EvidenceRefs)
	}
	health := result.Commands[len(result.Commands)-1]
	if health.ProcessPolicy == nil || health.StopReceipt == nil {
		t.Fatalf("health sin politica/receipt: %+v", health)
	}
	if health.StopReceipt.ReasonCode == "" || !health.StopReceipt.TreeStopConfirmed || health.StopReceipt.Ambiguous {
		t.Fatalf("stop_receipt=%+v", health.StopReceipt)
	}
	public := publicGuardianResultV0(config, result)
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshal public: %v", err)
	}
	for _, leaked := range []string{dir, current, candidate, "ORQUESTA_SERVER_STATE_DIR"} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("public contiene detalle local %q: %s", leaked, encoded)
		}
	}
}

func TestGuardianFakeCandidateProcessV0(t *testing.T) {
	mode := strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_FAKE_CANDIDATE_MODE"))
	if mode == "" {
		return
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_FAKE_CANDIDATE_NO_LISTEN")) == "1" {
		writeGuardianFakeCandidateStateV0(t, "guardian-test-candidate")
		select {}
	}
	addr := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_ADDR"))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	writeGuardianFakeCandidateStateV0(t, listener.Addr().String())
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v0/server/readiness", func(w http.ResponseWriter, r *http.Request) {
		ready := mode == "ready"
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "orquesta_server_readiness.v0",
			"ready":          ready,
			"status":         map[bool]string{true: "running", false: "running"}[ready],
			"startup_ready":  ready,
			"startup_status": map[bool]string{true: "startup_ready", false: "startup_blocked"}[ready],
			"evidence_refs":  []string{"evidence-ref-fake-candidate-readiness"},
		})
	})
	server := http.Server{Handler: mux}
	_ = server.Serve(listener)
}

func writeGuardianFakeCandidateScriptV0(t *testing.T, dir string, mode string) string {
	t.Helper()
	path := filepath.Join(dir, "candidate.sh")
	body := "#!/bin/sh\nORQUESTA_GUARDIAN_FAKE_CANDIDATE_MODE=" + shellQuoteV0(mode) +
		" ORQUESTA_GUARDIAN_FAKE_CANDIDATE_NO_LISTEN=1" +
		" exec " + shellQuoteV0(os.Args[0]) + " -test.run TestGuardianFakeCandidateProcessV0\n"
	mustWriteGuardianTestFileV0(t, path, body)
	return path
}

func withGuardianFakeReadinessHTTPClientV0(t *testing.T, mode string) {
	t.Helper()
	previous := newGuardianCandidateReadinessHTTPClientV0
	newGuardianCandidateReadinessHTTPClientV0 = func() http.Client {
		return http.Client{Transport: guardianFakeReadinessRoundTripperV0{mode: mode}}
	}
	t.Cleanup(func() {
		newGuardianCandidateReadinessHTTPClientV0 = previous
	})
}

type guardianFakeReadinessRoundTripperV0 struct {
	mode string
}

func (transport guardianFakeReadinessRoundTripperV0) RoundTrip(request *http.Request) (*http.Response, error) {
	status := http.StatusOK
	body := map[string]any{"status": "ok"}
	if request.URL.Path == "/api/v0/server/readiness" {
		ready := transport.mode == "ready"
		if !ready {
			status = http.StatusServiceUnavailable
		}
		body = map[string]any{
			"schema_version": "orquesta_server_readiness.v0",
			"ready":          ready,
			"status":         "running",
			"startup_ready":  ready,
			"startup_status": map[bool]string{true: "startup_ready", false: "startup_blocked"}[ready],
			"evidence_refs":  []string{"evidence-ref-fake-candidate-readiness"},
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(raw)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func writeGuardianFakeCandidateStateV0(t *testing.T, addr string) {
	t.Helper()
	stateDir := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_STATE_DIR"))
	if stateDir == "" {
		return
	}
	body, err := json.Marshal(map[string]any{
		"schema_version": "orquesta_server_state.v0",
		"status":         "running",
		"addr":           addr,
	})
	if err != nil {
		t.Fatalf("state marshal: %v", err)
	}
	path := filepath.Join(stateDir, "orquesta_server_state_v0.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("state mkdir: %v", err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		t.Fatalf("state write: %v", err)
	}
}

func containsStringForGuardianTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
