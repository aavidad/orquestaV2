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
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,operational_director_plan_ref?,queue_ref?,continue_message?,occurred_at?,idempotency_key?,max_ticks?,max_runs_per_tick?,max_executions?,allow_repeated_runs?,max_bursts?,max_steps_per_burst?,max_dispatches_per_wait?,max_commands?,max_outbox_per_cycle?,max_decision_cycles?,max_external_waits?,resident_mode?,operator_advice?}",
		Output:      "ok:{run_ref,stop_reason,ticks,last,history?,evidence_refs?,operation_ref?,operator_advice?,diagnostics?}|error:{errores_publicos,operation_ref?,operator_advice?,diagnostics?}",
		ResourceURI: MCPAutoprogrammingSuperviseResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"compatibilidad legacy/resident para runs del loop historico",
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
