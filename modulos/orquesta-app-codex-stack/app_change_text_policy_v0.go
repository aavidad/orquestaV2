package orquestaappcodexstack

import "strings"

func appChangeDirectorSafeTextV0(value string) string {
	replacer := strings.NewReplacer(
		"Codex", "agente externo",
		"codex", "agente externo",
		"Claude", "agente externo",
		"claude", "agente externo",
		"Ollama", "agente externo",
		"ollama", "agente externo",
		"vLLM", "agente externo",
		"vllm", "agente externo",
		"provider", "servicio externo",
		"Provider", "Servicio externo",
		"proveedor", "servicio externo",
		"Proveedor", "Servicio externo",
		"adapter", "conector",
		"Adapter", "Conector",
		"adaptador", "conector",
		"Adaptador", "Conector",
		"runtime", "ejecucion",
		"Runtime", "Ejecucion",
		"HOME", "directorio interno",
		"home", "directorio interno",
		"token", "dato sensible",
		"Token", "Dato sensible",
		"tokens", "datos sensibles",
		"Tokens", "Datos sensibles",
		"secret", "dato sensible",
		"Secret", "Dato sensible",
		"secreto", "dato sensible",
		"Secreto", "Dato sensible",
		"password", "dato sensible",
		"Password", "Dato sensible",
		"credential", "dato sensible",
		"Credential", "Dato sensible",
		"credencial", "dato sensible",
		"Credencial", "Dato sensible",
		"oauth", "identidad externa",
		"OAuth", "Identidad externa",
		"api_key", "clave externa",
		"API_KEY", "clave externa",
		"database", "almacen interno",
		"Database", "Almacen interno",
		"base de datos", "almacen interno",
		"Base de datos", "Almacen interno",
		"SQL", "consulta interna",
		"sql", "consulta interna",
		"dsn", "conexion interna",
		"DSN", "conexion interna",
	)
	return strings.TrimSpace(replacer.Replace(value))
}

func appChangeDirectorSafeRefV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(
		"codex", "agent",
		"claude", "agent",
		"ollama", "agent",
		"vllm", "agent",
		"provider", "service",
		"proveedor", "service",
		"adapter", "connector",
		"adaptador", "connector",
		"runtime", "execution",
		"home", "internal-dir",
		"token", "sensitive",
		"secret", "sensitive",
		"secreto", "sensitive",
		"password", "sensitive",
		"credential", "sensitive",
		"credencial", "sensitive",
		"oauth", "external-id",
		"api_key", "external-key",
		"database", "internal-store",
		"base-de-datos", "internal-store",
		"sql", "internal-query",
		"dsn", "internal-connection",
	)
	return strings.TrimSpace(replacer.Replace(value))
}
