package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGuardarRuntimeHandleYCerrarPrevioActivo(t *testing.T) {
	withTempDBPools(t, func() {
		firstID, err := GuardarRuntimeHandle(&RuntimeHandle{
			Agente:      "codex1",
			Transporte:  "local",
			HandleKind:  "process",
			HandleRef:   "1234",
			Estado:      "activo",
			MetadataJSON: `{"pid":1234}`,
		})
		if err != nil {
			t.Fatalf("GuardarRuntimeHandle first: %v", err)
		}
		secondID, err := GuardarRuntimeHandle(&RuntimeHandle{
			Agente:      "codex1",
			Transporte:  "local",
			HandleKind:  "process",
			HandleRef:   "4321",
			Estado:      "activo",
			MetadataJSON: `{"pid":4321}`,
		})
		if err != nil {
			t.Fatalf("GuardarRuntimeHandle second: %v", err)
		}
		if firstID == secondID {
			t.Fatalf("ids repetidos: %d", firstID)
		}
		handle, err := GetRuntimeHandleActivo("codex1")
		if err != nil {
			t.Fatalf("GetRuntimeHandleActivo: %v", err)
		}
		if handle.HandleRef != "4321" {
			t.Fatalf("handle activo inesperado: %+v", handle)
		}
	})
}

func TestEjecutarRuntimeOrderEnviarInstruccion(t *testing.T) {
	withTempDBPools(t, func() {
		path := filepath.Join(t.TempDir(), "agent-inbox.txt")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("write temp inbox: %v", err)
		}
		if _, err := GuardarRuntimeHandle(&RuntimeHandle{
			Agente:      "codex1",
			Transporte:  "local",
			HandleKind:  "pty",
			HandleRef:   path,
			Estado:      "activo",
			MetadataJSON: `{}`,
		}); err != nil {
			t.Fatalf("GuardarRuntimeHandle: %v", err)
		}
		payload, _ := json.Marshal(map[string]any{"mensaje": "haz checkpoint"})
		if _, err := CrearRuntimeOrder(&RuntimeOrder{
			Agente:      "codex1",
			Tipo:        "enviar_instruccion",
			PayloadJSON: string(payload),
		}); err != nil {
			t.Fatalf("CrearRuntimeOrder: %v", err)
		}

		order, err := EjecutarSiguienteRuntimeOrder("codex1")
		if err != nil {
			t.Fatalf("EjecutarSiguienteRuntimeOrder: %v", err)
		}
		if order == nil || order.Estado != "completada" {
			t.Fatalf("order inesperada: %+v", order)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "haz checkpoint\n" {
			t.Fatalf("contenido inesperado: %q", string(data))
		}
	})
}

func TestEjecutarRuntimeOrderSinHandleMarcaFallida(t *testing.T) {
	withTempDBPools(t, func() {
		payload, _ := json.Marshal(map[string]any{"mensaje": "hola"})
		id, err := CrearRuntimeOrder(&RuntimeOrder{
			Agente:      "codex1",
			Tipo:        "enviar_instruccion",
			PayloadJSON: string(payload),
		})
		if err != nil {
			t.Fatalf("CrearRuntimeOrder: %v", err)
		}
		if _, err := EjecutarSiguienteRuntimeOrder("codex1"); err == nil {
			t.Fatalf("se esperaba error sin handle")
		}
		orders, err := ListarRuntimeOrders("codex1", "fallida")
		if err != nil {
			t.Fatalf("ListarRuntimeOrders: %v", err)
		}
		if len(orders) != 1 || orders[0].ID != id {
			t.Fatalf("runtime order fallida inesperada: %+v", orders)
		}
	})
}
