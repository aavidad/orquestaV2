package orquestaruntime

import "testing"

func TestBuildClosedExternalAgentConnectorProfileV0CreaPerfilValidable(t *testing.T) {
	profile := BuildClosedExternalAgentConnectorProfileV0(ExternalAgentConnectorProfileRefsV0{
		ProfileRef:    "external-profile-ref-001",
		ConnectorRef:  "external-connector-ref-001",
		RuntimeKind:   "cli",
		CommandRef:    "command-ref-001",
		ExecutableRef: "executable-ref-001",
		WorkingDirRef: "working-dir-ref-001",
		ArgRefs:       []string{"arg-ref-001"},
		EnvRefs:       []string{"env-ref-001"},
	})

	if issues := ValidateExternalAgentConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("perfil invalido: %+v", issues)
	}
	if profile.Security.SecretsPolicy != ExternalAgentSecretsReferencesOnlyV0 ||
		profile.Security.HomePathsPolicy != ExternalAgentHomeOpaqueRefsOnlyV0 ||
		profile.Security.TranscriptsPolicy != ExternalAgentTranscriptsForbiddenV0 {
		t.Fatalf("politica no cerrada: %+v", profile.Security)
	}
	if len(profile.Command.ArgRefs) != 1 || len(profile.Command.EnvRefs) != 1 {
		t.Fatalf("refs de comando inesperadas: %+v", profile.Command)
	}
}

func TestBuildClosedExternalAgentConnectorProfileV0NoOcultaRefsInvalidas(t *testing.T) {
	profile := BuildClosedExternalAgentConnectorProfileV0(ExternalAgentConnectorProfileRefsV0{
		ProfileRef:    "external-profile-ref-001",
		ConnectorRef:  "provider=openai",
		RuntimeKind:   "cli",
		CommandRef:    "command-ref-001",
		ExecutableRef: "executable-ref-001",
	})

	requireExternalAgentCodeV0(
		t,
		ValidateExternalAgentConnectorProfileV0(profile),
		ExternalAgentDetalleProhibidoV0,
	)
}
