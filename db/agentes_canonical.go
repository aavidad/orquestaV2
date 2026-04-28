package db

import (
	"database/sql"
	"strings"
)

// CanonicalizeAgentName returns the canonical persisted agent name when the
// agent already exists. If the agent is unknown, it returns the trimmed input
// unchanged so write paths preserve current behavior.
func CanonicalizeAgentName(nombre string) (string, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "", nil
	}
	preferred := canonicalPreferredAgentName(nombre)
	if preferred != nombre && strings.HasPrefix(strings.ToLower(nombre), "codex") {
		return preferred, nil
	}
	nombreCanonico, _, _, err := resolverAgentePorNombreCI(nombre)
	if err == nil {
		return nombreCanonico, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	if preferred != nombre {
		nombreCanonico, _, _, err := resolverAgentePorNombreCI(preferred)
		if err == nil {
			return nombreCanonico, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	return preferred, nil
}

func canonicalPreferredAgentName(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	lower := strings.ToLower(nombre)
	if !strings.HasPrefix(lower, "codex") {
		return nombre
	}
	if len(nombre) <= len("codex") {
		return "Codex"
	}
	return "Codex" + strings.TrimSpace(nombre[len("codex"):])
}
