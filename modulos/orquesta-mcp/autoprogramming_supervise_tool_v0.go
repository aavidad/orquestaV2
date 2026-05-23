package orquestamcp

const (
	MCPAutoprogrammingSuperviseToolNameV0    = "orquesta.autoprogramming.supervise.v0"
	MCPAutoprogrammingSuperviseToolVersionV0 = "v0"
	MCPAutoprogrammingSuperviseResourceURIV0 = "orquesta://contracts/autoprogramming-supervise/v0"
)

type MCPAutoprogrammingSuperviseToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

func MCPAutoprogrammingSuperviseDescriptorV0() MCPAutoprogrammingSuperviseToolDescriptorV0 {
	return MCPAutoprogrammingSuperviseToolDescriptorV0{
		Name:        MCPAutoprogrammingSuperviseToolNameV0,
		Version:     MCPAutoprogrammingSuperviseToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,queue_ref?,resident_mode?,max_ticks?,limits?}",
		Output:      "ok:{run_ref,stop_reason,ticks,last,history?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPAutoprogrammingSuperviseResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"supervision puntual via runs.supervisor inyectado",
			"run_ref acota una run de autoprogramacion si se proporciona",
			"sin run_ref delega la decision de cola al executor inyectado",
			"resident_mode permite al executor seguir backlog durable hasta cierre o bloqueo",
			"no conoce Codex runtime DB filesystem ni proveedor concreto",
		},
	}
}
