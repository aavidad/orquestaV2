package main

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	mcpDuplicateResourceV0 = "mcp_duplicate_resource"
	mcpDuplicateToolV0     = "mcp_duplicate_tool"
)

type mcpRealRegistrationCollisionErrorV0 struct {
	code string
	name string
	uri  string
}

func (err mcpRealRegistrationCollisionErrorV0) Error() string {
	parts := []string{err.code}
	if name := publicMCPRegistrationRefV0(err.name, false); name != "" {
		parts = append(parts, "name="+name)
	}
	if uri := publicMCPRegistrationRefV0(err.uri, true); uri != "" {
		parts = append(parts, "uri="+uri)
	}
	return strings.Join(parts, " ")
}

func (registry *mcpRealTransportRegistryV0) registerResourceV0(
	resource orquestamcp.MCPTransportResourceEnvelopeV0,
) error {
	if _, ok := registry.resourcesByName[resource.Name]; ok {
		return mcpRealRegistrationCollisionErrorV0{
			code: mcpDuplicateResourceV0,
			name: resource.Name,
			uri:  resource.URI,
		}
	}
	if _, ok := registry.resourcesByURI[resource.URI]; ok {
		return mcpRealRegistrationCollisionErrorV0{
			code: mcpDuplicateResourceV0,
			name: resource.Name,
			uri:  resource.URI,
		}
	}
	registry.resourcesByName[resource.Name] = resource
	registry.resourcesByURI[resource.URI] = resource
	return nil
}

func (registry *mcpRealTransportRegistryV0) registerToolV0(
	tool orquestamcp.MCPTransportToolEnvelopeV0,
) error {
	if _, ok := registry.tools[tool.Name]; ok {
		return mcpRealRegistrationCollisionErrorV0{
			code: mcpDuplicateToolV0,
			name: tool.Name,
			uri:  tool.ResourceURI,
		}
	}
	registry.tools[tool.Name] = tool
	return nil
}

func publicMCPRegistrationRefV0(value string, allowOrquestaURI bool) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 160 {
		return ""
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{
		"/home/", "\\", "://127.", "://localhost", "@", "?", "=", "token",
		"secret", "password", "oauth", "bearer", "sk-",
	} {
		if strings.Contains(lower, forbidden) {
			return "redacted"
		}
	}
	if strings.ContainsAny(value, "\r\n\t ") {
		return "redacted"
	}
	if strings.Contains(value, "://") {
		if allowOrquestaURI && strings.HasPrefix(value, "orquesta://") {
			return value
		}
		return "redacted"
	}
	if strings.ContainsAny(value, `/`) {
		return "redacted"
	}
	return value
}
