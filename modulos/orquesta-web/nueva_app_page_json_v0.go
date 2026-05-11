package orquestaweb

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func (endpoint NuevaAppWebEndpointV0) page(locale string, vm WebNuevaAppViewModelV0) NuevaAppWebPageV0 {
	catalog := endpoint.catalog()
	displayLocale := catalog.normalizeLocale(locale)
	titulo, _ := catalog.Lookup(displayLocale, "nueva_app.titulo")
	submit, _ := catalog.Lookup(displayLocale, "nueva_app.accion.submit")
	return NuevaAppWebPageV0{
		SchemaVersion: NuevaAppWebEndpointSchemaV0,
		Locale:        displayLocale,
		Titulo:        titulo,
		Formulario: NuevaAppWebFormularioV0{
			Contrato: "WebNuevaAppFormV0",
			Campos:   NuevaAppWebCamposV0(displayLocale, catalog),
			Acciones: NuevaAppWebAccionesV0{Submit: submit},
		},
		ViewModel: vm,
		Textos:    nuevaAppWebTextosV0(displayLocale, vm, catalog),
		Opciones:  nuevaAppWebOpcionesV0(displayLocale, catalog),
	}
}

func NuevaAppWebCamposV0(locale string, catalog NuevaAppI18nCatalogV0) []NuevaAppWebCampoV0 {
	if catalog.messages == nil || catalog.defaultLocale == "" {
		catalog = NewNuevaAppI18nCatalogV0()
	}
	return nuevaAppWebCamposFromTypeV0(reflect.TypeOf(WebNuevaAppFormV0{}), "", locale, catalog)
}

func nuevaAppWebCamposFromTypeV0(t reflect.Type, prefix, locale string, catalog NuevaAppI18nCatalogV0) []NuevaAppWebCampoV0 {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var out []NuevaAppWebCampoV0
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, omitempty := jsonFieldNameV0(field)
		if name == "" || name == "-" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		fieldType := field.Type
		repeated := false
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Slice {
			repeated = true
			fieldType = fieldType.Elem()
			for fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
		}
		kind := nuevaAppWebFieldKindV0(fieldType, repeated)
		out = append(out, NuevaAppWebCampoV0{
			Path:      path,
			Label:     nuevaAppWebFieldLabelV0(path, locale, catalog),
			Tipo:      kind,
			Repetible: repeated,
			Requerido: !omitempty,
		})
		if fieldType.Kind() == reflect.Struct {
			childPrefix := path
			if repeated {
				childPrefix = path + ".0"
			}
			out = append(out, nuevaAppWebCamposFromTypeV0(fieldType, childPrefix, locale, catalog)...)
		}
	}
	if out == nil {
		return []NuevaAppWebCampoV0{}
	}
	return out
}

func nuevaAppWebFieldKindV0(t reflect.Type, repeated bool) string {
	if repeated && t.Kind() == reflect.Struct {
		return "lista_objeto"
	}
	if repeated {
		return "lista"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "booleano"
	case reflect.Struct:
		return "objeto"
	default:
		return "texto"
	}
}

func jsonFieldNameV0(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name, false
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	omitempty := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

func nuevaAppWebFieldLabelV0(path, locale string, catalog NuevaAppI18nCatalogV0) string {
	text, err := catalog.Lookup(locale, "nueva_app.campo."+stripNuevaAppWebPathIndexesV0(path))
	if err != nil {
		return ""
	}
	return text
}

func stripNuevaAppWebPathIndexesV0(path string) string {
	parts := strings.Split(path, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, ".")
}

func nuevaAppWebTextosV0(locale string, vm WebNuevaAppViewModelV0, catalog NuevaAppI18nCatalogV0) NuevaAppWebTextosV0 {
	return NuevaAppWebTextosV0{
		Estado:          nuevaAppWebLookupV0(catalog, locale, "nueva_app.estado."+string(vm.Estado)),
		BacklogPreview:  nuevaAppWebBacklogPreviewTextosV0(locale, catalog),
		ErroresPublicos: nuevaAppWebErrorTextsV0(locale, vm.ErroresPublicos, catalog),
	}
}

func nuevaAppWebBacklogPreviewTextosV0(locale string, catalog NuevaAppI18nCatalogV0) NuevaAppWebBacklogPreviewTextosV0 {
	return NuevaAppWebBacklogPreviewTextosV0{
		Titulo:      nuevaAppWebLookupV0(catalog, locale, "nueva_app.backlog_preview.titulo"),
		Fases:       nuevaAppWebLookupV0(catalog, locale, "nueva_app.backlog_preview.fases"),
		Microtareas: nuevaAppWebLookupV0(catalog, locale, "nueva_app.backlog_preview.microtareas"),
	}
}

func nuevaAppWebErrorTextsV0(locale string, issues []WebNuevaAppIssueV0, catalog NuevaAppI18nCatalogV0) map[string]string {
	out := map[string]string{}
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		out[code] = nuevaAppWebLookupV0(catalog, locale, "nueva_app.error."+code)
	}
	return out
}

func nuevaAppWebOpcionesV0(locale string, catalog NuevaAppI18nCatalogV0) NuevaAppWebOpcionesV0 {
	return NuevaAppWebOpcionesV0{
		Locales: nuevaAppWebLocaleOptionsV0(catalog),
		Estados: nuevaAppWebOptionsFromKeysV0(locale, catalog, []string{
			string(WebNuevaAppEstadoInicial),
			string(WebNuevaAppEstadoEnviando),
			string(WebNuevaAppEstadoValida),
			string(WebNuevaAppEstadoDirector),
			string(WebNuevaAppEstadoRequiereDatos),
			string(WebNuevaAppEstadoInvalida),
			string(WebNuevaAppEstadoError),
		}, "nueva_app.estado."),
		Errores: nuevaAppWebOptionsFromKeysV0(locale, catalog, []string{
			orquestafactory.ErrAppSpecInvalida,
			orquestafactory.ErrOpcionIncompatible,
			orquestafactory.ErrTargetNoSoportado,
			orquestafactory.ErrIdiomaInvalido,
			orquestafactory.ErrConectorRequeridoNoDisponible,
			WebNuevaAppErrFormIncompletoV0,
			WebNuevaAppErrMetodoNoSoportadoV0,
			WebNuevaAppErrTransporteV0,
			WebNuevaAppErrRespuestaInvalidaV0,
			WebNuevaAppErrTransporteNoConfiguradoV0,
		}, "nueva_app.error."),
	}
}

func nuevaAppWebLocaleOptionsV0(catalog NuevaAppI18nCatalogV0) []NuevaAppWebOpcionV0 {
	locales := catalog.SupportedLocales()
	out := make([]NuevaAppWebOpcionV0, 0, len(locales))
	for _, locale := range locales {
		out = append(out, NuevaAppWebOpcionV0{Valor: locale, Label: locale})
	}
	return out
}

func nuevaAppWebOptionsFromKeysV0(locale string, catalog NuevaAppI18nCatalogV0, values []string, prefix string) []NuevaAppWebOpcionV0 {
	out := make([]NuevaAppWebOpcionV0, 0, len(values))
	for _, value := range values {
		out = append(out, NuevaAppWebOpcionV0{
			Valor: value,
			Label: nuevaAppWebLookupV0(catalog, locale, prefix+value),
		})
	}
	return out
}

func nuevaAppWebLookupV0(catalog NuevaAppI18nCatalogV0, locale, key string) string {
	text, err := catalog.Lookup(locale, key)
	if err != nil {
		return text
	}
	return text
}

func localeFromNuevaAppRequestV0(r *http.Request, catalog NuevaAppI18nCatalogV0) string {
	return catalog.normalizeLocale(firstNuevaAppLocaleV0(r.URL.Query().Get("locale"), r.URL.Query().Get("lang"), r.Header.Get("Accept-Language")))
}

func firstNuevaAppLocaleV0(values ...string) string {
	for _, value := range values {
		if locale := strings.TrimSpace(strings.Split(value, ",")[0]); locale != "" {
			return locale
		}
	}
	return NuevaAppI18nDefaultLocaleV0
}
