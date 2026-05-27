package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0ShutdownServerUsaCooperativoPorDefecto(t *testing.T) {
	dir := t.TempDir()
	var received struct {
		Forced      bool   `json:"forced"`
		RequestedBy string `json:"requested_by"`
		QueueLimit  int    `json:"queue_limit"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(guardianShutdownResultV0{
			Estado:        "ok",
			Status:        "ready",
			ShutdownReady: true,
		})
	}))
	defer server.Close()
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:         dir,
		StateDir:           filepath.Join(dir, "guardian"),
		ServerAddr:         strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout:    time.Second,
		ShutdownQueueLimit: 123,
		OccurredAt:         time.Date(2026, 5, 24, 12, 4, 0, 0, time.UTC),
	}, false)

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusShutdownReadyV0 ||
		result.Shutdown == nil ||
		!result.Shutdown.ShutdownReady {
		t.Fatalf("result=%+v", result)
	}
	if received.Forced || received.RequestedBy != "orquesta-director" || received.QueueLimit != 123 {
		t.Fatalf("received=%+v", received)
	}
}

func TestGuardianV0ShutdownServerFuerzaTrasTimeoutCooperativo(t *testing.T) {
	dir := t.TempDir()
	var forcedValues []bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received struct {
			Forced bool `json:"forced"`
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode: %v", err)
		}
		forcedValues = append(forcedValues, received.Forced)
		if received.Forced {
			_ = json.NewEncoder(w).Encode(guardianShutdownResultV0{
				Estado:        "ok",
				Status:        "ready",
				ShutdownReady: true,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(guardianShutdownResultV0{
			Estado:        "ok",
			Status:        "waiting_checkpoint",
			ShutdownReady: false,
		})
	}))
	defer server.Close()
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:                    dir,
		StateDir:                      filepath.Join(dir, "guardian"),
		ServerAddr:                    strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout:               time.Millisecond,
		ForceAfterTimeout:             true,
		ShutdownEscalationEvidenceRef: "evidence-ref-director-break-glass-shutdown",
		OccurredAt:                    time.Date(2026, 5, 24, 12, 5, 0, 0, time.UTC),
	}, false)

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusShutdownReadyV0 ||
		result.Shutdown == nil ||
		!result.Shutdown.ShutdownReady {
		t.Fatalf("result=%+v", result)
	}
	if len(forcedValues) == 0 || !forcedValues[len(forcedValues)-1] {
		t.Fatalf("forcedValues=%+v", forcedValues)
	}
	if result.Shutdown.EscalationStatus != "forced_ready" ||
		result.Shutdown.EscalationEvidenceRef != "evidence-ref-director-break-glass-shutdown" {
		t.Fatalf("shutdown=%+v", result.Shutdown)
	}
}

func TestGuardianV0ShutdownServerNoEscalaPorDefectoTrasTimeout(t *testing.T) {
	dir := t.TempDir()
	var forcedValues []bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received struct {
			Forced bool `json:"forced"`
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode: %v", err)
		}
		forcedValues = append(forcedValues, received.Forced)
		_ = json.NewEncoder(w).Encode(guardianShutdownResultV0{
			Estado:             "ok",
			Status:             "waiting_checkpoint",
			ShutdownReady:      false,
			CheckpointsPending: 1,
		})
	}))
	defer server.Close()
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:      dir,
		StateDir:        filepath.Join(dir, "guardian"),
		ServerAddr:      strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout: time.Millisecond,
		OccurredAt:      time.Date(2026, 5, 24, 12, 6, 0, 0, time.UTC),
	}, false)

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 ||
		result.Shutdown == nil ||
		result.Shutdown.EscalationStatus != "cooperative_timeout" {
		t.Fatalf("result=%+v", result)
	}
	for _, forced := range forcedValues {
		if forced {
			t.Fatalf("forcedValues=%+v", forcedValues)
		}
	}
}

func TestGuardianV0ShutdownServerBloqueaEscaladaSinEvidenceRef(t *testing.T) {
	dir := t.TempDir()
	var forcedValues []bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received struct {
			Forced bool `json:"forced"`
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode: %v", err)
		}
		forcedValues = append(forcedValues, received.Forced)
		_ = json.NewEncoder(w).Encode(guardianShutdownResultV0{
			Estado:         "ok",
			Status:         "waiting_checkpoint",
			ShutdownReady:  false,
			AgentsInFlight: 1,
		})
	}))
	defer server.Close()
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:        dir,
		StateDir:          filepath.Join(dir, "guardian"),
		ServerAddr:        strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout:   time.Millisecond,
		ForceAfterTimeout: true,
		OccurredAt:        time.Date(2026, 5, 24, 12, 7, 0, 0, time.UTC),
	}, false)

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 ||
		result.Message != "guardian_shutdown_escalation_blocked" ||
		result.Shutdown == nil ||
		result.Shutdown.SignalStatus != "signal_blocked" {
		t.Fatalf("result=%+v", result)
	}
	for _, forced := range forcedValues {
		if forced {
			t.Fatalf("forcedValues=%+v", forcedValues)
		}
	}
}
