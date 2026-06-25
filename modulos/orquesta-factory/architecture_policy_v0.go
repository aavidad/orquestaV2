package orquestafactory

import "strings"

const DefaultArchitecturePatternV0 = "hexagonal"

var supportedArchitecturePatternsV0 = []string{
	"hexagonal",
	"clean_architecture",
	"onion",
	"modular_monolith",
	"layered",
	"event_driven",
	"microservices",
	"serverless",
	"plugin_based",
	"data_pipeline",
}

func SupportedArchitecturePatternsV0() []string {
	return append([]string{}, supportedArchitecturePatternsV0...)
}

func NormalizeArchitecturePatternV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "":
		return ""
	case "clean", "clean_architecture", "arquitectura_limpia":
		return "clean_architecture"
	case "onion", "arquitectura_onion":
		return "onion"
	case "modular", "modular_monolith", "monolito_modular":
		return "modular_monolith"
	case "layers", "layered", "capas", "arquitectura_en_capas":
		return "layered"
	case "events", "event_driven", "eventos", "orientada_a_eventos":
		return "event_driven"
	case "microservice", "microservices", "micro_servicios", "microservicios":
		return "microservices"
	case "serverless":
		return "serverless"
	case "plugin", "plugins", "plugin_based", "extensible_por_plugins":
		return "plugin_based"
	case "pipeline", "data_pipeline", "datos_pipeline":
		return "data_pipeline"
	default:
		return normalized
	}
}

func ArchitecturePatternSupportedV0(value string) bool {
	normalized := NormalizeArchitecturePatternV0(value)
	for _, candidate := range supportedArchitecturePatternsV0 {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func architecturePatternOrDefaultV0(value string) string {
	if normalized := NormalizeArchitecturePatternV0(value); normalized != "" {
		return normalized
	}
	return DefaultArchitecturePatternV0
}
