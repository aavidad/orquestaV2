package orquestamcp

type MCPNuevaAppPromptV0 struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ResourceURI       string   `json:"resource_uri"`
	Arguments         []string `json:"arguments"`
	Instructions      []string `json:"instructions"`
	MinimumQuestions  []string `json:"minimum_questions"`
	AssumptionPolicy  string   `json:"assumption_policy"`
	PublicErrorCodes  []string `json:"public_error_codes"`
	ForbiddenSurfaces []string `json:"forbidden_surfaces"`
}

func MCPNuevaAppPromptV0Value() MCPNuevaAppPromptV0 {
	return MCPNuevaAppPromptV0{
		Name:        MCPNuevaAppPromptNameV0,
		Version:     MCPNuevaAppToolVersionV0,
		ResourceURI: MCPNuevaAppResourceURIV0,
		Arguments: []string{
			"idioma",
			"contexto_usuario",
		},
		Instructions: []string{
			"Extrae nombre, objetivo, tipo de app, usuarios y plataformas desde el contexto del usuario.",
			"Antes de invocar el tool, pide solo la informacion obligatoria que no pueda inferirse con seguridad.",
			"Construye app_spec_request y deja la validacion de negocio al contrato SolicitarNuevaApp v0.",
		},
		MinimumQuestions: []string{
			"Que problema debe resolver la app y para quienes.",
			"Que tipo de aplicacion espera el usuario si no queda claro.",
			"Que integraciones o datos externos son obligatorios.",
		},
		AssumptionPolicy: "Los supuestos deben ser pocos, visibles y compatibles con SolicitarNuevaApp v0; si afectan alcance, datos o despliegue, pregunta antes.",
		PublicErrorCodes: []string{
			"idioma_invalido",
			"app_spec_invalida",
		},
		ForbiddenSurfaces: []string{
			"cli",
			"db",
			"filesystem",
			"runtime",
		},
	}
}
