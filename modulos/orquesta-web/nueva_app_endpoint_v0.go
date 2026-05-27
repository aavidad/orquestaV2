package orquestaweb

const (
	NuevaAppWebEndpointSchemaV0 = "nueva_app_web_endpoint.v0"

	WebNuevaAppErrMetodoNoSoportadoV0       = webPublicErrMethodUnsupportedV0
	WebNuevaAppErrFormIncompletoV0          = webPublicErrFormIncompleteV0
	WebNuevaAppErrTransporteNoConfiguradoV0 = webPublicErrTransportMissingV0
)

type NuevaAppWebEndpointV0 struct {
	Client         SolicitarNuevaAppClientV0
	DirectorClient ArrancarDirectorAppClientV0
	Catalog        NuevaAppI18nCatalogV0
}

func NewNuevaAppWebEndpointV0(client SolicitarNuevaAppClientV0) NuevaAppWebEndpointV0 {
	return NuevaAppWebEndpointV0{
		Client:  client,
		Catalog: NewNuevaAppI18nCatalogV0(),
	}
}

func (endpoint NuevaAppWebEndpointV0) catalog() NuevaAppI18nCatalogV0 {
	if endpoint.Catalog.messages == nil || endpoint.Catalog.defaultLocale == "" {
		return NewNuevaAppI18nCatalogV0()
	}
	return endpoint.Catalog
}
