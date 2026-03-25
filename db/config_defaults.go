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
