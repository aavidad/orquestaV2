/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
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
