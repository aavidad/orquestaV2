package orquestaruntime

import "fmt"

const (
	ExternalAgentConnectorProfileSchemaVersionV0 = "external_agent_connector_profile.v0"
	ExternalAgentLaunchSpecSchemaVersionV0       = "external_agent_launch_spec.v0"

	ExternalAgentShellForbiddenV0           = "forbidden"
	ExternalAgentPathInheritanceForbiddenV0 = "forbidden"
	ExternalAgentEnvExplicitRefsOnlyV0      = "explicit_refs_only"
	ExternalAgentSecretsReferencesOnlyV0    = "references_only"
	ExternalAgentHomeOpaqueRefsOnlyV0       = "opaque_refs_only"
	ExternalAgentNetworkClosedV0            = "closed_by_default"
	ExternalAgentTranscriptsForbiddenV0     = "forbidden"
)

type ExternalAgentConnectorErrorCodeV0 string

const (
	ExternalAgentProfileInvalidoV0    ExternalAgentConnectorErrorCodeV0 = "external_agent_connector_profile_invalido"
	ExternalAgentLaunchSpecInvalidoV0 ExternalAgentConnectorErrorCodeV0 = "external_agent_launch_spec_invalido"
	ExternalAgentCommandInvalidoV0    ExternalAgentConnectorErrorCodeV0 = "external_agent_command_invalido"
	ExternalAgentSecurityInvalidaV0   ExternalAgentConnectorErrorCodeV0 = "security_policy_closed_required"
	ExternalAgentReferenciaNoOpacaV0  ExternalAgentConnectorErrorCodeV0 = "referencia_no_opaca"
	ExternalAgentDetalleProhibidoV0   ExternalAgentConnectorErrorCodeV0 = "detalle_operacional_prohibido"
	ExternalAgentPacketInvalidoV0     ExternalAgentConnectorErrorCodeV0 = "agent_start_packet_invalido"
	ExternalAgentRequestInvalidaV0    ExternalAgentConnectorErrorCodeV0 = "runtime_launch_request_invalida"
)

type ExternalAgentConnectorProfileV0 struct {
	SchemaVersion string                        `json:"schema_version"`
	ProfileRef    string                        `json:"profile_ref"`
	ConnectorRef  string                        `json:"connector_ref"`
	RuntimeKind   string                        `json:"runtime_kind"`
	LaunchMode    string                        `json:"launch_mode"`
	Command       ExternalAgentCommandV0        `json:"command"`
	Security      ExternalAgentSecurityPolicyV0 `json:"security"`
}

type ExternalAgentCommandV0 struct {
	CommandRef    string   `json:"command_ref"`
	ExecutableRef string   `json:"executable_ref"`
	ArgRefs       []string `json:"arg_refs,omitempty"`
	EnvRefs       []string `json:"env_refs,omitempty"`
	WorkingDirRef string   `json:"working_dir_ref,omitempty"`
}

type ExternalAgentSecurityPolicyV0 struct {
	OptIn                 bool   `json:"opt_in"`
	ShellPolicy           string `json:"shell_policy"`
	PathInheritancePolicy string `json:"path_inheritance_policy"`
	EnvironmentPolicy     string `json:"environment_policy"`
	SecretsPolicy         string `json:"secrets_policy"`
	HomePathsPolicy       string `json:"home_paths_policy"`
	NetworkPolicy         string `json:"network_policy"`
	TranscriptsPolicy     string `json:"transcripts_policy"`
}

type ExternalAgentLaunchSpecV0 struct {
	SchemaVersion string                          `json:"schema_version"`
	RequestID     string                          `json:"request_id"`
	CorrelationID string                          `json:"correlation_id"`
	ProfileRef    string                          `json:"profile_ref"`
	ConnectorRef  string                          `json:"connector_ref"`
	RuntimeKind   string                          `json:"runtime_kind"`
	LaunchMode    string                          `json:"launch_mode"`
	Command       ExternalAgentCommandV0          `json:"command"`
	AgentPacket   AgentStartPacketV0              `json:"agent_packet"`
	Security      ExternalAgentSecurityPolicyV0   `json:"security"`
	Issues        []ExternalAgentConnectorErrorV0 `json:"issues,omitempty"`
}

type ExternalAgentConnectorErrorV0 struct {
	Code          ExternalAgentConnectorErrorCodeV0 `json:"code"`
	MessageKey    string                            `json:"message_key"`
	Field         string                            `json:"field,omitempty"`
	Retryable     bool                              `json:"retryable"`
	CorrelationID string                            `json:"correlation_id,omitempty"`
	Evidence      []string                          `json:"evidence,omitempty"`
}

func (e ExternalAgentConnectorErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func (profile ExternalAgentConnectorProfileV0) Validate() []ExternalAgentConnectorErrorV0 {
	return ValidateExternalAgentConnectorProfileV0(profile)
}

func (profile ExternalAgentConnectorProfileV0) Valid() bool {
	return len(ValidateExternalAgentConnectorProfileV0(profile)) == 0
}

func (spec ExternalAgentLaunchSpecV0) Validate() []ExternalAgentConnectorErrorV0 {
	return ValidateExternalAgentLaunchSpecV0(spec)
}

func (spec ExternalAgentLaunchSpecV0) Valid() bool {
	return len(ValidateExternalAgentLaunchSpecV0(spec)) == 0
}
