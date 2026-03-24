package db

func BackfillVotosPendientes() error {
	estado := PropuestaAbierta
	propuestas, err := ListarPropuestas(&estado, nil)
	if err != nil {
		return err
	}
	for _, propuesta := range propuestas {
		if _, err := asegurarVotosPendientesPropuesta(propuesta.ID); err != nil {
			return err
		}
	}
	return nil
}
