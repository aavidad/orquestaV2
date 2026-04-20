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
