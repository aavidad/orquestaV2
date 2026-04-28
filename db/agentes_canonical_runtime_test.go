package db

import "testing"

func TestRuntimeOrderCanonicalizaAgenteEnWriteYFiltro(t *testing.T) {
	prepararDBTemporal(t)
	for _, nombre := range []string{"Codex11", "codex11"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-runtime-order",
		Nombre:  "Demo Runtime Order",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "codex11",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		Estado:      "pendiente",
		PayloadJSON: "{}",
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	agente := "codex11"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].Agente != "Codex11" {
		t.Fatalf("orders inesperadas: %+v", orders)
	}
}

func TestRuntimeMailboxCanonicalizaAgentesEnWriteYFiltro(t *testing.T) {
	prepararDBTemporal(t)
	for _, nombre := range []string{"Codex12", "codex12", "Codex13", "codex13"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-runtime-mailbox",
		Nombre:  "Demo Runtime Mailbox",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "codex12",
		ToAgente:    "codex13",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: "{}",
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("EnviarRuntimeMailbox: %v", err)
	}
	to := "codex13"
	items, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &to, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("ListarRuntimeMailbox: %v", err)
	}
	if len(items) != 1 || items[0].FromAgente != "Codex12" || items[0].ToAgente != "Codex13" {
		t.Fatalf("mailbox inesperado: %+v", items)
	}
}

func TestObtenerProyectoActivoAgenteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)
	for _, nombre := range []string{"Codex14", "codex14"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-proyecto-activo",
		Nombre:  "Demo Proyecto Activo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex14", proyectoID, "test proyecto activo"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	got, err := ObtenerProyectoActivoAgente("codex14")
	if err != nil {
		t.Fatalf("ObtenerProyectoActivoAgente: %v", err)
	}
	if got != proyectoID {
		t.Fatalf("proyecto activo inesperado: got=%d want=%d", got, proyectoID)
	}
}
