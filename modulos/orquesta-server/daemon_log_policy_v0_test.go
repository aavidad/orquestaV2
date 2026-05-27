package orquestaserver

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeDaemonLogPolicyV0DefaultsRedactadosV0(t *testing.T) {
	policy := NormalizeDaemonLogPolicyV0(DaemonLogPolicyV0{})

	if policy.Owner != DaemonLogOwnerV0 ||
		policy.PolicyRef != DaemonLogPolicyRefV0 ||
		policy.Mode != DaemonLogModeSummaryRedactedV0 ||
		policy.Access != DaemonLogAccessLocalOnlyV0 {
		t.Fatalf("policy=%+v", policy)
	}
	if policy.MaxBytes <= 0 || policy.MaxRotatedFiles <= 0 || policy.RetentionDays <= 0 {
		t.Fatalf("retencion invalida: %+v", policy)
	}
	if policy.LocalRawEnabled || policy.TerminalEvidence || !policy.RedactionRequired {
		t.Fatalf("raw/evidence/redaction inesperado: %+v", policy)
	}
}

func TestServerPublicStatusV0ExponePoliticaDaemonLogsSinPathsV0(t *testing.T) {
	status := NewServerPublicStatusV0(StateV0{
		Status:         "running",
		ProjectWorkDir: "/home/alberto/proyecto-secreto",
		RuntimeWorkDir: "/home/alberto/runtime-secreto",
		DaemonLogPolicy: DaemonLogPolicyV0{
			LocalRawEnabled: true,
			LocalRawReason:  "/home/alberto/debug local token=abc",
		},
	})
	body, _ := json.Marshal(status)

	for _, forbidden := range []string{"/home/", "proyecto-secreto", "runtime-secreto", "token=abc"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("status filtra %q: %s", forbidden, string(body))
		}
	}
	if status.DaemonLogPolicy.Mode != DaemonLogModeLocalRawOptInV0 ||
		status.DaemonLogPolicy.TerminalEvidence ||
		status.DaemonLogPolicy.LocalRawReason != "local_diagnostic" {
		t.Fatalf("daemon_log_policy=%+v", status.DaemonLogPolicy)
	}
}
