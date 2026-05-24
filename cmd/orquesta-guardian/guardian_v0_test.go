package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuardianV0PromocionaCandidatoYGuardaLastGood(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf candidate > {candidate_bin}",
		TestCommands:   []string{"true"},
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusPromotedV0 || !result.Promoted {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "candidate" {
		t.Fatalf("current=%q", got)
	}
	if got := mustReadGuardianTestFileV0(t, config.LastGoodBin); got != "old" {
		t.Fatalf("last_good=%q", got)
	}
	if _, err := os.Stat(config.ManifestPath); err != nil {
		t.Fatalf("manifest: %v", err)
	}
}

func TestGuardianV0BuildFallidoNoPromocionaYCreaRepairPacket(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	repairSeen := filepath.Join(dir, "repair_seen.txt")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf build-failed >&2; exit 7",
		RepairCommand:  "printf \"$ORQUESTA_GUARDIAN_REPAIR_PACKET\" > " + shellQuoteV0(repairSeen),
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 1, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 ||
		result.Phase != "build" ||
		result.RepairPacketPath == "" ||
		!result.RepairStarted {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
	if got := mustReadGuardianTestFileV0(t, repairSeen); got != result.RepairPacketPath {
		t.Fatalf("repair_seen=%q packet=%q", got, result.RepairPacketPath)
	}
	packet := mustReadGuardianRepairPacketForTestV0(t, result.RepairPacketPath)
	if packet.FailurePhase != "build" || !strings.Contains(packet.Summary, "build") {
		t.Fatalf("packet=%+v", packet)
	}
}

func TestGuardianV0BuildFallidoPuedeLanzarCodexReparadorOptIn(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	argsPath := filepath.Join(dir, "codex_args.txt")
	script := "#!/bin/sh\nprintf \"%s\\n\" \"$@\" > " + shellQuoteV0(argsPath) + "\n"
	mustWriteGuardianTestFileV0(t, current, script)
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "exit 7",
		RepairCodex:    true,
		SkipHealth:     true,
		Promote:        true,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 6, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || !result.RepairStarted {
		t.Fatalf("result=%+v", result)
	}
	args := mustReadGuardianTestFileV0(t, argsPath)
	for _, want := range []string{"codex-launch-wave", "--agents", "1", "--prompt-file"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args no contiene %q: %s", want, args)
		}
	}
	promptPath := strings.TrimSuffix(result.RepairPacketPath, filepath.Ext(result.RepairPacketPath)) + ".md"
	if _, err := os.Stat(promptPath); err != nil {
		t.Fatalf("repair prompt: %v", err)
	}
}

func TestGuardianV0HealthcheckFallidoNoPromociona(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	mustWriteGuardianTestFileV0(t, current, "old")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		BuildCommand:   "printf '#!/bin/sh\\nexit 0\\n' > {candidate_bin}; chmod +x {candidate_bin}",
		Promote:        true,
		SkipHealth:     false,
		HealthTimeout:  100 * time.Millisecond,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 2, 0, 0, time.UTC),
	})

	result := runGuardianCheckPromoteV0(context.Background(), config)
	if result.Status != guardianStatusCandidateFailedV0 || result.Phase != "healthcheck" {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "old" {
		t.Fatalf("current modificado=%q", got)
	}
	if result.RepairPacketPath == "" {
		t.Fatalf("repair packet vacio: %+v", result)
	}
}

func TestGuardianV0RestauraLastGood(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "bin", "orquesta-server-latest")
	config := mustGuardianConfigForTestV0(t, guardianConfigV0{
		ProjectDir:     dir,
		StateDir:       filepath.Join(dir, "guardian"),
		CurrentBin:     current,
		CommandTimeout: time.Second,
		OccurredAt:     time.Date(2026, 5, 24, 12, 3, 0, 0, time.UTC),
	})
	mustWriteGuardianTestFileV0(t, config.LastGoodBin, "last-good")

	result := restoreLastGoodCommandV0(config)
	if result.Status != guardianStatusRestoredV0 || !result.Restored {
		t.Fatalf("result=%+v", result)
	}
	if got := mustReadGuardianTestFileV0(t, current); got != "last-good" {
		t.Fatalf("current=%q", got)
	}
}

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
		CurrentBin:         filepath.Join(dir, "bin", "orquesta-server-latest"),
		ServerAddr:         strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout:    time.Second,
		ShutdownQueueLimit: 123,
		OccurredAt:         time.Date(2026, 5, 24, 12, 4, 0, 0, time.UTC),
	})

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusShutdownReadyV0 ||
		result.Shutdown == nil ||
		!result.Shutdown.ShutdownReady {
		t.Fatalf("result=%+v", result)
	}
	if received.Forced || received.RequestedBy != "orquesta-guardian" || received.QueueLimit != 123 {
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
		ProjectDir:        dir,
		StateDir:          filepath.Join(dir, "guardian"),
		CurrentBin:        filepath.Join(dir, "bin", "orquesta-server-latest"),
		ServerAddr:        strings.TrimPrefix(server.URL, "http://"),
		ShutdownTimeout:   time.Millisecond,
		ForceAfterTimeout: true,
		OccurredAt:        time.Date(2026, 5, 24, 12, 5, 0, 0, time.UTC),
	})

	result := runGuardianShutdownServerV0(context.Background(), config)
	if result.Status != guardianStatusShutdownReadyV0 ||
		result.Shutdown == nil ||
		!result.Shutdown.ShutdownReady {
		t.Fatalf("result=%+v", result)
	}
	if len(forcedValues) == 0 || !forcedValues[len(forcedValues)-1] {
		t.Fatalf("forcedValues=%+v", forcedValues)
	}
}

func mustGuardianConfigForTestV0(t *testing.T, config guardianConfigV0) guardianConfigV0 {
	t.Helper()
	normalized, err := normalizeGuardianConfigV0(config)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	return normalized
}

func mustWriteGuardianTestFileV0(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadGuardianTestFileV0(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func mustReadGuardianRepairPacketForTestV0(t *testing.T, path string) guardianRepairPacketV0 {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read packet: %v", err)
	}
	var packet guardianRepairPacketV0
	if err := json.Unmarshal(body, &packet); err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	return packet
}
