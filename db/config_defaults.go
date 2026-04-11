package db

type configDefaultEntry struct {
	Clave string
	Valor string
}

func defaultConfigEntries() []configDefaultEntry {
	return []configDefaultEntry{
		{Clave: "distribuidor", Valor: "claude"},
		{Clave: "version", Valor: "1.0.0"},
		{Clave: "pool_handoff_threshold_seconds", Valor: "1800"},
		{Clave: "pool_handoff_threshold_ratio", Valor: "0.10"},
		{Clave: "pool_budget_snapshot_max_age_seconds", Valor: "300"},
		{Clave: "pool_budget_snapshot_observed_max_age_seconds", Valor: "3600"},
		{Clave: "pool_default_budget_source", Valor: "manual"},
		{Clave: "connector_circuit_breaker_threshold", Valor: "3"},
		{Clave: "connector_circuit_breaker_cooldown_seconds", Valor: "300"},
		{Clave: "runtime_remote_sync_failure_threshold", Valor: "3"},
		{Clave: "runtime_supervision_interval_seconds", Valor: "60"},
		{Clave: "runtime_supervision_batch_size", Valor: "10"},
		{Clave: "runtime_budget_background_observation_interval_seconds", Valor: "120"},
		{Clave: "runtime_shared_account_active_ceiling", Valor: "1"},
		{Clave: "runtime_shared_account_order_hold_seconds", Valor: "300"},
		{Clave: "autonomia_active_sessions_interval_seconds", Valor: "60"},
		{Clave: "runtime_order_lease_seconds", Valor: "120"},
		{Clave: "runtime_send_instruction_lease_seconds", Valor: "35"},
		{Clave: "runtime_send_instruction_retry_seconds", Valor: "15"},
		{Clave: "runtime_send_instruction_receipt_timeout_seconds", Valor: "120"},
		{Clave: "runtime_transcript_ingest_max_bytes", Valor: "65536"},
		{Clave: "runtime_transcript_auto_guidance_enabled", Valor: "true"},
		{Clave: "runtime_instances_retention_minutes", Valor: "30"},
		{Clave: "runtime_handles_retention_minutes", Valor: "10"},
		{Clave: "runtime_orders_retention_minutes", Valor: "30"},
		{Clave: "runtime_instances_retention_hours", Valor: "72"},
		{Clave: "runtime_handles_retention_hours", Valor: "24"},
		{Clave: "runtime_orders_retention_hours", Valor: "72"},
		{Clave: "session_operational_stale_seconds", Valor: "180"},
		{Clave: "openclaw_gateway_url", Valor: ""},
		{Clave: "openclaw_gateway_token", Valor: ""},
		{Clave: "openclaw_gateway_operator", Valor: "alberto"},
		{Clave: "autonomia_nudge_cooldown_seconds", Valor: "60"},
		{Clave: "autonomia_supervision_interval_seconds", Valor: "300"},
		{Clave: "autonomia_review_interval_seconds", Valor: "300"},
		{Clave: "server_autobootstrap_enabled", Valor: "false"},
		{Clave: "server_autobootstrap_project_slug", Valor: "orquestador"},
		{Clave: "server_autobootstrap_project_name", Valor: "Orquestador"},
		{Clave: "server_autobootstrap_supervisor_agent", Valor: "Codex1"},
		{Clave: "server_autobootstrap_worker_agents", Valor: "Codex2,Codex3,Codex4,Codex5"},
		{Clave: "server_autobootstrap_objective_general", Valor: "Terminar la app al completo, revisando el codigo real, reparando fallos de raiz y validando con pruebas reales."},
		{Clave: "server_autobootstrap_definition_of_done_json", Valor: "{\"estado\":\"app_completa\",\"criterios\":[\"codigo_real_y_funcional\",\"sin_humo\",\"pruebas_reales_en_verde\",\"frentes_cerrados\"]}"},
		{Clave: "model_policy_default_profile", Valor: "implementacion"},
		{Clave: "model_policy_default_reasoning", Valor: "high"},
	}
}

func defaultConfigSeedRows() [][]string {
	rows := make([][]string, 0, len(defaultConfigEntries()))
	for _, item := range defaultConfigEntries() {
		rows = append(rows, []string{item.Clave, item.Valor})
	}
	return rows
}
