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
