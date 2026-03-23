package runtimeobs

import (
	"encoding/json"
	"testing"
)

func TestNormalizeGenericProcess(t *testing.T) {
	payload := json.RawMessage(`{
		"runtime_ref":"proc:123",
		"pid":123,
		"command":"codex",
		"argv":["codex","exec"],
		"cwd":"/repo",
		"running":true,
		"cpu_pct":1.5,
		"rss_bytes":4096
	}`)

	sample, err := Normalize("generic_process", payload)
	if err != nil {
		t.Fatalf("Normalize generic_process: %v", err)
	}
	if sample.Provider != "generic_process" || sample.Kind != "process_snapshot" {
		t.Fatalf("sample inesperada: %+v", sample)
	}
	if sample.State != "running" {
		t.Fatalf("state inesperado: %s", sample.State)
	}
	if sample.PID == nil || *sample.PID != 123 {
		t.Fatalf("pid inesperado: %+v", sample.PID)
	}
	if sample.Details["cpu_pct"] != 1.5 {
		t.Fatalf("details inesperados: %+v", sample.Details)
	}
}

func TestNormalizeCodexCLI(t *testing.T) {
	payload := json.RawMessage(`{
		"runtime_ref":"codex:alpha",
		"session_id":"sess-1",
		"model":"gpt-5.4",
		"cwd":"/repo",
		"command":"codex",
		"active":true
	}`)

	sample, err := Normalize("codex_cli", payload)
	if err != nil {
		t.Fatalf("Normalize codex_cli: %v", err)
	}
	if sample.Provider != "codex_cli" || sample.Kind != "cli_session_snapshot" {
		t.Fatalf("sample inesperada: %+v", sample)
	}
	if sample.State != "running" {
		t.Fatalf("state inesperado: %s", sample.State)
	}
	if sample.SessionRef != "sess-1" || sample.Model != "gpt-5.4" {
		t.Fatalf("sample inesperada: %+v", sample)
	}
}

func TestNormalizeUnknownProvider(t *testing.T) {
	if _, err := Normalize("otro", json.RawMessage(`{}`)); err == nil {
		t.Fatalf("se esperaba error por proveedor desconocido")
	}
}
