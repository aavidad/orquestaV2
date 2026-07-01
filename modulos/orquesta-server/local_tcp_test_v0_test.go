package orquestaserver

import (
	"net"
	"testing"
)

func requireLocalTCPForServerTestV0(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP no disponible para RuntimeV0.RunV0: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("cerrar probe TCP: %v", err)
	}
}
