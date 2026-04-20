package db

import "testing"

func TestDefaultConfigSeedRowsReflejanDefaults(t *testing.T) {
	t.Parallel()

	entries := defaultConfigEntries()
	rows := defaultConfigSeedRows()
	if len(entries) == 0 {
		t.Fatalf("defaultConfigEntries vacio")
	}
	if len(rows) != len(entries) {
		t.Fatalf("rows=%d entries=%d", len(rows), len(entries))
	}
	for i, item := range entries {
		if rows[i][0] != item.Clave || rows[i][1] != item.Valor {
			t.Fatalf("fila %d inesperada: %+v vs %+v", i, rows[i], item)
		}
	}
}

func TestConfigInt64FallbackUsaDefaultsDeclarados(t *testing.T) {
	t.Parallel()

	if got := configInt64Fallback("runtime_orders_retention_minutes", -1); got != 30 {
		t.Fatalf("runtime_orders_retention_minutes default inesperado: %d", got)
	}
	if got := configInt64Fallback("runtime_handles_retention_minutes", -1); got != 10 {
		t.Fatalf("runtime_handles_retention_minutes default inesperado: %d", got)
	}
	if got := configInt64Fallback("clave_inexistente_para_test", 77); got != 77 {
		t.Fatalf("fallback duro inesperado: %d", got)
	}
}
