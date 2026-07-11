package orquestaservershutdown

import "testing"

func TestServerShutdownRequesterAuthorizedV0(t *testing.T) {
	for _, tc := range []struct {
		name        string
		requestedBy string
		want        bool
	}{
		{name: "director", requestedBy: "orquesta-director", want: true},
		{name: "director con espacios", requestedBy: " Director operativo ", want: true},
		{name: "vacio", requestedBy: "", want: false},
		{name: "agente", requestedBy: "agent-director", want: false},
		{name: "sin director", requestedBy: "orquesta-operator", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ServerShutdownRequesterAuthorizedV0(tc.requestedBy); got != tc.want {
				t.Fatalf("ServerShutdownRequesterAuthorizedV0(%q)=%t, want %t", tc.requestedBy, got, tc.want)
			}
		})
	}
}
