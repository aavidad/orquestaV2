package observabilidadruntime

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Adapter interface {
	Provider() string
	Normalize(payload json.RawMessage) (*Sample, error)
}

var adapters = map[string]Adapter{
	"generic_process": genericProcessAdapter{},
	"codex_cli":       codexCLIAdapter{},
}

func Providers() []string {
	return []string{"codex_cli", "generic_process"}
}

func Normalize(provider string, payload json.RawMessage) (*Sample, error) {
	adapter, ok := adapters[strings.TrimSpace(provider)]
	if !ok {
		return nil, fmt.Errorf("proveedor pasivo no soportado: %s", provider)
	}
	return adapter.Normalize(payload)
}
