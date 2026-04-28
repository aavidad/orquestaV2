package ollamapool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Mensaje struct {
	Rol       string `json:"role"`
	Contenido string `json:"content"`
}

type Sesion struct {
	ID                  string    `json:"id"`
	HandleRef           string    `json:"handle_ref"`
	Agente              string    `json:"agente"`
	Proyecto            string    `json:"proyecto"`
	PoolSlug            string    `json:"pool_slug,omitempty"`
	Modelo              string    `json:"modelo"`
	PerfilTarea         string    `json:"perfil_tarea,omitempty"`
	Razonamiento        string    `json:"razonamiento,omitempty"`
	WorktreeID          int64     `json:"worktree_id,omitempty"`
	RutaWorktree        string    `json:"ruta_worktree,omitempty"`
	BranchWorktree      string    `json:"branch_worktree,omitempty"`
	BaseRefWorktree     string    `json:"base_ref_worktree,omitempty"`
	MaxMensajesContexto int       `json:"max_mensajes_contexto,omitempty"`
	Estado              string    `json:"estado"`
	ErrorUltimo         string    `json:"error_ultimo,omitempty"`
	ResumenContinuidad  string    `json:"resumen_continuidad,omitempty"`
	CreadoEn            time.Time `json:"creado_en"`
	ActualizadoEn       time.Time `json:"actualizado_en"`
	Mensajes            []Mensaje `json:"mensajes,omitempty"`
}

type EntradaLanzamiento struct {
	Agente              string
	Proyecto            string
	PoolSlug            string
	SlotsMaximos        int
	Modelo              string
	PerfilTarea         string
	Razonamiento        string
	Sistema             string
	ResumenContinuidad  string
	MaxMensajesContexto int
}

type ResultadoEntrada struct {
	Sesion    *Sesion `json:"sesion"`
	Respuesta string  `json:"respuesta"`
}

type CallbackResultadoEntrada func(*ResultadoEntrada, error)

type TelemetriaPool struct {
	PoolSlug               string    `json:"pool_slug"`
	SlotsActivos           int       `json:"slots_activos"`
	SesionesLogicasActivas int       `json:"sesiones_logicas_activas"`
	SesionesReady          int       `json:"sesiones_ready"`
	SesionesWorking        int       `json:"sesiones_working"`
	SesionesFailed         int       `json:"sesiones_failed"`
	ActualizadoEn          time.Time `json:"actualizado_en"`
}

type Gestor struct {
	mu       sync.Mutex
	endpoint string
	client   *http.Client
	nextID   int64
	sesiones map[string]*Sesion
}

// TimeoutClienteDefecto es el timeout HTTP por defecto para llamadas a Ollama.
// El bootstrap de cmd/ lo sobreescribe via ORQUESTA_OLLAMA_TIMEOUT_MS si está definida.
const TimeoutClienteDefecto = 10 * time.Minute

func NuevoGestor(endpoint string, client *http.Client) *Gestor {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if client == nil {
		client = &http.Client{Timeout: TimeoutClienteDefecto}
	}
	return &Gestor{
		endpoint: endpoint,
		client:   client,
		sesiones: map[string]*Sesion{},
	}
}

func (g *Gestor) Lanzar(ctx context.Context, in EntradaLanzamiento) (*Sesion, error) {
	_ = ctx
	modelo := strings.TrimSpace(in.Modelo)
	if modelo == "" {
		return nil, fmt.Errorf("modelo obligatorio")
	}
	agente := strings.TrimSpace(in.Agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	poolSlug := strings.TrimSpace(in.PoolSlug)
	if poolSlug == "" {
		return nil, fmt.Errorf("pool_slug obligatorio")
	}
	slotsMaximos := in.SlotsMaximos
	if slotsMaximos <= 0 {
		slotsMaximos = 1
	}
	ahora := time.Now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()
	if conflicto := g.modeloActivoDistintoBloqueado(modelo, ""); conflicto != "" {
		return nil, fmt.Errorf("modelo ollama activo incompatible: %s; detenerlo antes de lanzar %s", conflicto, modelo)
	}
	if g.slotsActivos(poolSlug) >= slotsMaximos {
		return nil, fmt.Errorf("pool %q sin slots libres", poolSlug)
	}
	g.nextID++
	id := fmt.Sprintf(
		"ollama-pool-%s-%d-%d",
		slugPoolLogico(agente),
		ahora.UnixNano(),
		g.nextID,
	)
	sesion := &Sesion{
		ID:                  id,
		HandleRef:           id,
		Agente:              agente,
		Proyecto:            strings.TrimSpace(in.Proyecto),
		PoolSlug:            poolSlug,
		Modelo:              modelo,
		PerfilTarea:         strings.TrimSpace(in.PerfilTarea),
		Razonamiento:        strings.TrimSpace(in.Razonamiento),
		MaxMensajesContexto: normalizarMaxMensajesContexto(in.MaxMensajesContexto),
		ResumenContinuidad:  strings.TrimSpace(in.ResumenContinuidad),
		Estado:              "ready",
		CreadoEn:            ahora,
		ActualizadoEn:       ahora,
	}
	if sistema := strings.TrimSpace(in.Sistema); sistema != "" {
		sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "system", Contenido: sistema})
	}
	if resumen := strings.TrimSpace(in.ResumenContinuidad); resumen != "" {
		sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "system", Contenido: "CONTINUIDAD BREVE: " + resumen})
	}
	compactarSesion(sesion, sesion.MaxMensajesContexto)
	g.sesiones[id] = sesion
	return clonSesion(sesion), nil
}

func slugPoolLogico(valor string) string {
	valor = strings.ToLower(strings.TrimSpace(valor))
	if valor == "" {
		return "sesion"
	}
	var out strings.Builder
	for _, r := range valor {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		default:
			out.WriteByte('-')
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug == "" {
		return "sesion"
	}
	return slug
}

func (g *Gestor) slotsActivos(poolSlug string) int {
	total := 0
	for _, sesion := range g.sesiones {
		if sesion == nil || strings.TrimSpace(sesion.PoolSlug) != strings.TrimSpace(poolSlug) {
			continue
		}
		switch sesion.Estado {
		case "stopped", "failed":
			continue
		default:
			total++
		}
	}
	return total
}

func (g *Gestor) Estado(handleRef string) (*Sesion, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	sesion, ok := g.sesiones[strings.TrimSpace(handleRef)]
	if !ok {
		return nil, fmt.Errorf("sesion %q no encontrada", handleRef)
	}
	return clonSesion(sesion), nil
}

func (g *Gestor) RegistrarSesion(sesion *Sesion) (*Sesion, error) {
	if g == nil {
		return nil, fmt.Errorf("gestor de pool no disponible")
	}
	if sesion == nil {
		return nil, fmt.Errorf("sesion obligatoria")
	}
	handleRef := strings.TrimSpace(sesion.HandleRef)
	if handleRef == "" {
		handleRef = strings.TrimSpace(sesion.ID)
	}
	if handleRef == "" {
		return nil, fmt.Errorf("handle_ref obligatorio")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if existente := g.sesiones[handleRef]; existente != nil {
		return clonSesion(existente), nil
	}
	copia := clonSesion(sesion)
	copia.HandleRef = handleRef
	if strings.TrimSpace(copia.ID) == "" {
		copia.ID = handleRef
	}
	if copia.CreadoEn.IsZero() {
		copia.CreadoEn = time.Now().UTC()
	}
	if copia.ActualizadoEn.IsZero() {
		copia.ActualizadoEn = copia.CreadoEn
	}
	if copia.MaxMensajesContexto <= 0 {
		copia.MaxMensajesContexto = normalizarMaxMensajesContexto(copia.MaxMensajesContexto)
	}
	copia.Estado = strings.TrimSpace(copia.Estado)
	if copia.Estado == "" {
		copia.Estado = "ready"
	}
	if resumen := strings.TrimSpace(copia.ResumenContinuidad); resumen != "" && len(copia.Mensajes) == 0 {
		copia.Mensajes = append(copia.Mensajes, Mensaje{Rol: "system", Contenido: "CONTINUIDAD BREVE: " + resumen})
	}
	g.sesiones[handleRef] = copia
	return clonSesion(copia), nil
}

func (g *Gestor) Detener(handleRef string) error {
	handleRef = strings.TrimSpace(handleRef)
	if handleRef == "" {
		return fmt.Errorf("handle_ref obligatorio")
	}
	g.mu.Lock()
	sesion, ok := g.sesiones[handleRef]
	if !ok {
		g.mu.Unlock()
		return fmt.Errorf("sesion %q no encontrada", handleRef)
	}
	sesion.Estado = "stopped"
	sesion.ActualizadoEn = time.Now().UTC()
	modelo := strings.TrimSpace(sesion.Modelo)
	descargarModelo := modelo != "" && !g.modeloTieneSesionesActivasBloqueado(modelo, handleRef)
	g.mu.Unlock()
	if descargarModelo {
		g.solicitarDescargaModelo(context.Background(), modelo)
	}
	return nil
}

func (g *Gestor) modeloTieneSesionesActivasBloqueado(modelo, exceptHandleRef string) bool {
	modelo = strings.TrimSpace(modelo)
	exceptHandleRef = strings.TrimSpace(exceptHandleRef)
	if modelo == "" {
		return false
	}
	for handleRef, sesion := range g.sesiones {
		if sesion == nil || strings.TrimSpace(handleRef) == exceptHandleRef {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(sesion.Modelo), modelo) {
			continue
		}
		switch strings.TrimSpace(sesion.Estado) {
		case "", "ready", "working":
			return true
		}
	}
	return false
}

func (g *Gestor) modeloActivoDistintoBloqueado(modelo, exceptHandleRef string) string {
	modelo = strings.TrimSpace(modelo)
	exceptHandleRef = strings.TrimSpace(exceptHandleRef)
	if modelo == "" {
		return ""
	}
	for handleRef, sesion := range g.sesiones {
		if sesion == nil || strings.TrimSpace(handleRef) == exceptHandleRef {
			continue
		}
		modeloSesion := strings.TrimSpace(sesion.Modelo)
		if modeloSesion == "" || strings.EqualFold(modeloSesion, modelo) {
			continue
		}
		switch strings.TrimSpace(sesion.Estado) {
		case "stopped", "failed":
			continue
		default:
			return modeloSesion
		}
	}
	return ""
}

func (g *Gestor) DescribirPool(poolSlug string) *TelemetriaPool {
	poolSlug = strings.TrimSpace(poolSlug)
	if poolSlug == "" {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	var out TelemetriaPool
	out.PoolSlug = poolSlug
	for _, sesion := range g.sesiones {
		if sesion == nil || strings.TrimSpace(sesion.PoolSlug) != poolSlug {
			continue
		}
		if sesion.ActualizadoEn.After(out.ActualizadoEn) {
			out.ActualizadoEn = sesion.ActualizadoEn
		}
		switch sesion.Estado {
		case "stopped":
			continue
		case "failed":
			out.SesionesFailed++
			out.SesionesLogicasActivas++
		case "working":
			out.SesionesWorking++
			out.SesionesLogicasActivas++
			out.SlotsActivos++
		case "ready":
			out.SesionesReady++
			out.SesionesLogicasActivas++
			out.SlotsActivos++
		default:
			out.SesionesLogicasActivas++
			out.SlotsActivos++
		}
	}
	return &out
}

func (g *Gestor) Enviar(ctx context.Context, handleRef, texto string) (*ResultadoEntrada, error) {
	handleRef = strings.TrimSpace(handleRef)
	texto = strings.TrimSpace(texto)
	if handleRef == "" {
		return nil, fmt.Errorf("handle_ref obligatorio")
	}
	if texto == "" {
		return nil, fmt.Errorf("texto obligatorio")
	}

	g.mu.Lock()
	sesion, ok := g.sesiones[handleRef]
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("sesion %q no encontrada", handleRef)
	}
	if sesion.Estado == "stopped" {
		g.mu.Unlock()
		return nil, fmt.Errorf("sesion %q detenida", handleRef)
	}
	sesion.Estado = "working"
	sesion.ErrorUltimo = ""
	sesion.ActualizadoEn = time.Now().UTC()
	sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "user", Contenido: texto})
	modelo := sesion.Modelo
	mensajes := append([]Mensaje(nil), sesion.Mensajes...)
	g.mu.Unlock()

	respuesta, err := g.chat(ctx, modelo, mensajes)

	g.mu.Lock()
	defer g.mu.Unlock()
	sesion = g.sesiones[handleRef]
	if sesion == nil {
		return nil, fmt.Errorf("sesion %q no encontrada tras enviar", handleRef)
	}
	sesion.ActualizadoEn = time.Now().UTC()
	if err != nil {
		sesion.Estado = "failed"
		sesion.ErrorUltimo = err.Error()
		return nil, err
	}
	sesion.Estado = "ready"
	sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "assistant", Contenido: respuesta})
	compactarSesion(sesion, sesion.MaxMensajesContexto)
	return &ResultadoEntrada{
		Sesion:    clonSesion(sesion),
		Respuesta: respuesta,
	}, nil
}

func (g *Gestor) EnviarAsincrono(ctx context.Context, handleRef, texto string, callback CallbackResultadoEntrada) (*Sesion, error) {
	handleRef = strings.TrimSpace(handleRef)
	texto = strings.TrimSpace(texto)
	if handleRef == "" {
		return nil, fmt.Errorf("handle_ref obligatorio")
	}
	if texto == "" {
		return nil, fmt.Errorf("texto obligatorio")
	}

	g.mu.Lock()
	sesion, ok := g.sesiones[handleRef]
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("sesion %q no encontrada", handleRef)
	}
	if sesion.Estado == "stopped" {
		g.mu.Unlock()
		return nil, fmt.Errorf("sesion %q detenida", handleRef)
	}
	sesion.Estado = "working"
	sesion.ErrorUltimo = ""
	sesion.ActualizadoEn = time.Now().UTC()
	sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "user", Contenido: texto})
	modelo := sesion.Modelo
	mensajes := append([]Mensaje(nil), sesion.Mensajes...)
	snapshot := clonSesion(sesion)
	g.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		respuesta, err := g.chat(ctx, modelo, mensajes)

		g.mu.Lock()
		sesion := g.sesiones[handleRef]
		if sesion == nil {
			g.mu.Unlock()
			if callback != nil {
				callback(nil, fmt.Errorf("sesion %q no encontrada tras enviar", handleRef))
			}
			return
		}
		sesion.ActualizadoEn = time.Now().UTC()
		if err != nil {
			sesion.Estado = "failed"
			sesion.ErrorUltimo = err.Error()
			resultado := &ResultadoEntrada{Sesion: clonSesion(sesion)}
			g.mu.Unlock()
			if callback != nil {
				callback(resultado, err)
			}
			return
		}
		sesion.Estado = "ready"
		sesion.Mensajes = append(sesion.Mensajes, Mensaje{Rol: "assistant", Contenido: respuesta})
		compactarSesion(sesion, sesion.MaxMensajesContexto)
		resultado := &ResultadoEntrada{
			Sesion:    clonSesion(sesion),
			Respuesta: respuesta,
		}
		g.mu.Unlock()
		if callback != nil {
			callback(resultado, nil)
		}
	}()

	return snapshot, nil
}

func (g *Gestor) chat(ctx context.Context, modelo string, mensajes []Mensaje) (string, error) {
	if g == nil || strings.TrimSpace(g.endpoint) == "" {
		return "", fmt.Errorf("endpoint de ollama no configurado")
	}
	payload := map[string]any{
		"model":    strings.TrimSpace(modelo),
		"messages": mensajes,
		"stream":   false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama devolvio status %d", resp.StatusCode)
	}
	var decoded struct {
		Message struct {
			Role     string `json:"role"`
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	return resolverContenidoRespuestaOllama(decoded.Message.Content, decoded.Message.Thinking), nil
}

func (g *Gestor) solicitarDescargaModelo(ctx context.Context, modelo string) {
	modelo = strings.TrimSpace(modelo)
	if g == nil || modelo == "" || strings.TrimSpace(g.endpoint) == "" || g.client == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload := map[string]any{
		"model":      modelo,
		"prompt":     "",
		"stream":     false,
		"keep_alive": 0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func resolverContenidoRespuestaOllama(contenido, thinking string) string {
	contenido = strings.TrimSpace(contenido)
	if contenido != "" {
		return contenido
	}
	thinking = strings.TrimSpace(thinking)
	if thinking == "" {
		return ""
	}
	thinkingNormalizado := strings.ToLower(strings.TrimSpace(thinking))
	if strings.Contains(thinkingNormalizado, "patch_unificado") || strings.Contains(thinkingNormalizado, "bloqueo:") {
		return thinking
	}
	return ""
}

func clonSesion(in *Sesion) *Sesion {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Mensajes) > 0 {
		out.Mensajes = append([]Mensaje(nil), in.Mensajes...)
	}
	return &out
}

const (
	maxMensajesContextoPoolDefault = 6
	maxResumenContinuidadChars     = 600
)

func normalizarMaxMensajesContexto(maxMensajes int) int {
	if maxMensajes <= 0 {
		return maxMensajesContextoPoolDefault
	}
	return maxMensajes
}

func compactarSesion(sesion *Sesion, maxMensajes int) {
	if sesion == nil {
		return
	}
	if maxMensajes <= 0 {
		maxMensajes = maxMensajesContextoPoolDefault
	}
	sistemas := make([]Mensaje, 0, len(sesion.Mensajes))
	dialogo := make([]Mensaje, 0, len(sesion.Mensajes))
	for _, msg := range sesion.Mensajes {
		if strings.EqualFold(strings.TrimSpace(msg.Rol), "system") {
			sistemas = append(sistemas, msg)
			continue
		}
		dialogo = append(dialogo, msg)
	}
	if len(dialogo) <= maxMensajes {
		sesion.Mensajes = append(sistemas[:0:0], append(sistemas, dialogo...)...)
		return
	}
	recortados := dialogo[:len(dialogo)-maxMensajes]
	sesion.ResumenContinuidad = resumirMensajes(recortados, sesion.ResumenContinuidad)
	resumenSistema := Mensaje{Rol: "system", Contenido: "CONTINUIDAD BREVE: " + sesion.ResumenContinuidad}
	sistemas = filtrarSistemasSinContinuidad(sistemas)
	if strings.TrimSpace(sesion.ResumenContinuidad) != "" {
		sistemas = append(sistemas, resumenSistema)
	}
	dialogo = dialogo[len(dialogo)-maxMensajes:]
	sesion.Mensajes = append(sistemas[:0:0], append(sistemas, dialogo...)...)
}

func filtrarSistemasSinContinuidad(items []Mensaje) []Mensaje {
	out := make([]Mensaje, 0, len(items))
	for _, msg := range items {
		if strings.HasPrefix(strings.TrimSpace(msg.Contenido), "CONTINUIDAD BREVE:") {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func resumirMensajes(items []Mensaje, previo string) string {
	partes := make([]string, 0, len(items)+1)
	if base := strings.TrimSpace(previo); base != "" {
		partes = append(partes, base)
	}
	for _, msg := range items {
		texto := strings.TrimSpace(msg.Contenido)
		if texto == "" {
			continue
		}
		rol := strings.ToUpper(strings.TrimSpace(msg.Rol))
		if rol == "" {
			rol = "MSG"
		}
		partes = append(partes, rol+": "+recortarTextoPlano(texto, 120))
	}
	resumen := strings.Join(partes, " | ")
	return recortarTextoPlano(resumen, maxResumenContinuidadChars)
}

func recortarTextoPlano(texto string, max int) string {
	texto = strings.Join(strings.Fields(strings.TrimSpace(texto)), " ")
	if max <= 0 || len(texto) <= max {
		return texto
	}
	if max <= 3 {
		return texto[:max]
	}
	return strings.TrimSpace(texto[:max-3]) + "..."
}
