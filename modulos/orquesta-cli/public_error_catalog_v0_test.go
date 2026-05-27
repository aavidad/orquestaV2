package orquestacli

import "testing"

func TestCliPublicErrorCatalogV0CubreCodigosBaseV0(t *testing.T) {
	for _, code := range []string{
		CliErrOpcionInvalidaV0,
		CliErrInputTooLargeV0,
		CliErrConfiguracionInvalidaV0,
		CliErrContratoNoConfiguradoV0,
		CliErrRespuestaInvalidaV0,
		CliErrErrorTransporteV0,
		CliErrSalidaNoSerializableV0,
	} {
		entry, ok := CliPublicErrorDescriptorV0(code)
		if !ok || entry.I18nKey == "" || entry.Severity == "" {
			t.Fatalf("codigo CLI fuera de catalogo: %s %+v", code, entry)
		}
	}
}
