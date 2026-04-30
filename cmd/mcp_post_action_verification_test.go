package cmd

import (
	"testing"
	"time"
)

func TestBuildSupervisorPostActionVerificationPrefiereOperationalMasFrescoDelSnapshot(t *testing.T) {
	prevBuilder := supervisorVerificationOperationalBuilder
	prevSnapshotBuilder := supervisorOperatorSnapshotBuilder
	t.Cleanup(func() {
		supervisorVerificationOperationalBuilder = prevBuilder
		supervisorOperatorSnapshotBuilder = prevSnapshotBuilder
	})

	supervisorVerificationOperationalBuilder = func() (serverOperationalInfo, error) {
		return serverOperationalInfo{
			State:         "degraded",
			Operational:   false,
			Reason:        "tasks_without_workers",
			Generated:     "2026-04-30T09:52:21Z",
			ActiveAgents:  11,
			WorkingAgents: 0,
		}, nil
	}
	supervisorOperatorSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"server_operational": serverOperationalInfo{
				State:            "ready",
				Operational:      true,
				Reason:           "control_plane_responsive",
				Generated:        "2026-04-30T09:52:24Z",
				ActiveAgents:     2,
				WorkingAgents:    2,
				ConnectedWorkers: 1,
				WorkingWorkers:   1,
			},
		}, nil
	}

	verification := buildSupervisorPostActionVerification("OpenClaw")
	info, _ := verification["server_operational"].(serverOperationalInfo)
	if info.Generated != "2026-04-30T09:52:24Z" || !info.Operational || info.State != "ready" {
		t.Fatalf("server_operational deberia preferir snapshot mas fresco: %#v", verification["server_operational"])
	}
}

func TestSupervisorOperationalSnapshotShouldReplaceVerification(t *testing.T) {
	current := serverOperationalInfo{Generated: "2026-04-30T09:52:21Z"}
	candidate := serverOperationalInfo{Generated: "2026-04-30T09:52:24Z"}
	if !supervisorOperationalSnapshotShouldReplaceVerification(current, candidate) {
		t.Fatal("snapshot mas fresco deberia reemplazar verification")
	}
	if supervisorOperationalSnapshotShouldReplaceVerification(candidate, current) {
		t.Fatal("snapshot mas viejo no deberia reemplazar verification")
	}
	if !supervisorOperationalSnapshotShouldReplaceVerification(serverOperationalInfo{}, candidate) {
		t.Fatal("sin generated previo deberia aceptar snapshot")
	}
	if supervisorOperationalSnapshotShouldReplaceVerification(current, serverOperationalInfo{}) {
		t.Fatal("snapshot sin generated no deberia reemplazar verification")
	}
	if !supervisorOperationalSnapshotShouldReplaceVerification(current, serverOperationalInfo{Generated: "2026-04-30T09:52:21Z"}) {
		t.Fatal("snapshot con mismo generated deberia reemplazar verification")
	}
}

func TestBuildSupervisorPostActionVerificationMantieneOperationalActualSiSnapshotViejo(t *testing.T) {
	prevBuilder := supervisorVerificationOperationalBuilder
	prevSnapshotBuilder := supervisorOperatorSnapshotBuilder
	t.Cleanup(func() {
		supervisorVerificationOperationalBuilder = prevBuilder
		supervisorOperatorSnapshotBuilder = prevSnapshotBuilder
	})

	supervisorVerificationOperationalBuilder = func() (serverOperationalInfo, error) {
		return serverOperationalInfo{
			State:            "ready",
			Operational:      true,
			Reason:           "control_plane_responsive",
			Generated:        "2026-04-30T09:52:24Z",
			ActiveAgents:     2,
			WorkingAgents:    2,
			ConnectedWorkers: 1,
			WorkingWorkers:   1,
		}, nil
	}
	supervisorOperatorSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"server_operational": serverOperationalInfo{
				State:         "degraded",
				Operational:   false,
				Reason:        "tasks_without_workers",
				Generated:     "2026-04-30T09:52:21Z",
				ActiveAgents:  11,
				WorkingAgents: 0,
			},
			"verified_at": time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	verification := buildSupervisorPostActionVerification("OpenClaw")
	info, _ := verification["server_operational"].(serverOperationalInfo)
	if info.Generated != "2026-04-30T09:52:24Z" || !info.Operational || info.State != "ready" {
		t.Fatalf("server_operational no deberia degradarse con snapshot viejo: %#v", verification["server_operational"])
	}
}
