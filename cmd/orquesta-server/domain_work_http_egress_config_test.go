package main

import (
	"testing"

	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestDomainWorkExecutorFromEnvV0RechazaHTTPNeutralSinPoliticaEgress(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != orquestadomainworkhttp.ErrDomainWorkHTTPEgressPolicyRequiredV0 || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaHTTPNeutralDeclaradoComoOPES(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "https://opes.example.com")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "allowlist")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_ALLOWED_HOSTS", "opes.example.com")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_DOMAIN_REF", "opes")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "opes_destination_productive_not_allowed" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}
