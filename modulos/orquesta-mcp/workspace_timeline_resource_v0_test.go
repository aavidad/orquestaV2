package orquestamcp

import "testing"

func TestMCPWorkspaceTimelineResourceV0DeclaraContratoYGuardas(t *testing.T) {
	descriptor := MCPWorkspaceTimelineDescriptorV0()
	if descriptor.Name != MCPWorkspaceTimelineResourceNameV0 ||
		descriptor.URI != MCPWorkspaceTimelineResourceURIV0 ||
		descriptor.ContentType != MCPWorkspaceTimelineContentTypeV0 {
		t.Fatalf("descriptor inesperado: %+v", descriptor)
	}
	resource := NewMCPWorkspaceTimelineResourceV0()
	if resource.RecommendedEndpoint != MCPWorkspaceTimelineEndpointV0 ||
		len(resource.AllowedSources) != 7 {
		t.Fatalf("resource incompleto: %+v", resource)
	}
	if !containsStringMCPWorkspaceTimelineTestV0(resource.Guardrails, "sin_shell_git_local_runtime_filesystem_directo") {
		t.Fatalf("guardrail shell/runtime ausente: %+v", resource.Guardrails)
	}
}

func TestMCPTransportResourcesV0IncluyeWorkspaceTimeline(t *testing.T) {
	found := false
	for _, resource := range MCPTransportResourcesV0() {
		if resource.Name == MCPWorkspaceTimelineResourceNameV0 {
			found = true
			if resource.URI != MCPWorkspaceTimelineResourceURIV0 {
				t.Fatalf("uri=%q", resource.URI)
			}
		}
	}
	if !found {
		t.Fatalf("workspace timeline no registrado")
	}
}

func containsStringMCPWorkspaceTimelineTestV0(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
