package orquestacoreworkflow

import "testing"

func TestSupportedOrchestrationCommandTypesV0CompletoYClonado(t *testing.T) {
	got := SupportedOrchestrationCommandTypesV0()
	if len(got) != 34 {
		t.Fatalf("command catalog size=%d, want 34: %v", len(got), got)
	}
	for _, commandType := range got {
		if !isSupportedCommandTypeV0(commandType) {
			t.Fatalf("catalog command no soportado por validator: %q", commandType)
		}
	}

	got[0] = "mutado"
	if SupportedOrchestrationCommandTypesV0()[0] == "mutado" {
		t.Fatalf("catalogo de comandos debe devolver copia defensiva")
	}
}

func TestSupportedOrchestrationEventTypesV0CompletoYClonado(t *testing.T) {
	got := SupportedOrchestrationEventTypesV0()
	if len(got) != 34 {
		t.Fatalf("event catalog size=%d, want 34: %v", len(got), got)
	}
	for _, eventType := range got {
		if !isSupportedEventTypeV0(eventType) {
			t.Fatalf("catalog event no soportado por validator: %q", eventType)
		}
	}

	got[0] = "mutado"
	if SupportedOrchestrationEventTypesV0()[0] == "mutado" {
		t.Fatalf("catalogo de eventos debe devolver copia defensiva")
	}
}

func TestCommandRouterV0SincronizadoConCatalogo(t *testing.T) {
	assertRouterCoversCatalogV0(t, "command", orchestrationCommandTypeCatalogV0, commandHandlersV0)
}

func TestEventRouterV0SincronizadoConCatalogo(t *testing.T) {
	assertRouterCoversCatalogV0(t, "event", orchestrationEventTypeCatalogV0, eventAppliersV0)
}

func assertRouterCoversCatalogV0[T any](t *testing.T, name string, catalog []string, routes map[string]T) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range catalog {
		if seen[item] {
			t.Fatalf("%s catalog duplicate: %q", name, item)
		}
		seen[item] = true
		if _, ok := routes[item]; !ok {
			t.Fatalf("%s route missing catalog item %q", name, item)
		}
	}
	for item := range routes {
		if !seen[item] {
			t.Fatalf("%s route outside catalog: %q", name, item)
		}
	}
}
