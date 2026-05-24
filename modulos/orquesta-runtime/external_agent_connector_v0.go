package orquestaruntime

import (
	"encoding/json"
	"fmt"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func BuildExternalAgentLaunchSpecV0(
	request RuntimeLaunchRequestV0,
	packet AgentStartPacketV0,
	profile ExternalAgentConnectorProfileV0,
) ExternalAgentLaunchSpecV0 {
	spec := ExternalAgentLaunchSpecV0{
		SchemaVersion: ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     request.RequestID,
		CorrelationID: request.CorrelationID,
		ProfileRef:    profile.ProfileRef,
		ConnectorRef:  profile.ConnectorRef,
		RuntimeKind:   profile.RuntimeKind,
		LaunchMode:    profile.LaunchMode,
		Command:       profile.Command,
		AgentPacket:   packet,
		Security:      profile.Security,
	}
	spec.Issues = append(spec.Issues, externalAgentErrorsFromRuntimeLaunchV0(
		ValidateRuntimeLaunchRequestV0(request),
	)...)
	spec.Issues = append(spec.Issues, ValidateExternalAgentConnectorProfileV0(profile)...)
	spec.Issues = append(spec.Issues, validateExternalAgentPacketForRequestV0(request, packet)...)
	if len(spec.Issues) == 0 {
		spec.Issues = append(spec.Issues, ValidateExternalAgentLaunchSpecV0(spec)...)
	}
	return spec
}

func ValidateExternalAgentConnectorProfileV0(
	profile ExternalAgentConnectorProfileV0,
) []ExternalAgentConnectorErrorV0 {
	v := externalAgentConnectorValidatorV0{}
	v.requireConst(
		"schema_version",
		profile.SchemaVersion,
		ExternalAgentConnectorProfileSchemaVersionV0,
		ExternalAgentProfileInvalidoV0,
	)
	v.requireOpaque("profile_ref", profile.ProfileRef, ExternalAgentProfileInvalidoV0)
	v.requireOpaque("connector_ref", profile.ConnectorRef, ExternalAgentProfileInvalidoV0)
	v.requireRuntimeKind("runtime_kind", profile.RuntimeKind)
	v.requireConst("launch_mode", profile.LaunchMode, RuntimeLaunchModeNewSessionV0, ExternalAgentProfileInvalidoV0)
	v.validateCommand("command", profile.Command)
	v.validateSecurity("security", profile.Security)
	return v.errors
}

func ValidateExternalAgentLaunchSpecV0(spec ExternalAgentLaunchSpecV0) []ExternalAgentConnectorErrorV0 {
	v := externalAgentConnectorValidatorV0{correlationID: spec.CorrelationID}
	v.requireConst(
		"schema_version",
		spec.SchemaVersion,
		ExternalAgentLaunchSpecSchemaVersionV0,
		ExternalAgentLaunchSpecInvalidoV0,
	)
	v.requireOpaque("request_id", spec.RequestID, ExternalAgentLaunchSpecInvalidoV0)
	v.requireOpaque("correlation_id", spec.CorrelationID, ExternalAgentLaunchSpecInvalidoV0)
	v.requireOpaque("profile_ref", spec.ProfileRef, ExternalAgentLaunchSpecInvalidoV0)
	v.requireOpaque("connector_ref", spec.ConnectorRef, ExternalAgentLaunchSpecInvalidoV0)
	v.requireRuntimeKind("runtime_kind", spec.RuntimeKind)
	v.requireConst("launch_mode", spec.LaunchMode, RuntimeLaunchModeNewSessionV0, ExternalAgentLaunchSpecInvalidoV0)
	v.validateCommand("command", spec.Command)
	v.validateSecurity("security", spec.Security)
	if !spec.AgentPacket.Valid() || externalAgentPacketHasForbiddenDetailV0(spec.AgentPacket) {
		v.add(ExternalAgentPacketInvalidoV0, "agent_packet")
	}
	return v.errors
}

type externalAgentConnectorValidatorV0 struct {
	correlationID string
	errors        []ExternalAgentConnectorErrorV0
}

func (v *externalAgentConnectorValidatorV0) validateCommand(field string, command ExternalAgentCommandV0) {
	v.requireOpaque(field+".command_ref", command.CommandRef, ExternalAgentCommandInvalidoV0)
	v.requireOpaque(field+".executable_ref", command.ExecutableRef, ExternalAgentCommandInvalidoV0)
	for i, ref := range command.ArgRefs {
		v.requireOpaque(fmt.Sprintf("%s.arg_refs[%d]", field, i), ref, ExternalAgentCommandInvalidoV0)
	}
	for i, ref := range command.EnvRefs {
		v.requireOpaque(fmt.Sprintf("%s.env_refs[%d]", field, i), ref, ExternalAgentCommandInvalidoV0)
	}
	v.optionalOpaque(field+".working_dir_ref", command.WorkingDirRef)
}

func (v *externalAgentConnectorValidatorV0) validateSecurity(
	field string,
	security ExternalAgentSecurityPolicyV0,
) {
	if !security.OptIn {
		v.add(ExternalAgentSecurityInvalidaV0, field+".opt_in")
	}
	v.requireConst(field+".shell_policy", security.ShellPolicy, ExternalAgentShellForbiddenV0, ExternalAgentSecurityInvalidaV0)
	v.requireConst(
		field+".path_inheritance_policy",
		security.PathInheritancePolicy,
		ExternalAgentPathInheritanceForbiddenV0,
		ExternalAgentSecurityInvalidaV0,
	)
	v.requireConst(
		field+".environment_policy",
		security.EnvironmentPolicy,
		ExternalAgentEnvExplicitRefsOnlyV0,
		ExternalAgentSecurityInvalidaV0,
	)
	v.requireConst(
		field+".secrets_policy",
		security.SecretsPolicy,
		ExternalAgentSecretsReferencesOnlyV0,
		ExternalAgentSecurityInvalidaV0,
	)
	v.requireConst(
		field+".home_paths_policy",
		security.HomePathsPolicy,
		ExternalAgentHomeOpaqueRefsOnlyV0,
		ExternalAgentSecurityInvalidaV0,
	)
	v.requireConst(field+".network_policy", security.NetworkPolicy, ExternalAgentNetworkClosedV0, ExternalAgentSecurityInvalidaV0)
	v.requireConst(
		field+".transcripts_policy",
		security.TranscriptsPolicy,
		ExternalAgentTranscriptsForbiddenV0,
		ExternalAgentSecurityInvalidaV0,
	)
}

func (v *externalAgentConnectorValidatorV0) requireRuntimeKind(field, value string) {
	if !isOneOf(value, "cli", "api", "local_runtime", "remote_runtime", "other") {
		v.add(ExternalAgentProfileInvalidoV0, field)
	}
}

func (v *externalAgentConnectorValidatorV0) requireConst(
	field string,
	got string,
	want string,
	code ExternalAgentConnectorErrorCodeV0,
) {
	if got != want {
		v.add(code, field)
	}
}

func (v *externalAgentConnectorValidatorV0) requireOpaque(
	field string,
	value string,
	missingCode ExternalAgentConnectorErrorCodeV0,
) {
	if value == "" {
		v.add(missingCode, field)
		return
	}
	switch {
	case externalAgentUnsafeValueV0(value):
		v.add(ExternalAgentDetalleProhibidoV0, field)
	case !opaqueRefPatternV0.MatchString(value):
		v.add(ExternalAgentReferenciaNoOpacaV0, field)
	}
}

func (v *externalAgentConnectorValidatorV0) optionalOpaque(field, value string) {
	if value == "" {
		return
	}
	v.requireOpaque(field, value, ExternalAgentReferenciaNoOpacaV0)
}

func (v *externalAgentConnectorValidatorV0) add(code ExternalAgentConnectorErrorCodeV0, field string) {
	v.errors = append(v.errors, ExternalAgentConnectorErrorV0{
		Code:          code,
		MessageKey:    "orquesta.runtime.external_agent_connector." + string(code),
		Field:         field,
		Retryable:     false,
		CorrelationID: v.correlationID,
	})
}

func validateExternalAgentPacketForRequestV0(
	request RuntimeLaunchRequestV0,
	packet AgentStartPacketV0,
) []ExternalAgentConnectorErrorV0 {
	v := externalAgentConnectorValidatorV0{correlationID: request.CorrelationID}
	if !packet.Valid() || externalAgentPacketHasForbiddenDetailV0(packet) {
		v.add(ExternalAgentPacketInvalidoV0, "agent_packet")
		return v.errors
	}
	if packet.RequestID != request.RequestID ||
		packet.CorrelationID != request.CorrelationID ||
		packet.Locale != request.Locale ||
		request.ContextBundle == nil ||
		packet.WorkOrderRef != request.ContextBundle.WorkOrderRef ||
		packet.TargetModule != request.ContextBundle.TargetModule {
		v.add(ExternalAgentPacketInvalidoV0, "agent_packet")
	}
	return v.errors
}

func externalAgentErrorsFromRuntimeLaunchV0(
	issues []RuntimeLaunchErrorV0,
) []ExternalAgentConnectorErrorV0 {
	if len(issues) == 0 {
		return nil
	}
	errors := make([]ExternalAgentConnectorErrorV0, 0, len(issues))
	for _, issue := range issues {
		errors = append(errors, ExternalAgentConnectorErrorV0{
			Code:          ExternalAgentRequestInvalidaV0,
			MessageKey:    "orquesta.runtime.external_agent_connector." + string(ExternalAgentRequestInvalidaV0),
			Field:         issue.Field,
			Retryable:     false,
			CorrelationID: issue.CorrelationID,
			Evidence:      []string{string(issue.Code)},
		})
	}
	return errors
}

func externalAgentPacketHasForbiddenDetailV0(packet AgentStartPacketV0) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	if agentStartPacketHasForbiddenOperationalDetailV0(packet) {
		return true
	}
	raw, err := json.Marshal(packet)
	if err != nil {
		return true
	}
	return externalAgentUnsafeTextV0(string(raw))
}

func externalAgentUnsafeValueV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	trimmed := strings.TrimSpace(value)
	low := strings.ToLower(trimmed)
	return looksLikeSecret(trimmed) ||
		looksLikeConcreteProviderValueV0(trimmed) ||
		looksLikeConcreteModelValueV0(trimmed) ||
		processRuntimeContainsForbiddenMarkerV0(trimmed) ||
		externalAgentLooksLikeShellV0(low) ||
		strings.Contains(low, "provider") ||
		strings.Contains(low, "proveedor") ||
		strings.Contains(low, "model") ||
		strings.Contains(low, "modelo") ||
		strings.Contains(low, "home-ref") ||
		strings.Contains(low, "credential-ref") ||
		externalAgentUnsafeTextV0(trimmed)
}

func externalAgentUnsafeTextV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "transcript") ||
		strings.Contains(low, "prompt=") ||
		strings.Contains(low, "completion=") ||
		strings.Contains(low, "://") ||
		strings.HasPrefix(low, "/") ||
		strings.HasPrefix(low, `\`) ||
		strings.HasPrefix(low, "~")
}

func externalAgentLooksLikeShellV0(value string) bool {
	switch strings.TrimSpace(value) {
	case "bash", "dash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return strings.Contains(value, "shell")
	}
}
