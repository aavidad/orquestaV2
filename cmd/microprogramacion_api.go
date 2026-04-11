package cmd

import (
	"net/http"
	"strconv"
	"strings"

	"orquesta/microprogramacionapp"
)

type apiEspecificacionFuncionCreateRequest struct {
	TareaID                *int64   `json:"tarea_id,omitempty"`
	ProyectoID             *int64   `json:"proyecto_id,omitempty"`
	Proyecto               string   `json:"proyecto,omitempty"`
	Titulo                 string   `json:"titulo"`
	ArchivoObjetivo        string   `json:"archivo_objetivo"`
	SimboloObjetivo        string   `json:"simbolo_objetivo"`
	Descripcion            string   `json:"descripcion"`
	Precondiciones         []string `json:"precondiciones,omitempty"`
	Postcondiciones        []string `json:"postcondiciones,omitempty"`
	DependenciasPermitidas []string `json:"dependencias_permitidas,omitempty"`
	DependenciasProhibidas []string `json:"dependencias_prohibidas,omitempty"`
	TestsObligatorios      []string `json:"tests_obligatorios,omitempty"`
	WriteSet               []string `json:"write_set,omitempty"`
	FormatoSalida          string   `json:"formato_salida,omitempty"`
	CreadoPor              string   `json:"creado_por,omitempty"`
}

type apiEspecificacionFuncionResponse struct {
	Especificacion *microprogramacionapp.EspecificacionFuncion `json:"especificacion"`
}

type apiEspecificacionesFuncionResponse struct {
	Especificaciones []*microprogramacionapp.EspecificacionFuncion `json:"especificaciones"`
}

type apiEspecificacionFuncionCreateResponse struct {
	OK             bool                                        `json:"ok"`
	ID             int64                                       `json:"id"`
	Especificacion *microprogramacionapp.EspecificacionFuncion `json:"especificacion"`
}

type apiEmitirMicrotareaRequest struct {
	Contexto string `json:"contexto,omitempty"`
}

type apiMicrotareaEmitidaResponse struct {
	Microtarea *microprogramacionapp.MicrotareaEmitida `json:"microtarea"`
}

type apiDespacharMicrotareaRequest struct {
	Agente     string `json:"agente"`
	Proyecto   string `json:"proyecto,omitempty"`
	ProyectoID *int64 `json:"proyecto_id,omitempty"`
	Contexto   string `json:"contexto,omitempty"`
}

type apiMicrotareaDespachadaResponse struct {
	Despacho *microprogramacionapp.MicrotareaDespachada `json:"despacho"`
}

type apiValidarEntregaRequest struct {
	SimboloEntregado   string   `json:"simbolo_entregado"`
	WriteSetEntregado  []string `json:"write_set_entregado,omitempty"`
	DependenciasUsadas []string `json:"dependencias_usadas,omitempty"`
	TestsEjecutados    []string `json:"tests_ejecutados,omitempty"`
	TestsFallidos      []string `json:"tests_fallidos,omitempty"`
	Evidencia          string   `json:"evidencia,omitempty"`
}

type apiValidacionEntregaResponse struct {
	Resultado *microprogramacionapp.ResultadoValidacionEntrega `json:"resultado"`
}

func apiHandlerMicroprogramacionEspecificaciones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filtro, err := apiFiltroEspecificacionesFuncionDesdeRequest(r)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		items, err := microprogramacionService.Listar(filtro)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiEspecificacionesFuncionResponse{Especificaciones: items})
	case http.MethodPost:
		var req apiEspecificacionFuncionCreateRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID, err := resolverProyectoEspecificacionFuncion(req.ProyectoID, req.Proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
			TareaID:                req.TareaID,
			ProyectoID:             proyectoID,
			Titulo:                 req.Titulo,
			ArchivoObjetivo:        req.ArchivoObjetivo,
			SimboloObjetivo:        req.SimboloObjetivo,
			Descripcion:            req.Descripcion,
			Precondiciones:         req.Precondiciones,
			Postcondiciones:        req.Postcondiciones,
			DependenciasPermitidas: req.DependenciasPermitidas,
			DependenciasProhibidas: req.DependenciasProhibidas,
			TestsObligatorios:      req.TestsObligatorios,
			WriteSet:               req.WriteSet,
			FormatoSalida:          req.FormatoSalida,
			CreadoPor:              req.CreadoPor,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		item, err := microprogramacionService.Obtener(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiEspecificacionFuncionCreateResponse{
			OK:             true,
			ID:             id,
			Especificacion: item,
		})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterMicroprogramacionEspecificaciones(w http.ResponseWriter, r *http.Request) {
	ref := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/microprogramacion/especificaciones/"), "/")
	if ref == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(ref, "/emitir") {
		apiHandlerEmitirMicrotarea(w, r, strings.TrimSuffix(ref, "/emitir"))
		return
	}
	if strings.HasSuffix(ref, "/despachar") {
		apiHandlerDespacharMicrotarea(w, r, strings.TrimSuffix(ref, "/despachar"))
		return
	}
	if strings.HasSuffix(ref, "/validar-entrega") {
		apiHandlerValidarEntregaMicrotarea(w, r, strings.TrimSuffix(ref, "/validar-entrega"))
		return
	}
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	id, err := strconv.ParseInt(ref, 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, errParametroInvalido("id"))
		return
	}
	item, err := microprogramacionService.Obtener(id)
	if err != nil || item == nil {
		apiError(w, http.StatusNotFound, errNoEncontrado("especificacion_funcion"))
		return
	}
	apiWriteJSON(w, http.StatusOK, apiEspecificacionFuncionResponse{Especificacion: item})
}

func apiHandlerEmitirMicrotarea(w http.ResponseWriter, r *http.Request, ref string) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, errParametroInvalido("id"))
		return
	}
	var req apiEmitirMicrotareaRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	item, err := microprogramacionService.Emitir(id, microprogramacionapp.EntradaEmitirMicrotarea{
		Contexto: req.Contexto,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiMicrotareaEmitidaResponse{Microtarea: item})
}

func apiHandlerDespacharMicrotarea(w http.ResponseWriter, r *http.Request, ref string) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, errParametroInvalido("id"))
		return
	}
	var req apiDespacharMicrotareaRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	proyectoID, err := resolverProyectoEspecificacionFuncion(req.ProyectoID, req.Proyecto)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	despacho, err := microprogramacionService.EmitirYDespachar(id, microprogramacionapp.EntradaDespacharMicrotarea{
		AgenteDestino: strings.TrimSpace(req.Agente),
		ProyectoID:    proyectoID,
		Contexto:      req.Contexto,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiMicrotareaDespachadaResponse{Despacho: despacho})
}

func apiHandlerValidarEntregaMicrotarea(w http.ResponseWriter, r *http.Request, ref string) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, errParametroInvalido("id"))
		return
	}
	var req apiValidarEntregaRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := microprogramacionService.ValidarEntrega(id, microprogramacionapp.EntradaValidarEntrega{
		SimboloEntregado:   req.SimboloEntregado,
		WriteSetEntregado:  req.WriteSetEntregado,
		DependenciasUsadas: req.DependenciasUsadas,
		TestsEjecutados:    req.TestsEjecutados,
		TestsFallidos:      req.TestsFallidos,
		Evidencia:          req.Evidencia,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiValidacionEntregaResponse{Resultado: resultado})
}

func apiFiltroEspecificacionesFuncionDesdeRequest(r *http.Request) (microprogramacionapp.FiltroEspecificaciones, error) {
	var filtro microprogramacionapp.FiltroEspecificaciones
	query := r.URL.Query()
	if tarea := strings.TrimSpace(query.Get("tarea_id")); tarea != "" {
		id, err := strconv.ParseInt(tarea, 10, 64)
		if err != nil || id <= 0 {
			return filtro, errParametroInvalido("tarea_id")
		}
		filtro.TareaID = &id
	}
	if proyecto := strings.TrimSpace(query.Get("proyecto")); proyecto != "" {
		id, err := tareasService.ResolveProjectID(proyecto)
		if err != nil {
			return filtro, errParametroInvalido("proyecto")
		}
		filtro.ProyectoID = id
	}
	if filtro.ProyectoID == nil {
		if proyecto := strings.TrimSpace(query.Get("proyecto_id")); proyecto != "" {
			id, err := strconv.ParseInt(proyecto, 10, 64)
			if err != nil || id <= 0 {
				return filtro, errParametroInvalido("proyecto_id")
			}
			filtro.ProyectoID = &id
		}
	}
	if estado := strings.TrimSpace(query.Get("estado")); estado != "" {
		value := microprogramacionapp.EstadoEspecificacion(estado)
		filtro.Estado = &value
	}
	if limit := strings.TrimSpace(query.Get("limit")); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil || value < 0 {
			return filtro, errParametroInvalido("limit")
		}
		filtro.Limit = value
	}
	return filtro, nil
}

func resolverProyectoEspecificacionFuncion(proyectoID *int64, proyecto string) (*int64, error) {
	if proyectoID != nil && strings.TrimSpace(proyecto) != "" {
		return nil, errParametroInvalido("proyecto")
	}
	if proyectoID != nil {
		if *proyectoID <= 0 {
			return nil, errParametroInvalido("proyecto_id")
		}
		return proyectoID, nil
	}
	if strings.TrimSpace(proyecto) == "" {
		return nil, nil
	}
	id, err := tareasService.ResolveProjectID(proyecto)
	if err != nil {
		return nil, errParametroInvalido("proyecto")
	}
	return id, nil
}

func errParametroInvalido(nombre string) error {
	return &apiValidationError{message: "parametro invalido: " + strings.TrimSpace(nombre)}
}

func errNoEncontrado(tipo string) error {
	return &apiValidationError{message: strings.TrimSpace(tipo) + " no encontrado"}
}

type apiValidationError struct {
	message string
}

func (e *apiValidationError) Error() string {
	return e.message
}
