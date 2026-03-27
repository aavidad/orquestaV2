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
		{Clave: "pool_default_budget_source", Valor: "manual"},
		{Clave: "connector_circuit_breaker_threshold", Valor: "3"},
		{Clave: "connector_circuit_breaker_cooldown_seconds", Valor: "300"},
		{Clave: "runtime_remote_sync_failure_threshold", Valor: "3"},
		{Clave: "runtime_transcript_ingest_max_bytes", Valor: "65536"},
		{Clave: "runtime_transcript_auto_guidance_enabled", Valor: "true"},
		{Clave: "autonomia_supervision_interval_seconds", Valor: "300"},
		{Clave: "autonomia_review_interval_seconds", Valor: "300"},
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
