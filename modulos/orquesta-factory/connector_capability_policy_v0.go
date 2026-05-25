package orquestafactory

import (
	"fmt"
	"strings"
	"unicode"
)

type ConnectorCapabilityPolicyPortV0 interface {
	ValidateConnectorCapabilityV0(input ConnectorCapabilityPolicyInputV0) []ValidationIssue
}

type ConnectorCapabilityPolicyInputV0 struct {
	Index         int
	Connector     ConnectorRequestV0
	FieldBasePath string
}

type DefaultConnectorCapabilityPolicyV0 struct{}

func validateConnectorNamesV0(req AppSpecRequestV0) []ValidationIssue {
	policy := DefaultConnectorCapabilityPolicyV0{}
	var issues []ValidationIssue
	for index, connector := range req.Integraciones {
		issues = append(issues, policy.ValidateConnectorCapabilityV0(ConnectorCapabilityPolicyInputV0{
			Index:         index,
			Connector:     connector,
			FieldBasePath: fmt.Sprintf("integraciones.%d", index),
		})...)
	}
	return issues
}

func (DefaultConnectorCapabilityPolicyV0) ValidateConnectorCapabilityV0(input ConnectorCapabilityPolicyInputV0) []ValidationIssue {
	connector := input.Connector
	name := strings.TrimSpace(connector.Nombre)
	if name == "" {
		return nil
	}
	if connectorUsesCredentialOrDSNV0(connector) {
		return []ValidationIssue{connectorCapabilityIssueV0(input.FieldBasePath, "la integracion no puede incluir credenciales, DSN ni cadenas de conexion")}
	}
	if connectorNamesConcreteProviderV0(name) || connectorForcesProviderBackendV0(connector) {
		return []ValidationIssue{connectorCapabilityIssueV0(input.FieldBasePath, "la integracion debe declarar una capacidad, no imponer proveedor o backend concreto")}
	}
	return nil
}

func connectorCapabilityIssueV0(basePath, message string) ValidationIssue {
	return issue(ErrConectorRequeridoNoDisponible, strings.TrimSpace(basePath)+".nombre", message)
}

func connectorNamesConcreteProviderV0(name string) bool {
	tokens := connectorPolicyTokensV0(name)
	return len(tokens) == 1 && concreteConnectorProviderTokensV0()[tokens[0]]
}

func connectorForcesProviderBackendV0(connector ConnectorRequestV0) bool {
	joined := strings.Join(append([]string{connector.Nombre, connector.Tipo, connector.Proposito}, connector.Restricciones...), " ")
	tokens := connectorPolicyTokensV0(joined)
	hasProvider := false
	hasBackendSignal := false
	for _, token := range tokens {
		if concreteConnectorProviderTokensV0()[token] {
			hasProvider = true
		}
		if providerBackendSignalTokensV0()[token] {
			hasBackendSignal = true
		}
	}
	return hasProvider && hasBackendSignal
}

func connectorUsesCredentialOrDSNV0(connector ConnectorRequestV0) bool {
	values := append([]string{connector.Nombre, connector.Tipo, connector.Proposito}, connector.Restricciones...)
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "://") || strings.HasPrefix(normalized, "jdbc:") {
			return true
		}
		if connectorContainsCredentialPhraseV0(normalized) {
			return true
		}
		for _, token := range connectorPolicyTokensV0(normalized) {
			if credentialOrDSNTokensV0()[token] {
				return true
			}
		}
	}
	return false
}

func connectorPolicyTokensV0(value string) []string {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return out
}

func connectorContainsCredentialPhraseV0(value string) bool {
	for _, phrase := range []string{
		"api_key",
		"api-key",
		"access_key",
		"access-key",
		"connection_string",
		"connection-string",
		"password=",
		"secret=",
		"token=",
	} {
		if strings.Contains(value, phrase) {
			return true
		}
	}
	return false
}

func concreteConnectorProviderTokensV0() map[string]bool {
	return map[string]bool{
		"anthropic":  true,
		"aws":        true,
		"azure":      true,
		"claude":     true,
		"cloud":      true,
		"codex":      true,
		"gcp":        true,
		"gemini":     true,
		"mongodb":    true,
		"mysql":      true,
		"openai":     true,
		"postgres":   true,
		"postgresql": true,
		"redis":      true,
		"sqlite":     true,
	}
}

func providerBackendSignalTokensV0() map[string]bool {
	return map[string]bool{
		"backend":      true,
		"cache":        true,
		"cola":         true,
		"database":     true,
		"db":           true,
		"driver":       true,
		"filesystem":   true,
		"fs":           true,
		"persistence":  true,
		"persistencia": true,
		"proveedor":    true,
		"provider":     true,
		"queue":        true,
		"runtime":      true,
		"sdk":          true,
		"storage":      true,
	}
}

func credentialOrDSNTokensV0() map[string]bool {
	return map[string]bool{
		"apikey":           true,
		"clave":            true,
		"connectionstring": true,
		"credential":       true,
		"credentials":      true,
		"credencial":       true,
		"credenciales":     true,
		"dsn":              true,
		"password":         true,
		"secret":           true,
		"secreto":          true,
		"token":            true,
	}
}
