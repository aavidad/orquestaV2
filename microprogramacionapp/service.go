package microprogramacionapp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type Repositorio interface {
	CrearEspecificacionFuncion(spec *EspecificacionFuncion) (int64, error)
	ObtenerEspecificacionFuncion(id int64) (*EspecificacionFuncion, error)
	ListarEspecificacionesFuncion(filtro FiltroEspecificaciones) ([]*EspecificacionFuncion, error)
}

type Despachador interface {
	DespacharMicrotarea(solicitud SolicitudDespachoMicrotarea) (*MicrotareaDespachada, error)
}

type RecolectorEntregaGit interface {
	CapturarEntregaGit(agente string, proyectoID *int64, proyectoSlug string) (*EntregaGitCapturada, error)
}

type RecolectorEntregaGitPreferente interface {
	CapturarEntregaGitPreferente(agente string, proyectoID *int64, proyectoSlug string, entrada EntradaRegistrarEntregaGit) (*EntregaGitCapturada, error)
}

type IntegradorGit interface {
	RegistrarSolicitudMerge(entrada SolicitudMergeMicroprogramacion) (int64, error)
}

type SolicitudMergeMicroprogramacion struct {
	ProyectoSlug  string
	SourceBranch  string
	TargetBranch  string
	SolicitadoPor string
	CommitOrigen  string
	Notas         string
	MetadataJSON  string
}

type Servicio struct {
	repositorio   Repositorio
	despachador   Despachador
	escritor      EscritorArchivos
	recolectorGit RecolectorEntregaGit
	integradorGit IntegradorGit
}

func NewService(repositorio Repositorio) *Servicio {
	return &Servicio{repositorio: repositorio}
}

func (s *Servicio) SetDespachador(despachador Despachador) {
	if s == nil {
		return
	}
	s.despachador = despachador
}

func (s *Servicio) SetEscritorArchivos(escritor EscritorArchivos) {
	if s == nil {
		return
	}
	s.escritor = escritor
}

func (s *Servicio) SetRecolectorEntregaGit(recolector RecolectorEntregaGit) {
	if s == nil {
		return
	}
	s.recolectorGit = recolector
}

func (s *Servicio) SetIntegradorGit(integrador IntegradorGit) {
	if s == nil {
		return
	}
	s.integradorGit = integrador
}

func (s *Servicio) Crear(entrada EntradaCrearEspecificacion) (int64, error) {
	spec, err := normalizarEntradaCrearEspecificacion(entrada)
	if err != nil {
		return 0, err
	}
	return s.repositorio.CrearEspecificacionFuncion(spec)
}

func (s *Servicio) Obtener(id int64) (*EspecificacionFuncion, error) {
	return s.repositorio.ObtenerEspecificacionFuncion(id)
}

func (s *Servicio) Listar(filtro FiltroEspecificaciones) ([]*EspecificacionFuncion, error) {
	return s.repositorio.ListarEspecificacionesFuncion(filtro)
}

func (s *Servicio) Emitir(id int64, entrada EntradaEmitirMicrotarea) (*MicrotareaEmitida, error) {
	item, err := s.repositorio.ObtenerEspecificacionFuncion(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("especificacion no encontrada")
	}
	return construirMicrotareaEmitida(item, entrada), nil
}

func (s *Servicio) EmitirYDespachar(id int64, entrada EntradaDespacharMicrotarea) (*MicrotareaDespachada, error) {
	item, err := s.repositorio.ObtenerEspecificacionFuncion(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("especificacion no encontrada")
	}
	if s.despachador == nil {
		return nil, fmt.Errorf("despachador de microprogramacion no configurado")
	}
	agenteDestino := strings.TrimSpace(entrada.AgenteDestino)
	if agenteDestino == "" {
		return nil, fmt.Errorf("agente destino obligatorio")
	}
	microtarea := construirMicrotareaEmitida(item, EntradaEmitirMicrotarea{Contexto: entrada.Contexto})
	proyectoID := entrada.ProyectoID
	if proyectoID == nil {
		proyectoID = item.ProyectoID
	}
	return s.despachador.DespacharMicrotarea(SolicitudDespachoMicrotarea{
		EspecificacionID: item.ID,
		ProyectoID:       proyectoID,
		AgenteDestino:    agenteDestino,
		Microtarea:       microtarea,
	})
}

func (s *Servicio) ValidarEntrega(id int64, entrada EntradaValidarEntrega) (*ResultadoValidacionEntrega, error) {
	item, err := s.repositorio.ObtenerEspecificacionFuncion(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("especificacion no encontrada")
	}
	resultado := &ResultadoValidacionEntrega{
		EspecificacionID:       item.ID,
		ArchivoObjetivo:        item.ArchivoObjetivo,
		SimboloObjetivo:        item.SimboloObjetivo,
		WriteSetPermitido:      slices.Clone(item.WriteSet),
		DependenciasPermitidas: slices.Clone(item.DependenciasPermitidas),
		DependenciasProhibidas: slices.Clone(item.DependenciasProhibidas),
		TestsObligatorios:      slices.Clone(item.TestsObligatorios),
	}
	addHallazgo := func(campo, codigo, mensaje string) {
		resultado.Hallazgos = append(resultado.Hallazgos, HallazgoValidacionEntrega{
			Campo:   strings.TrimSpace(campo),
			Codigo:  strings.TrimSpace(codigo),
			Mensaje: strings.TrimSpace(mensaje),
		})
	}

	simboloEntregado := strings.TrimSpace(entrada.SimboloEntregado)
	switch {
	case simboloEntregado == "":
		addHallazgo("simbolo", "simbolo_requerido", "la entrega debe declarar el simbolo afectado")
	case simboloEntregado != item.SimboloObjetivo:
		addHallazgo("simbolo", "simbolo_distinto", fmt.Sprintf("la entrega afecta %q pero la especificacion exige %q", simboloEntregado, item.SimboloObjetivo))
	}

	if strings.TrimSpace(entrada.Evidencia) == "" {
		addHallazgo("evidencia", "evidencia_requerida", "la entrega debe incluir evidencia verificable")
	}

	writeSetEntregado, err := normalizarWriteSetEntregado(entrada.WriteSetEntregado)
	if err != nil {
		addHallazgo("write_set", "write_set_invalido", err.Error())
	} else {
		if len(writeSetEntregado) == 0 {
			addHallazgo("write_set", "write_set_vacio", "la entrega debe declarar los archivos tocados")
		}
		if !slices.Contains(writeSetEntregado, item.ArchivoObjetivo) {
			addHallazgo("write_set", "archivo_objetivo_ausente", fmt.Sprintf("la entrega debe incluir el archivo objetivo %q", item.ArchivoObjetivo))
		}
		for _, ruta := range writeSetEntregado {
			if !slices.Contains(item.WriteSet, ruta) {
				addHallazgo("write_set", "fuera_de_write_set", fmt.Sprintf("el archivo %q queda fuera del write_set permitido", ruta))
			}
		}
	}

	dependenciasUsadas := normalizarListaDependencias(entrada.DependenciasUsadas)
	for _, dep := range dependenciasUsadas {
		if dependenciaIncluida(item.DependenciasProhibidas, dep) {
			addHallazgo("dependencias", "dependencia_prohibida", fmt.Sprintf("la entrega declara la dependencia prohibida %q", dep))
			continue
		}
		if len(item.DependenciasPermitidas) > 0 && !dependenciaIncluida(item.DependenciasPermitidas, dep) {
			addHallazgo("dependencias", "dependencia_no_permitida", fmt.Sprintf("la dependencia %q no está permitida por la especificacion", dep))
		}
	}

	testsEjecutados := normalizarListaTexto(entrada.TestsEjecutados)
	testsFallidos := normalizarListaTexto(entrada.TestsFallidos)
	for _, test := range item.TestsObligatorios {
		if !slices.Contains(testsEjecutados, test) {
			addHallazgo("tests", "test_obligatorio_no_ejecutado", fmt.Sprintf("falta ejecutar el test obligatorio %q", test))
		}
		if slices.Contains(testsFallidos, test) {
			addHallazgo("tests", "test_obligatorio_fallido", fmt.Sprintf("el test obligatorio %q ha fallado", test))
		}
	}
	for _, test := range testsFallidos {
		if !slices.Contains(item.TestsObligatorios, test) {
			addHallazgo("tests", "test_fallido", fmt.Sprintf("la entrega reporta un test fallido %q", test))
		}
	}

	resultado.Valida = len(resultado.Hallazgos) == 0
	return resultado, nil
}

func (s *Servicio) MaterializarEntrega(id int64, entrada EntradaMaterializarEntrega) (*ResultadoMaterializarEntrega, error) {
	item, err := s.repositorio.ObtenerEspecificacionFuncion(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("especificacion no encontrada")
	}
	if s.escritor == nil {
		return nil, fmt.Errorf("escritor de archivos no configurado")
	}
	if strings.TrimSpace(entrada.Evidencia) == "" {
		return nil, fmt.Errorf("evidencia obligatoria para materializar entrega")
	}
	if strings.TrimSpace(entrada.RaizProyecto) == "" {
		return nil, fmt.Errorf("raiz de proyecto obligatoria")
	}
	archivos, rutasEntregadas, err := normalizarArchivosEntrega(item, entrada.Archivos)
	if err != nil {
		return nil, err
	}
	if err := validarArchivosMaterializables(item, archivos); err != nil {
		return nil, err
	}
	escritos, err := s.escritor.Escribir(entrada.RaizProyecto, archivos)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(escritos, rutasEntregadas) {
		// Mantener el conjunto final escrito aunque el escritor reordene.
		rutasEntregadas = escritos
	}
	return &ResultadoMaterializarEntrega{
		EspecificacionID:       item.ID,
		ArchivoObjetivo:        item.ArchivoObjetivo,
		WriteSetPermitido:      slices.Clone(item.WriteSet),
		ArchivosMaterializados: rutasEntregadas,
	}, nil
}

func (s *Servicio) MaterializarEntregaDesdeRespuesta(id int64, raizProyecto, respuesta, evidencia string) (*ResultadoMaterializarEntrega, error) {
	return s.MaterializarEntrega(id, EntradaMaterializarEntrega{
		RaizProyecto: strings.TrimSpace(raizProyecto),
		Archivos:     ExtraerArchivosEntrega(respuesta),
		Evidencia:    strings.TrimSpace(evidencia),
	})
}

func (s *Servicio) RegistrarEntregaGit(id int64, entrada EntradaRegistrarEntregaGit) (*ResultadoRegistrarEntregaGit, error) {
	item, err := s.repositorio.ObtenerEspecificacionFuncion(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("especificacion no encontrada")
	}
	if s.recolectorGit == nil {
		return nil, fmt.Errorf("recolector de entrega git no configurado")
	}
	if s.integradorGit == nil {
		return nil, fmt.Errorf("integrador git no configurado")
	}
	agente := strings.TrimSpace(entrada.Agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	var captura *EntregaGitCapturada
	var errCaptura error
	if recolectorPreferente, ok := s.recolectorGit.(RecolectorEntregaGitPreferente); ok {
		captura, errCaptura = recolectorPreferente.CapturarEntregaGitPreferente(agente, entrada.ProyectoID, strings.TrimSpace(entrada.ProyectoSlug), entrada)
	} else {
		captura, errCaptura = s.recolectorGit.CapturarEntregaGit(agente, entrada.ProyectoID, strings.TrimSpace(entrada.ProyectoSlug))
	}
	if errCaptura != nil {
		return nil, errCaptura
	}
	if captura == nil {
		return nil, fmt.Errorf("no existe entrega git capturada")
	}
	if strings.TrimSpace(captura.ProyectoSlug) == "" {
		return nil, fmt.Errorf("proyecto slug obligatorio en entrega git")
	}
	if strings.TrimSpace(captura.Branch) == "" {
		return nil, fmt.Errorf("branch de entrega obligatoria")
	}
	if strings.TrimSpace(captura.RutaWorktree) == "" {
		return nil, fmt.Errorf("ruta de worktree obligatoria")
	}
	archivosEntregados, err := normalizarWriteSetEntregado(captura.ArchivosModificados)
	if err != nil {
		return nil, err
	}
	if len(archivosEntregados) == 0 {
		return nil, fmt.Errorf("la entrega git no contiene archivos modificados")
	}
	if !slices.Contains(archivosEntregados, item.ArchivoObjetivo) {
		return nil, fmt.Errorf("la entrega git no incluye el archivo objetivo %q", item.ArchivoObjetivo)
	}
	for _, ruta := range archivosEntregados {
		if !slices.Contains(item.WriteSet, ruta) {
			return nil, fmt.Errorf("el archivo %q queda fuera del write_set permitido", ruta)
		}
	}
	targetBranch := normalizarTargetBranchGit(captura.BaseRef)
	if targetBranch == "" {
		targetBranch = "main"
	}
	solicitadoPor := strings.TrimSpace(entrada.SolicitadoPor)
	if solicitadoPor == "" {
		solicitadoPor = agente
	}
	notas := fmt.Sprintf("Entrega de microprogramacion registrada desde worktree %s.", strings.TrimSpace(captura.RutaWorktree))
	if evidencia := strings.TrimSpace(entrada.Evidencia); evidencia != "" {
		notas = notas + "\n" + evidencia
	}
	metadataJSON := buildGitMetadataMicroprogramacion(item.ID, captura, item)
	mergeID, err := s.integradorGit.RegistrarSolicitudMerge(SolicitudMergeMicroprogramacion{
		ProyectoSlug:  strings.TrimSpace(captura.ProyectoSlug),
		SourceBranch:  strings.TrimSpace(captura.Branch),
		TargetBranch:  targetBranch,
		SolicitadoPor: solicitadoPor,
		CommitOrigen:  strings.TrimSpace(captura.HeadCommit),
		Notas:         notas,
		MetadataJSON:  metadataJSON,
	})
	if err != nil {
		return nil, err
	}
	return &ResultadoRegistrarEntregaGit{
		EspecificacionID:   item.ID,
		ArchivoObjetivo:    item.ArchivoObjetivo,
		WriteSetPermitido:  slices.Clone(item.WriteSet),
		WorktreeID:         captura.WorktreeID,
		RutaWorktree:       strings.TrimSpace(captura.RutaWorktree),
		SourceBranch:       strings.TrimSpace(captura.Branch),
		TargetBranch:       targetBranch,
		HeadCommit:         strings.TrimSpace(captura.HeadCommit),
		ArchivosEntregados: archivosEntregados,
		GitMergeID:         mergeID,
	}, nil
}

func normalizarEntradaCrearEspecificacion(entrada EntradaCrearEspecificacion) (*EspecificacionFuncion, error) {
	archivoObjetivo, err := normalizarRutaRelativa(entrada.ArchivoObjetivo)
	if err != nil {
		return nil, fmt.Errorf("archivo objetivo inválido: %w", err)
	}
	simboloObjetivo := strings.TrimSpace(entrada.SimboloObjetivo)
	if simboloObjetivo == "" {
		return nil, fmt.Errorf("el simbolo objetivo es obligatorio")
	}
	descripcion := strings.TrimSpace(entrada.Descripcion)
	if descripcion == "" {
		return nil, fmt.Errorf("la descripcion es obligatoria")
	}
	writeSet, err := normalizarWriteSet(entrada.WriteSet, archivoObjetivo)
	if err != nil {
		return nil, err
	}
	testsObligatorios := normalizarListaTexto(entrada.TestsObligatorios)
	if len(testsObligatorios) == 0 {
		return nil, fmt.Errorf("debe declararse al menos un test obligatorio")
	}
	dependenciasPermitidas := normalizarListaDependencias(entrada.DependenciasPermitidas)
	dependenciasProhibidas := normalizarListaDependencias(entrada.DependenciasProhibidas)
	if overlap := dependenciaSolapada(dependenciasPermitidas, dependenciasProhibidas); overlap != "" {
		return nil, fmt.Errorf("la dependencia %q no puede estar permitida y prohibida a la vez", overlap)
	}
	titulo := strings.TrimSpace(entrada.Titulo)
	if titulo == "" {
		titulo = fmt.Sprintf("%s::%s", archivoObjetivo, simboloObjetivo)
	}
	formatoSalida := strings.TrimSpace(entrada.FormatoSalida)
	if formatoSalida == "" {
		formatoSalida = "patch+evidencia"
	}
	creadoPor := strings.TrimSpace(entrada.CreadoPor)
	if creadoPor == "" {
		creadoPor = "alberto"
	}
	return &EspecificacionFuncion{
		TareaID:                entrada.TareaID,
		ProyectoID:             entrada.ProyectoID,
		Titulo:                 titulo,
		ArchivoObjetivo:        archivoObjetivo,
		SimboloObjetivo:        simboloObjetivo,
		Descripcion:            descripcion,
		Precondiciones:         normalizarListaTexto(entrada.Precondiciones),
		Postcondiciones:        normalizarListaTexto(entrada.Postcondiciones),
		DependenciasPermitidas: dependenciasPermitidas,
		DependenciasProhibidas: dependenciasProhibidas,
		TestsObligatorios:      testsObligatorios,
		WriteSet:               writeSet,
		FormatoSalida:          formatoSalida,
		Estado:                 EstadoEspecificacionActiva,
		Version:                1,
		CreadoPor:              creadoPor,
	}, nil
}

func normalizarWriteSet(writeSet []string, archivoObjetivo string) ([]string, error) {
	items := normalizarListaTexto(writeSet)
	if archivoObjetivo != "" && !slices.Contains(items, archivoObjetivo) {
		items = append(items, archivoObjetivo)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("el write_set no puede quedar vacío")
	}
	resultado := make([]string, 0, len(items))
	for _, item := range items {
		ruta, err := normalizarRutaRelativa(item)
		if err != nil {
			return nil, fmt.Errorf("write_set inválido: %w", err)
		}
		if !slices.Contains(resultado, ruta) {
			resultado = append(resultado, ruta)
		}
	}
	return resultado, nil
}

func normalizarWriteSetEntregado(writeSet []string) ([]string, error) {
	items := normalizarListaTexto(writeSet)
	if len(items) == 0 {
		return nil, nil
	}
	resultado := make([]string, 0, len(items))
	for _, item := range items {
		ruta, err := normalizarRutaRelativa(item)
		if err != nil {
			return nil, fmt.Errorf("la ruta %q no es válida: %w", item, err)
		}
		if !slices.Contains(resultado, ruta) {
			resultado = append(resultado, ruta)
		}
	}
	return resultado, nil
}

func normalizarTargetBranchGit(baseRef string) string {
	baseRef = strings.TrimSpace(baseRef)
	baseRef = strings.TrimPrefix(baseRef, "refs/heads/")
	baseRef = strings.TrimPrefix(baseRef, "origin/")
	baseRef = strings.TrimPrefix(baseRef, "refs/remotes/")
	baseRef = strings.TrimPrefix(baseRef, "HEAD -> ")
	baseRef = strings.TrimSpace(baseRef)
	switch baseRef {
	case "", "HEAD":
		return ""
	default:
		return baseRef
	}
}

func buildGitMetadataMicroprogramacion(especificacionID int64, captura *EntregaGitCapturada, item *EspecificacionFuncion) string {
	payload := map[string]any{
		"source":            "microprogramacion_git",
		"especificacion_id": especificacionID,
		"archivo_objetivo":  strings.TrimSpace(item.ArchivoObjetivo),
		"simbolo_objetivo":  strings.TrimSpace(item.SimboloObjetivo),
		"write_set":         slices.Clone(item.WriteSet),
		"worktree_id":       captura.WorktreeID,
		"worktree_path":     strings.TrimSpace(captura.RutaWorktree),
		"head_commit":       strings.TrimSpace(captura.HeadCommit),
		"archivos":          slices.Clone(captura.ArchivosModificados),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func normalizarArchivosEntrega(item *EspecificacionFuncion, archivos []ArchivoEntrega) ([]ArchivoEntrega, []string, error) {
	if item == nil {
		return nil, nil, fmt.Errorf("especificacion obligatoria")
	}
	if len(archivos) == 0 {
		return nil, nil, fmt.Errorf("la entrega no contiene ficheros materializables")
	}
	resultado := make([]ArchivoEntrega, 0, len(archivos))
	rutas := make([]string, 0, len(archivos))
	for _, archivo := range archivos {
		ruta, err := normalizarRutaRelativa(archivo.RutaRelativa)
		if err != nil {
			return nil, nil, fmt.Errorf("ruta de entrega invalida: %w", err)
		}
		if !slices.Contains(item.WriteSet, ruta) {
			return nil, nil, fmt.Errorf("el archivo %q queda fuera del write_set permitido", ruta)
		}
		if strings.TrimSpace(archivo.Contenido) == "" {
			return nil, nil, fmt.Errorf("contenido vacio para %q", ruta)
		}
		if !slices.Contains(rutas, ruta) {
			rutas = append(rutas, ruta)
			resultado = append(resultado, ArchivoEntrega{
				RutaRelativa: ruta,
				Contenido:    archivo.Contenido,
			})
		}
	}
	if !slices.Contains(rutas, item.ArchivoObjetivo) {
		return nil, nil, fmt.Errorf("la entrega debe incluir el archivo objetivo %q", item.ArchivoObjetivo)
	}
	return resultado, rutas, nil
}

func normalizarRutaRelativa(valor string) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", fmt.Errorf("ruta vacía")
	}
	if filepath.IsAbs(valor) {
		return "", fmt.Errorf("debe ser relativa al repo")
	}
	limpia := filepath.Clean(valor)
	limpia = filepath.ToSlash(strings.TrimSpace(limpia))
	switch {
	case limpia == "", limpia == ".":
		return "", fmt.Errorf("ruta vacía")
	case limpia == "..", strings.HasPrefix(limpia, "../"):
		return "", fmt.Errorf("no puede salir del repo")
	}
	return limpia, nil
}

func normalizarListaTexto(items []string) []string {
	resultado := make([]string, 0, len(items))
	for _, item := range items {
		normalizado := strings.TrimSpace(item)
		if normalizado == "" || slices.Contains(resultado, normalizado) {
			continue
		}
		resultado = append(resultado, normalizado)
	}
	return resultado
}

func normalizarListaDependencias(items []string) []string {
	resultado := make([]string, 0, len(items))
	for _, item := range items {
		normalizado := strings.ToLower(strings.TrimSpace(item))
		if normalizado == "" || slices.Contains(resultado, normalizado) {
			continue
		}
		resultado = append(resultado, normalizado)
	}
	return resultado
}

func dependenciaSolapada(permitidas, prohibidas []string) string {
	for _, item := range permitidas {
		if dependenciaIncluida(prohibidas, item) {
			return item
		}
	}
	return ""
}

func dependenciaIncluida(candidatas []string, objetivo string) bool {
	objetivo = strings.ToLower(strings.TrimSpace(objetivo))
	if objetivo == "" {
		return false
	}
	for _, candidata := range candidatas {
		candidata = strings.ToLower(strings.TrimSpace(candidata))
		if candidata == "" {
			continue
		}
		if candidata == objetivo {
			return true
		}
		if nombreBaseDependencia(candidata) == nombreBaseDependencia(objetivo) {
			return true
		}
	}
	return false
}

func nombreBaseDependencia(valor string) string {
	valor = strings.ToLower(strings.TrimSpace(valor))
	if valor == "" {
		return ""
	}
	partes := strings.Split(valor, "/")
	return strings.TrimSpace(partes[len(partes)-1])
}

func construirMicrotareaEmitida(item *EspecificacionFuncion, entrada EntradaEmitirMicrotarea) *MicrotareaEmitida {
	partes := []string{
		"MICROTAREA CERRADA",
		fmt.Sprintf("SIMBOLO: %s", item.SimboloObjetivo),
		fmt.Sprintf("ARCHIVO: %s", item.ArchivoObjetivo),
		fmt.Sprintf("OBJETIVO: %s", item.Descripcion),
		"REGLAS: solo este cambio; sin refactors laterales; sin arquitectura; sin archivos fuera del write-set.",
	}
	if len(item.Precondiciones) > 0 {
		partes = append(partes, "PRE: "+strings.Join(item.Precondiciones, " | "))
	}
	if len(item.Postcondiciones) > 0 {
		partes = append(partes, "POST: "+strings.Join(item.Postcondiciones, " | "))
	}
	if len(item.DependenciasPermitidas) > 0 {
		partes = append(partes, "PERMITIDO: "+strings.Join(item.DependenciasPermitidas, ", "))
	}
	if len(item.DependenciasProhibidas) > 0 {
		partes = append(partes, "PROHIBIDO: "+strings.Join(item.DependenciasProhibidas, ", "))
	}
	partes = append(partes,
		"WRITE_SET: "+strings.Join(item.WriteSet, ", "),
		"TESTS: "+strings.Join(item.TestsObligatorios, " | "),
		"SALIDA: "+item.FormatoSalida,
		"SI_BLOQUEO: responde solo 'BLOQUEO: <motivo concreto>'.",
	)
	if FormatoSalidaUsaGitWorktree(item.FormatoSalida) {
		partes = append(partes,
			"ENTREGA_GIT: trabaja dentro de tu worktree activa; no pegues codigo ni diff en la respuesta.",
			"RESPUESTA_ESPERADA: si no hay bloqueo, responde solo con resumen breve, tests ejecutados y estado del diff/branch para que Orquesta recoja la entrega por git.",
		)
	}
	if contexto := strings.TrimSpace(entrada.Contexto); contexto != "" {
		partes = append(partes, "CONTEXTO: "+contexto)
	}
	return &MicrotareaEmitida{
		EspecificacionID:  idOZero(item),
		Titulo:            item.Titulo,
		ArchivoObjetivo:   item.ArchivoObjetivo,
		SimboloObjetivo:   item.SimboloObjetivo,
		WriteSet:          slices.Clone(item.WriteSet),
		TestsObligatorios: slices.Clone(item.TestsObligatorios),
		FormatoSalida:     item.FormatoSalida,
		Mensaje:           strings.Join(partes, "\n"),
	}
}

func FormatoSalidaUsaGitWorktree(formato string) bool {
	formato = strings.TrimSpace(strings.ToLower(formato))
	return strings.Contains(formato, "git") || strings.Contains(formato, "worktree")
}

func idOZero(item *EspecificacionFuncion) int64 {
	if item == nil {
		return 0
	}
	return item.ID
}
