package controlruntime

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordWorkQueueFromMetadataJSONAndMarkState(t *testing.T) {
	dir := t.TempDir()
	metaRaw, _ := json.Marshal(map[string]any{
		"trace_dir": dir,
	})
	recordedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	if err := RecordWorkQueueFromMetadataJSON(string(metaRaw), WorkQueueRecordInput{
		MailboxID:       41,
		RuntimeOrderID:  99,
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		TaskID:          123,
		VerificationKey: "vk-1",
		State:           "pending",
		Title:           "seguir frente",
		Reason:          "inbox durable",
		RecordedAt:      recordedAt,
	}); err != nil {
		t.Fatalf("RecordWorkQueueFromMetadataJSON: %v", err)
	}
	if err := MarkWorkQueueStateFromMetadataJSON(string(metaRaw), 41, "consumed", "ack", recordedAt.Add(time.Minute)); err != nil {
		t.Fatalf("MarkWorkQueueStateFromMetadataJSON: %v", err)
	}
	data, err := loadWorkQueueFile(filepath.Join(dir, "work-queue.json"))
	if err != nil {
		t.Fatalf("loadWorkQueueFile: %v", err)
	}
	if data.Current == nil || data.Current.MailboxID != 41 {
		t.Fatalf("current inesperado: %+v", data.Current)
	}
	if data.Current.State != "consumed" || data.Current.Reason != "ack" {
		t.Fatalf("current no actualizado: %+v", data.Current)
	}
	if len(data.Recent) != 1 || data.Recent[0].State != "consumed" {
		t.Fatalf("recent inesperado: %+v", data.Recent)
	}
}

func TestHasPendingAndRecentWorkQueueEntryFromMetadataJSON(t *testing.T) {
	dir := t.TempDir()
	metaRaw, _ := json.Marshal(map[string]any{
		"trace_dir": dir,
	})
	now := time.Now().UTC()
	if err := RecordWorkQueueFromMetadataJSON(string(metaRaw), WorkQueueRecordInput{
		MailboxID:       41,
		RuntimeOrderID:  99,
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		TaskID:          123,
		VerificationKey: "vk-1",
		State:           "pending",
		Title:           "seguir frente",
		Reason:          "inbox durable",
		RecordedAt:      now,
	}); err != nil {
		t.Fatalf("RecordWorkQueueFromMetadataJSON: %v", err)
	}
	match := WorkQueueMatch{
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		TaskID:          123,
		VerificationKey: "vk-1",
		Within:          time.Minute,
	}
	if ok, err := HasPendingWorkQueueEntryFromMetadataJSON(string(metaRaw), match); err != nil || !ok {
		t.Fatalf("HasPendingWorkQueueEntryFromMetadataJSON=%v err=%v", ok, err)
	}
	if ok, err := HasRecentWorkQueueEntryFromMetadataJSON(string(metaRaw), match); err != nil || !ok {
		t.Fatalf("HasRecentWorkQueueEntryFromMetadataJSON=%v err=%v", ok, err)
	}
	if err := MarkWorkQueueStateFromMetadataJSON(string(metaRaw), 41, "consumed", "ack", now.Add(30*time.Second)); err != nil {
		t.Fatalf("MarkWorkQueueStateFromMetadataJSON: %v", err)
	}
	if ok, err := HasPendingWorkQueueEntryFromMetadataJSON(string(metaRaw), match); err != nil {
		t.Fatalf("HasPendingWorkQueueEntryFromMetadataJSON consumed: %v", err)
	} else if ok {
		t.Fatalf("no deberia seguir pendiente tras consumed")
	}
	if ok, err := HasRecentWorkQueueEntryFromMetadataJSON(string(metaRaw), match); err != nil || !ok {
		t.Fatalf("HasRecentWorkQueueEntryFromMetadataJSON consumed=%v err=%v", ok, err)
	}
}
