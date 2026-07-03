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
		InputSchema: "envelope:{request_id?,correlation_id?,director_execution_mode?:goal_first|legacy_director_loop,run_ref?,queue_ref?,max_ticks?,resident_mode?,operator_advice?}",
		Output:      "ok:{run_ref,stop_reason,ticks,last,history?,evidence_refs?,idempotency_key?,operation_ref?,repair_run_refs?,operator_advice?,diagnostics?,next_actions?}|error:{errores_publicos,evidence_refs?,idempotency_key?,operation_ref?,repair_run_refs?,operator_advice?,diagnostics?,next_actions?}",
		ResourceURI: MCPAutoprogrammingSuperviseResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"compatibilidad legacy/resident para runs del loop historico",
			"legacy exige opt-in de composicion y director_execution_mode=legacy_director_loop",
			"supervision puntual via runs.supervisor inyectado",
			"run_ref acota una run de autoprogramacion si se proporciona",
			"sin run_ref delega la decision de cola al executor inyectado",
			"no supervisa runs goal-first; usar orquesta.autoprogramming.observe_goal.v0 para observar goals",
			"resident_mode permite al executor seguir backlog durable hasta cierre o bloqueo",
			"operator_advice se conserva como observacion no bloqueante",
			"no conoce Codex runtime DB filesystem ni proveedor concreto",
		},
	}
}
