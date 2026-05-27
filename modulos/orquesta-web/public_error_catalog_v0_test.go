package orquestaweb

import "testing"

func TestWebPublicErrorCatalogV0CubreCodigosCompartidosV0(t *testing.T) {
	for _, code := range []string{
		WebNuevaAppErrMetodoNoSoportadoV0,
		WebNuevaAppErrFormIncompletoV0,
		WebNuevaAppErrTransporteNoConfiguradoV0,
		WebNuevaAppErrTransporteV0,
		WebNuevaAppErrRespuestaInvalidaV0,
		WebDirectorStatsErrRunRefRequeridoV0,
		WebDirectorStatsErrTransporteV0,
		WebDirectorStatsErrRespuestaInvalidaV0,
	} {
		if !WebPublicErrorCodeKnownV0(code) {
			t.Fatalf("codigo web fuera de catalogo: %s", code)
		}
	}
}
