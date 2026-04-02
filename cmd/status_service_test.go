package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestAgenteCuentaComoConectadoRespetaEstadoCuotaVisible(t *testing.T) {
	if agenteCuentaComoConectado(nil) {
		t.Fatalf("nil no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex1", Activo: false, EstadoCuota: "activo"}) {
		t.Fatalf("un agente sin sesion activa no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex2", Activo: true, EstadoCuota: "agotado"}) {
		t.Fatalf("un agente agotado no deberia contar como conectado visible")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex3", Activo: true, EstadoCuota: "enfriamiento"}) {
		t.Fatalf("un agente en enfriamiento no deberia contar como conectado visible")
	}
	if !agenteCuentaComoConectado(&db.Agente{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}) {
		t.Fatalf("un agente activo con cuota activa deberia contar como conectado")
	}
	if !agenteCuentaComoConectado(&db.Agente{
		Nombre:               "antigravity",
		Activo:               true,
		EstadoCuota:          "agotado",
		PresupuestoSemanalPct: intPtr(69),
		CuotaRestantePct:     intPtr(69),
		PresupuestoVentana:   "weekly",
	}) {
		t.Fatalf("una cuenta solo semanal con saldo positivo no deberia caer por una cuota legacy")
	}
}

func TestStatusServiceCacheaSnapshotCorto(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = 2 * time.Second

	calls := 0
	statusFreshFetcher = func() (apiStatusResponse, error) {
		calls++
		return apiStatusResponse{Generado: current.Format(time.RFC3339)}, nil
	}

	service := dbStatusService{}
	first, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	second, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if calls != 1 {
		t.Fatalf("la cache deberia evitar una segunda carga, calls=%d", calls)
	}
	if first.Generado != second.Generado {
		t.Fatalf("snapshot cacheado inesperado: %q vs %q", first.Generado, second.Generado)
	}

	current = current.Add(3 * time.Second)
	third, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("third fetch: %v", err)
	}
	if calls != 2 {
		t.Fatalf("tras expirar la cache deberia recargar, calls=%d", calls)
	}
	if third.Generado == second.Generado {
		t.Fatalf("deberia haberse regenerado el snapshot tras expirar la cache")
	}
}
