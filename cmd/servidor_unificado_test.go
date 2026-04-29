/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestNormalizarAddrServidorLocal(t *testing.T) {
	casos := []struct {
		nombre string
		in     string
		want   string
	}{
		{nombre: "solo puerto", in: ":16543", want: "127.0.0.1:16543"},
		{nombre: "wildcard", in: "0.0.0.0:16543", want: "127.0.0.1:16543"},
		{nombre: "http wildcard", in: "http://0.0.0.0:16543", want: "127.0.0.1:16543"},
		{nombre: "host explicito", in: "127.0.0.1:16543", want: "127.0.0.1:16543"},
	}

	for _, tc := range casos {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			if got := normalizarAddrServidorLocal(tc.in); got != tc.want {
				t.Fatalf("normalizarAddrServidorLocal(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscucharServidorUnificadoReservaListenerAntesDelArranque(t *testing.T) {
	prev := listenServerTCP
	t.Cleanup(func() {
		listenServerTCP = prev
	})
	listenServerTCP = func(network, address string) (net.Listener, error) {
		return fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 16543}}, nil
	}

	listener, advertised, err := escucharServidorUnificado("127.0.0.1:0")
	if err != nil {
		t.Fatalf("escucharServidorUnificado: %v", err)
	}
	defer listener.Close()

	if advertised == "" {
		t.Fatalf("se esperaba addr anunciada")
	}
	if listener.Addr() == nil {
		t.Fatalf("listener sin addr")
	}
}

type fakeListener struct {
	addr net.Addr
}

func (f fakeListener) Accept() (net.Conn, error) { return nil, io.EOF }
func (f fakeListener) Close() error              { return nil }
func (f fakeListener) Addr() net.Addr            { return f.addr }

func TestRegistrarRutasServeMontaSuperficieOperativa(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registrarRutasServe(mux)

	for _, path := range []string{"/asignaciones", "/sesiones", "/proyectos", "/progreso", "/pools", "/modelo", "/memoria", "/deploy", "/config", "/conectores", "/diagnostico", "/auditoria", "/refineria", "/respaldo", "/gobernanza", "/git"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("ruta %s no montada", path)
		}
	}
}

func TestLoadUnifiedServerRuntimeOptionsDefaults(t *testing.T) {
	for _, key := range []string{
		"ORQUESTA_SERVER_CORE_ONLY",
		"ORQUESTA_SERVER_DISABLE_NONRESIDENT_WORKER",
		"ORQUESTA_SERVER_DISABLE_AUTOBOOTSTRAP",
		"ORQUESTA_SERVER_DISABLE_WARM_WORKER",
		"ORQUESTA_SERVER_DISABLE_COLD_WORKER",
		"ORQUESTA_SERVER_DISABLE_NOTIFICATION_RETRY",
	} {
		t.Setenv(key, "")
	}
	got := loadUnifiedServerRuntimeOptions()
	if got.CoreOnly {
		t.Fatalf("CoreOnly no deberia activarse por defecto")
	}
	if got.DisableAutobootstrap {
		t.Fatalf("DisableAutobootstrap no deberia activarse por defecto")
	}
	if got.DisableWarmWorker || got.DisableColdWorker || got.DisableNotificationRetry {
		t.Fatalf("los carriles no residentes no deberian desactivarse por defecto: %+v", got)
	}
}

func TestLoadUnifiedServerRuntimeOptionsCoreOnlyDesactivaWarmYAutobootstrap(t *testing.T) {
	t.Setenv("ORQUESTA_SERVER_CORE_ONLY", "1")
	got := loadUnifiedServerRuntimeOptions()
	if !got.CoreOnly {
		t.Fatalf("CoreOnly deberia activarse")
	}
	if !got.DisableAutobootstrap {
		t.Fatalf("DisableAutobootstrap deberia activarse con core-only")
	}
	if !got.DisableWarmWorker || !got.DisableColdWorker || !got.DisableNotificationRetry {
		t.Fatalf("core-only deberia apagar todos los carriles no residentes: %+v", got)
	}
}

func TestLoadUnifiedServerRuntimeOptionsAceptaFlagsSeparadas(t *testing.T) {
	t.Setenv("ORQUESTA_SERVER_DISABLE_NONRESIDENT_WORKER", "true")
	t.Setenv("ORQUESTA_SERVER_DISABLE_AUTOBOOTSTRAP", "yes")
	got := loadUnifiedServerRuntimeOptions()
	if !got.CoreOnly {
		t.Fatalf("CoreOnly deberia activarse con ORQUESTA_SERVER_DISABLE_NONRESIDENT_WORKER")
	}
	if !got.DisableAutobootstrap {
		t.Fatalf("DisableAutobootstrap deberia activarse con ORQUESTA_SERVER_DISABLE_AUTOBOOTSTRAP")
	}
	if !got.DisableWarmWorker || !got.DisableColdWorker || !got.DisableNotificationRetry {
		t.Fatalf("DisableNonResidentWorker deberia apagar todos los carriles no residentes: %+v", got)
	}
}

func TestLoadUnifiedServerRuntimeOptionsAceptaFlagsGranularesNoResidentes(t *testing.T) {
	t.Setenv("ORQUESTA_SERVER_DISABLE_WARM_WORKER", "1")
	t.Setenv("ORQUESTA_SERVER_DISABLE_NOTIFICATION_RETRY", "yes")
	got := loadUnifiedServerRuntimeOptions()
	if got.CoreOnly {
		t.Fatalf("flags granulares no deberian forzar core-only")
	}
	if !got.DisableWarmWorker {
		t.Fatalf("DisableWarmWorker deberia activarse")
	}
	if got.DisableColdWorker {
		t.Fatalf("DisableColdWorker no deberia activarse")
	}
	if !got.DisableNotificationRetry {
		t.Fatalf("DisableNotificationRetry deberia activarse")
	}
	if got.DisableAutobootstrap {
		t.Fatalf("DisableAutobootstrap no deberia activarse por flags granulares")
	}
}

func TestLoadUnifiedServerHTTPTimeoutsDefaults(t *testing.T) {
	got := loadUnifiedServerHTTPTimeouts()
	if got.ReadHeader != 5*time.Second {
		t.Fatalf("ReadHeader=%s want 5s", got.ReadHeader)
	}
	if got.Read != 15*time.Second {
		t.Fatalf("Read=%s want 15s", got.Read)
	}
	if got.Write != 30*time.Second {
		t.Fatalf("Write=%s want 30s", got.Write)
	}
	if got.Idle != 60*time.Second {
		t.Fatalf("Idle=%s want 60s", got.Idle)
	}
}

func TestEnvBoolServidorUnificadoAceptaValoresVerdaderos(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "si", "sí", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ORQUESTA_SERVER_CORE_ONLY", value)
			if !envBoolServidorUnificado("ORQUESTA_SERVER_CORE_ONLY") {
				t.Fatalf("valor %q deberia considerarse verdadero", value)
			}
		})
	}
}

func TestEnvBoolServidorUnificadoIgnoraValoresFalsos(t *testing.T) {
	for _, value := range []string{"", "0", "false", "no", "off", "otro"} {
		t.Run(value, func(t *testing.T) {
			if value == "" {
				_ = os.Unsetenv("ORQUESTA_SERVER_CORE_ONLY")
			} else {
				t.Setenv("ORQUESTA_SERVER_CORE_ONLY", value)
			}
			if envBoolServidorUnificado("ORQUESTA_SERVER_CORE_ONLY") {
				t.Fatalf("valor %q no deberia considerarse verdadero", value)
			}
		})
	}
}

func TestPrewarmUnifiedServerReadModelsSiembraStatusYPanel(t *testing.T) {
	prevStatusTimeout := unifiedServerStartupStatusPrimeTimeout
	prevPanelTimeout := unifiedServerStartupPanelPrimeTimeout
	prevStatusFetcher := unifiedServerStartupStatusPrimeFetcher
	prevPanelFetcher := unifiedServerStartupPanelPrimeFetcher
	t.Cleanup(func() {
		unifiedServerStartupStatusPrimeTimeout = prevStatusTimeout
		unifiedServerStartupPanelPrimeTimeout = prevPanelTimeout
		unifiedServerStartupStatusPrimeFetcher = prevStatusFetcher
		unifiedServerStartupPanelPrimeFetcher = prevPanelFetcher
		resetStatusSnapshotCache()
		resetAgentPanelSnapshotCache()
	})
	resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()

	statusCalls := 0
	panelCalls := 0
	unifiedServerStartupStatusPrimeFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		statusCalls++
		return apiStatusResponse{
			AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
			TareasEnProgreso:  []tareaLite{{ID: 40, Agente: "Codex1", Estado: db.TareaEnProgreso}},
			Generado:          time.Now().UTC().Format(time.RFC3339),
		}, true
	}
	unifiedServerStartupPanelPrimeFetcher = func(timeout time.Duration) ([]agentesapp.Row, error) {
		panelCalls++
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex1", Activo: true},
			EstadoOperativo: "trabajando",
		}}, nil
	}

	prewarmUnifiedServerReadModels(nil)

	if statusCalls != 1 {
		t.Fatalf("deberia sembrar status una vez, calls=%d", statusCalls)
	}
	if panelCalls != 1 {
		t.Fatalf("deberia sembrar panel una vez, calls=%d", panelCalls)
	}
	if snapshot, ok := readStatusSnapshotFreshUsable(); !ok || len(snapshot.AgentesActivos) != 1 {
		t.Fatalf("deberia dejar status fresco usable: %+v ok=%v", snapshot, ok)
	}
	if rows, ok := readAgentPanelSnapshotFresh(); !ok || len(rows) != 1 {
		t.Fatalf("deberia dejar panel fresco: rows=%d ok=%v", len(rows), ok)
	}
}

func TestPrewarmUnifiedServerReadModelsOmitePanelSiStatusPuedeSeguirLigero(t *testing.T) {
	prevStatusFetcher := unifiedServerStartupStatusPrimeFetcher
	prevPanelFetcher := unifiedServerStartupPanelPrimeFetcher
	t.Cleanup(func() {
		unifiedServerStartupStatusPrimeFetcher = prevStatusFetcher
		unifiedServerStartupPanelPrimeFetcher = prevPanelFetcher
		resetStatusSnapshotCache()
		resetAgentPanelSnapshotCache()
	})
	resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()

	panelCalls := 0
	unifiedServerStartupStatusPrimeFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		quota := &db.Agente{Nombre: "CodexQuota", Activo: false}
		return apiStatusResponse{
			Agentes:             []*db.Agente{quota},
			AgentesQuotaBlocked: []*db.Agente{quota},
			Generado:            time.Now().UTC().Format(time.RFC3339),
		}, true
	}
	unifiedServerStartupPanelPrimeFetcher = func(timeout time.Duration) ([]agentesapp.Row, error) {
		panelCalls++
		return nil, errors.New("no deberia pedir panel en status ligero")
	}

	prewarmUnifiedServerReadModels(nil)

	if panelCalls != 0 {
		t.Fatalf("no deberia pedir panel cuando status puede quedarse ligero, calls=%d", panelCalls)
	}
}
