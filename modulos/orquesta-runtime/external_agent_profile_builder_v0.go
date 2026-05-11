package orquestaruntime

type ExternalAgentConnectorProfileRefsV0 struct {
	ProfileRef    string
	ConnectorRef  string
	RuntimeKind   string
	CommandRef    string
	ExecutableRef string
	WorkingDirRef string
	ArgRefs       []string
	EnvRefs       []string
}

func BuildClosedExternalAgentConnectorProfileV0(
	refs ExternalAgentConnectorProfileRefsV0,
) ExternalAgentConnectorProfileV0 {
	return ExternalAgentConnectorProfileV0{
		SchemaVersion: ExternalAgentConnectorProfileSchemaVersionV0,
		ProfileRef:    refs.ProfileRef,
		ConnectorRef:  refs.ConnectorRef,
		RuntimeKind:   refs.RuntimeKind,
		LaunchMode:    RuntimeLaunchModeNewSessionV0,
		Command: ExternalAgentCommandV0{
			CommandRef:    refs.CommandRef,
			ExecutableRef: refs.ExecutableRef,
			ArgRefs:       append([]string(nil), refs.ArgRefs...),
			EnvRefs:       append([]string(nil), refs.EnvRefs...),
			WorkingDirRef: refs.WorkingDirRef,
		},
		Security: ClosedExternalAgentSecurityPolicyV0(),
	}
}

func ClosedExternalAgentSecurityPolicyV0() ExternalAgentSecurityPolicyV0 {
	return ExternalAgentSecurityPolicyV0{
		OptIn:                 true,
		ShellPolicy:           ExternalAgentShellForbiddenV0,
		PathInheritancePolicy: ExternalAgentPathInheritanceForbiddenV0,
		EnvironmentPolicy:     ExternalAgentEnvExplicitRefsOnlyV0,
		SecretsPolicy:         ExternalAgentSecretsReferencesOnlyV0,
		HomePathsPolicy:       ExternalAgentHomeOpaqueRefsOnlyV0,
		NetworkPolicy:         ExternalAgentNetworkClosedV0,
		TranscriptsPolicy:     ExternalAgentTranscriptsForbiddenV0,
	}
}
