package orquestaruntime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExternalAgentConnectorProfileV0ValidaOptInCerrado(t *testing.T) {
	profile := externalAgentConnectorProfileValidoV0()

	if issues := ValidateExternalAgentConnectorProfileV0(profile); len(issues) > 0 {
		t.Fatalf("profile invalido: %+v", issues)
	}
}

func TestExternalAgentConnectorProfileV0ExigeRefsYPoliticaCerrada(t *testing.T) {
	profile := externalAgentConnectorProfileValidoV0()
	profile.ConnectorRef = ""
	profile.RuntimeKind = ""
	profile.LaunchMode = ""
	profile.Security.OptIn = false
	profile.Security.ShellPolicy = "allowed"

	issues := ValidateExternalAgentConnectorProfileV0(profile)

	requireExternalAgentCodeV0(t, issues, ExternalAgentProfileInvalidoV0)
	requireExternalAgentCodeV0(t, issues, ExternalAgentSecurityInvalidaV0)
}

func TestExternalAgentConnectorProfileV0RechazaPoliticaAbiertaAunqueRailsDetalleOff(t *testing.T) {
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "off")
	profile := externalAgentConnectorProfileValidoV0()
	profile.Security.OptIn = false
	profile.Security.ShellPolicy = "allowed"
	profile.Security.PathInheritancePolicy = "allowed"
	profile.Security.EnvironmentPolicy = "inherited"
	profile.Security.NetworkPolicy = "open"
	profile.Security.TranscriptsPolicy = "allowed"

	requireExternalAgentCodeV0(t, ValidateExternalAgentConnectorProfileV0(profile), ExternalAgentSecurityInvalidaV0)
}

func TestExternalAgentConnectorProfileV0RechazaDetallesOperacionales(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ExternalAgentConnectorProfileV0)
	}{
		{
			name: "provider concreto",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.ArgRefs = []string{"openai"}
			},
		},
		{
			name: "modelo concreto",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.ArgRefs = []string{"gpt-5"}
			},
		},
		{
			name: "shell",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.ExecutableRef = "bash"
			},
		},
		{
			name: "home real",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.ExecutableRef = "/home/alberto/.local/bin/agent"
			},
		},
		{
			name: "token",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.EnvRefs = []string{"access_token=abc"}
			},
		},
		{
			name: "secret",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.EnvRefs = []string{"sk-test-secret"}
			},
		},
		{
			name: "ruta absoluta peligrosa",
			mutate: func(profile *ExternalAgentConnectorProfileV0) {
				profile.Command.WorkingDirRef = "/etc/orquesta"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := externalAgentConnectorProfileValidoV0()
			tc.mutate(&profile)

			issues := ValidateExternalAgentConnectorProfileV0(profile)

			requireExternalAgentCodeV0(t, issues, ExternalAgentDetalleProhibidoV0)
		})
	}
}

func TestExternalAgentConnectorProfileV0PermiteRefsOperativasOpacas(t *testing.T) {
	profile := externalAgentConnectorProfileValidoV0()
	profile.Command.ArgRefs = []string{
		"provider-ref-001",
		"model-ref-001",
		"home-ref-001",
		"credential-ref-001",
		"transcript-ref-001",
	}
	profile.Command.EnvRefs = []string{"oauth-token-ref-001"}

	if issues := ValidateExternalAgentConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("refs opacas operativas rechazadas: %+v", issues)
	}
}

func TestBuildExternalAgentLaunchSpecV0GeneraSpecOptInSinDetallesDeProveedor(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	packet := BuildAgentStartPacketV0(request, runtimeMaterializedContextValidoV0(t, *request.ContextBundle))
	profile := externalAgentConnectorProfileValidoV0()

	spec := BuildExternalAgentLaunchSpecV0(request, packet, profile)
	if !spec.Valid() {
		t.Fatalf("spec invalida: %+v", spec.Issues)
	}
	if spec.ConnectorRef != request.RuntimeBinding.ConnectorRef ||
		spec.RuntimeKind != request.RuntimeBinding.RuntimeKind ||
		spec.LaunchMode != request.LaunchMode {
		t.Fatalf("spec no conserva refs runtime permitidas: %+v", spec)
	}
	assertExternalAgentSpecNoOperationalDetailsV0(t, spec, request)
}

func TestBuildExternalAgentLaunchSpecV0RechazaPacketQueNoCoincide(t *testing.T) {
	request := runtimeLaunchRequestValidaV0()
	packet := BuildAgentStartPacketV0(request, runtimeMaterializedContextValidoV0(t, *request.ContextBundle))
	packet.RequestID = "req-runtime-otra"

	spec := BuildExternalAgentLaunchSpecV0(request, packet, externalAgentConnectorProfileValidoV0())

	requireExternalAgentCodeV0(t, spec.Issues, ExternalAgentPacketInvalidoV0)
}

func externalAgentConnectorProfileValidoV0() ExternalAgentConnectorProfileV0 {
	request := runtimeLaunchRequestValidaV0()
	return ExternalAgentConnectorProfileV0{
		SchemaVersion: ExternalAgentConnectorProfileSchemaVersionV0,
		ProfileRef:    "external-agent-profile-001",
		ConnectorRef:  request.RuntimeBinding.ConnectorRef,
		RuntimeKind:   request.RuntimeBinding.RuntimeKind,
		LaunchMode:    request.LaunchMode,
		Command: ExternalAgentCommandV0{
			CommandRef:    "external-agent-command-001",
			ExecutableRef: "external-agent-executable-001",
			ArgRefs:       []string{"external-agent-arg-001"},
			EnvRefs:       []string{"external-agent-env-001"},
			WorkingDirRef: "external-agent-workdir-001",
		},
		Security: ExternalAgentSecurityPolicyV0{
			OptIn:                 true,
			ShellPolicy:           ExternalAgentShellForbiddenV0,
			PathInheritancePolicy: ExternalAgentPathInheritanceForbiddenV0,
			EnvironmentPolicy:     ExternalAgentEnvExplicitRefsOnlyV0,
			SecretsPolicy:         ExternalAgentSecretsReferencesOnlyV0,
			HomePathsPolicy:       ExternalAgentHomeOpaqueRefsOnlyV0,
			NetworkPolicy:         ExternalAgentNetworkClosedV0,
			TranscriptsPolicy:     ExternalAgentTranscriptsForbiddenV0,
		},
	}
}

func requireExternalAgentCodeV0(
	t *testing.T,
	issues []ExternalAgentConnectorErrorV0,
	code ExternalAgentConnectorErrorCodeV0,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}

func assertExternalAgentSpecNoOperationalDetailsV0(
	t *testing.T,
	spec ExternalAgentLaunchSpecV0,
	request RuntimeLaunchRequestV0,
) {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		request.RuntimeBinding.ProviderRef,
		request.RuntimeBinding.ModelRef,
		request.RuntimeBinding.HomeRef,
		request.RuntimeBinding.CredentialRef,
		"provider_ref",
		"model_ref",
		"home_ref",
		"credential_ref",
		"oauth_ref",
		"access_token",
		"refresh_token",
		"transcript-ref",
		"transcript=",
	} {
		if forbidden != "" && strings.Contains(lower, strings.ToLower(forbidden)) {
			t.Fatalf("spec filtra detalle prohibido %q: %s", forbidden, string(raw))
		}
	}
}
