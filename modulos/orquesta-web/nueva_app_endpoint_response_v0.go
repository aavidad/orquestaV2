package orquestaweb

import (
	"encoding/json"
	"net/http"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

type NuevaAppWebPageV0 struct {
	SchemaVersion string                  `json:"schema_version"`
	Locale        string                  `json:"locale"`
	Titulo        string                  `json:"titulo"`
	Formulario    NuevaAppWebFormularioV0 `json:"formulario"`
	ViewModel     WebNuevaAppViewModelV0  `json:"view_model"`
	Textos        NuevaAppWebTextosV0     `json:"textos"`
	Opciones      NuevaAppWebOpcionesV0   `json:"opciones"`
}

type NuevaAppWebFormularioV0 struct {
	Contrato string                `json:"contrato"`
	Campos   []NuevaAppWebCampoV0  `json:"campos"`
	Acciones NuevaAppWebAccionesV0 `json:"acciones"`
}

type NuevaAppWebCampoV0 struct {
	Path      string `json:"path"`
	Label     string `json:"label,omitempty"`
	Tipo      string `json:"tipo"`
	Repetible bool   `json:"repetible,omitempty"`
	Requerido bool   `json:"requerido,omitempty"`
}

type NuevaAppWebAccionesV0 struct {
	Submit  string `json:"submit"`
	Preview string `json:"preview"`
}

type NuevaAppWebTextosV0 struct {
	Estado          string                            `json:"estado"`
	BacklogPreview  NuevaAppWebBacklogPreviewTextosV0 `json:"backlog_preview"`
	ErroresPublicos map[string]string                 `json:"errores_publicos"`
}

type NuevaAppWebBacklogPreviewTextosV0 struct {
	Titulo      string `json:"titulo"`
	Fases       string `json:"fases"`
	Microtareas string `json:"microtareas"`
}

type NuevaAppWebOpcionesV0 struct {
	Locales []NuevaAppWebOpcionV0 `json:"locales"`
	Estados []NuevaAppWebOpcionV0 `json:"estados"`
	Errores []NuevaAppWebOpcionV0 `json:"errores_publicos"`
}

type NuevaAppWebOpcionV0 struct {
	Valor string `json:"valor"`
	Label string `json:"label"`
}

func initialNuevaAppViewModelV0(locale string) WebNuevaAppViewModelV0 {
	return WebNuevaAppViewModelV0{
		Locale:              trimV0(locale),
		Estado:              WebNuevaAppEstadoInicial,
		BacklogPreview:      emptyBacklogPreviewV0(),
		PreguntasAbiertas:   []string{},
		Fases:               []WebNuevaAppFaseV0{},
		Microtareas:         []WebNuevaAppMicrotareaV0{},
		ContratosRequeridos: []string{},
		Riesgos:             []string{},
	}
}

func nuevaAppWebPublicErrorViewModelV0(requestID, locale string, estado WebNuevaAppEstadoV0, code string) WebNuevaAppViewModelV0 {
	vm := NewWebNuevaAppErrorViewModelV0(requestID, locale, []orquestafactory.ValidationIssue{
		publicValidationIssueV0(code, "", code),
	})
	vm.Estado = estado
	return vm
}

func writeNuevaAppWebPageV0(w http.ResponseWriter, status int, page NuevaAppWebPageV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(page)
}
