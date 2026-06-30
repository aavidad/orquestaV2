package orquestaweb

import (
	"fmt"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func estadoFromSpecV0(spec orquestafactory.AppSpecV0) WebNuevaAppEstadoV0 {
	if len(spec.Validation.Errores) > 0 {
		return WebNuevaAppEstadoInvalida
	}
	if len(spec.Scope.PreguntasAbiertas) > 0 {
		return WebNuevaAppEstadoRequiereDatos
	}
	if spec.Validation.Estado == "valida" {
		return WebNuevaAppEstadoValida
	}
	if spec.Validation.Estado == "" {
		return WebNuevaAppEstadoInicial
	}
	return WebNuevaAppEstadoError
}

func resumenFromSpecV0(spec orquestafactory.AppSpecV0) WebNuevaAppResumenV0 {
	return WebNuevaAppResumenV0{
		Nombre:        spec.App.Nombre,
		Slug:          spec.App.Slug,
		Objetivo:      spec.App.Objetivo,
		Descripcion:   spec.App.Descripcion,
		TipoApp:       spec.App.TipoApp,
		RequestKind:   spec.RequestKind,
		ExecutionMode: spec.ExecutionMode,
		Locale:        spec.Locale,
		DefaultLocale: spec.I18N.DefaultLocale,
		I18NEnabled:   spec.I18N.Enabled,
		Plataformas:   compactStringsV0(spec.Platforms),
		DeployTarget:  spec.Deploy.Target,
	}
}

func defaultsFromSpecV0(values []orquestafactory.DefaultAppliedV0) []WebNuevaAppDefaultV0 {
	out := make([]WebNuevaAppDefaultV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebNuevaAppDefaultV0{
			Campo:  value.Campo,
			Valor:  stringifyDefaultV0(value.Valor),
			Motivo: value.Motivo,
		})
	}
	if out == nil {
		return []WebNuevaAppDefaultV0{}
	}
	return out
}

func stringifyDefaultV0(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case nil:
		return ""
	case []string:
		return strings.Join(compactStringsV0(typed), ",")
	default:
		return trimV0(fmt.Sprint(typed))
	}
}

func issuesFromFactoryV0(values []orquestafactory.ValidationIssue, locale string) []WebNuevaAppIssueV0 {
	out := make([]WebNuevaAppIssueV0, 0, len(values))
	for _, value := range values {
		code := trimV0(value.Code)
		if code == "" {
			code = orquestafactory.ErrAppSpecInvalida
		}
		out = append(out, WebNuevaAppIssueV0{
			Code:    code,
			Field:   trimV0(value.Field),
			Message: nuevaAppPublicIssueMessageV0(locale, code, value.Message),
		})
	}
	if out == nil {
		return []WebNuevaAppIssueV0{}
	}
	return out
}

func nuevaAppPublicIssueMessageV0(locale, code, rawMessage string) string {
	catalog := NewNuevaAppI18nCatalogV0()
	if nuevaAppIssueLooksRequiredV0(rawMessage) {
		return nuevaAppWebLookupV0(catalog, locale, "nueva_app.validation.required")
	}
	if code != "" {
		if text, err := catalog.Lookup(locale, "nueva_app.error."+code); err == nil {
			return text
		}
	}
	return nuevaAppWebLookupV0(catalog, locale, "nueva_app.error."+orquestafactory.ErrAppSpecInvalida)
}

func nuevaAppIssueLooksRequiredV0(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return normalized == "campo obligatorio" ||
		strings.Contains(normalized, "required field") ||
		strings.Contains(normalized, "complete this field") ||
		strings.Contains(normalized, "please fill")
}

func fasesFromBacklogV0(values []orquestafactory.FaseInicialV0) []WebNuevaAppFaseV0 {
	out := make([]WebNuevaAppFaseV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebNuevaAppFaseV0{
			ID:       value.ID,
			Nombre:   value.Nombre,
			Objetivo: value.Objetivo,
			Orden:    value.Orden,
		})
	}
	if out == nil {
		return []WebNuevaAppFaseV0{}
	}
	return out
}

func microtareasFromBacklogV0(values []orquestafactory.MicrotareaPropuestaV0) []WebNuevaAppMicrotareaV0 {
	out := make([]WebNuevaAppMicrotareaV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebNuevaAppMicrotareaV0{
			ID:               value.ID,
			Fase:             value.Fase,
			ModuloSugerido:   value.ModuloSugerido,
			Objetivo:         value.Objetivo,
			WriteSetPrevisto: compactStringsV0(value.WriteSetPrevisto),
			Contrato:         value.Contrato,
			Validacion:       value.Validacion,
			Bloqueos:         compactStringsV0(value.Bloqueos),
		})
	}
	if out == nil {
		return []WebNuevaAppMicrotareaV0{}
	}
	return out
}
