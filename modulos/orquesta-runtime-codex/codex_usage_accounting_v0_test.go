package orquestaruntimecodex

import "testing"

func TestBuildCodexUsageAccountingSnapshotV0ParseaUsoRedactado(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"prompt_tokens":1000,"completion_tokens":250,"total_tokens":1250}`,
		"quota status: limited\nquota remaining: 42\nquota limit: 100\naccess_token=abc123",
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaLimitedV0 ||
		snapshot.QuotaRemaining != 42 ||
		snapshot.QuotaLimit != 100 ||
		snapshot.PromptTokens != 1000 ||
		snapshot.CompletionTokens != 250 ||
		snapshot.TotalTokens != 1250 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0ParseaTokensUsed(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		"turn interrupted\nTokens used\n47,498\nusage limit reached",
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaExhaustedV0 ||
		snapshot.TotalTokens != 47498 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0ParseaUsageLimitActual(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		"ERROR: You've hit your usage limit. Upgrade to Pro or try again at 1:57 PM.",
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaExhaustedV0 ||
		snapshot.QuotaReason != "" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0ParseaAliasesDeProveedor(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"input_tokens":1200,"output_tokens":300},"quota_status": "available"}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaAvailableV0 ||
		snapshot.PromptTokens != 1200 ||
		snapshot.CompletionTokens != 300 ||
		snapshot.TotalTokens != 1500 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0UsoSinCuotaQuedaUnknown(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"input_tokens":1200,"output_tokens":300}}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaUnknownV0 ||
		snapshot.QuotaReason != CodexUsageQuotaReasonNotReportedV0 ||
		snapshot.QuotaRemaining != 0 ||
		snapshot.QuotaLimit != 0 ||
		snapshot.TotalTokens != 1500 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0CuotaUnknownReportadaConservaMotivo(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"total_tokens":1200},"quota":{"status":"unknown"}}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaUnknownV0 ||
		snapshot.QuotaReason != CodexUsageQuotaReasonUnknownReportedV0 ||
		snapshot.TotalTokens != 1200 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0CuotaUnavailableConservaMotivo(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{},"quota":{"status":"unknown","reason":"quota_observed_unavailable"}}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaUnknownV0 ||
		snapshot.QuotaReason != CodexUsageQuotaReasonObservedUnavailableV0 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0CuotaUnavailableReportadaNoQuedaNotConfigured(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"total_tokens":1200},"quota":{"status":"unavailable"}}`,
		"quota status: unavailable",
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaUnknownV0 ||
		snapshot.QuotaReason != CodexUsageQuotaReasonObservedUnavailableV0 ||
		snapshot.TotalTokens != 1200 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0ParseaReporteJSONRedactado(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"input_tokens":2000,"output_tokens":500},` +
			`"quota":{"status":"rate_limited","remaining_tokens":7,"limit_tokens":100}}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaLimitedV0 ||
		snapshot.QuotaRemaining != 7 ||
		snapshot.QuotaLimit != 100 ||
		snapshot.PromptTokens != 2000 ||
		snapshot.CompletionTokens != 500 ||
		snapshot.TotalTokens != 2500 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestBuildCodexUsageAccountingSnapshotV0ParseaNumerosJSONComoString(t *testing.T) {
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{
		`{"usage":{"input_tokens":"2,000","output_tokens":"500"},` +
			`"quota":{"status":"available","remaining_tokens":"7","limit_tokens":"100"}}`,
	})
	if !snapshot.Observed ||
		snapshot.QuotaStatus != CodexUsageQuotaAvailableV0 ||
		snapshot.QuotaRemaining != 7 ||
		snapshot.QuotaLimit != 100 ||
		snapshot.PromptTokens != 2000 ||
		snapshot.CompletionTokens != 500 ||
		snapshot.TotalTokens != 2500 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestCodexUsageAccountingReportRedactedV0(t *testing.T) {
	if !CodexUsageAccountingReportRedactedV0(
		`{"usage":{"input_tokens":2000,"output_tokens":500},"quota":{"status":"available"}}`,
	) {
		t.Fatalf("reporte redactado rechazado")
	}
	if CodexUsageAccountingReportRedactedV0(
		`{"usage":{"total_tokens":10},"prompt":"texto crudo"}`,
	) {
		t.Fatalf("prompt crudo aceptado")
	}
	if CodexUsageAccountingReportRedactedV0("tokens used: 10") {
		t.Fatalf("reporte no JSON aceptado")
	}
	if CodexUsageAccountingReportRedactedV0(
		`{"usage":{"total_tokens":10},"quota":{"status":"available","note":"access_token=abc"}}`,
	) {
		t.Fatalf("token crudo aceptado")
	}
	if CodexUsageAccountingReportRedactedV0(
		`{"usage":{"total_tokens":10},"provider":"codex","model":"gpt-x","cost_micros":7}`,
	) {
		t.Fatalf("provider/model/coste aceptado")
	}
	if CodexUsageAccountingReportRedactedV0(
		`{"usage":{"total_tokens":10},"quota":{"status":"available","account":"user@example.test"}}`,
	) {
		t.Fatalf("cuenta real aceptada")
	}
	if CodexUsageAccountingReportRedactedV0(
		`{"usage":{"total_tokens":10},"working_dir":"/tmp/project"}`,
	) {
		t.Fatalf("ruta absoluta aceptada")
	}
}
