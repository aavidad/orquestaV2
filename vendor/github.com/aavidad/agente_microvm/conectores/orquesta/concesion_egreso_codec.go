package microvm

const codigoConcesionEgresoCodificacionInvalida = "concesion.egreso_codificacion_invalida"

var esquemaDestinoEgresoV1 = esquemaObjetoJSONEstricto{
	"host": esquemaEscalarJSONEstricto, "puertos": esquemaArrayEstricto(esquemaEscalarJSONEstricto),
}

var esquemaConcesionEgresoV1 = esquemaObjetoJSONEstricto{
	"esquema": esquemaEscalarJSONEstricto, "referencia": esquemaEscalarJSONEstricto,
	"destinos":          esquemaArrayEstricto(esquemaObjetoEstricto(esquemaDestinoEgresoV1)),
	"maximo_conexiones": esquemaEscalarJSONEstricto, "limite_tiempo_ms": esquemaEscalarJSONEstricto,
	"limite_subida_bytes": esquemaEscalarJSONEstricto, "limite_bajada_bytes": esquemaEscalarJSONEstricto,
}

// DecodificarConcesionEgresoV1 decodifica exclusivamente la forma JSON
// pública de ConcesionEgreso. La validez operativa de esquema, destinos y
// límites continúa perteneciendo al plan que la contiene.
func DecodificarConcesionEgresoV1(raw []byte) (ConcesionEgreso, error) {
	var concesion ConcesionEgreso
	if !decodificarObjetoJSONEstricto(raw, esquemaConcesionEgresoV1, &concesion) {
		return ConcesionEgreso{}, errorCodificacionEgreso()
	}
	return concesion, nil
}

func errorCodificacionEgreso() error {
	return &ErrorConcesion{Codigo: codigoConcesionEgresoCodificacionInvalida}
}
