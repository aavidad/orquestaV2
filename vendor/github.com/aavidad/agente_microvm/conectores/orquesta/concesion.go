package microvm

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	EsquemaPlanLanzamiento      = "agentmicrovm.plan-lanzamiento.v1"
	EsquemaConcesionLanzamiento = "agentmicrovm.concesion-lanzamiento.v1"
	EsquemaConcesionEgreso      = "agentmicrovm.concesion-egreso.v1"
	AudienciaLanzamiento        = "agentmicrovm.lanzamiento.v1"
	AlgoritmoConcesion          = "ed25519"
	VigenciaMaximaConcesion     = 5 * time.Minute
	maximoDestinosEgreso        = 32
	maximoPuertosDestino        = 16
	maximoConexionesEgreso      = 256
	maximoBytesEgreso           = uint64(1 << 30)
)

type ServicioVsock struct {
	Papel           string `json:"papel"`
	ServicioRef     string `json:"servicio_ref"`
	Puerto          uint32 `json:"puerto"`
	IdentidadRef    string `json:"identidad_ref"`
	IdentidadSHA256 string `json:"identidad_sha256"`
}

// AutoridadServiciosHostLanzamientoV1 es el subconjunto firmado que Agente
// MicroVM proyecta después en cada AperturaServicioHostV1. No contiene rutas,
// identidad física ni material criptográfico.
type AutoridadServiciosHostLanzamientoV1 struct {
	RunRef          string
	Cerca           uint64
	PlanSHA256      string
	ConcesionSHA256 string
	Servicios       []ServicioVsock
}

type DestinoEgreso struct {
	Host    string   `json:"host"`
	Puertos []uint16 `json:"puertos"`
}

type ConcesionEgreso struct {
	Esquema           string          `json:"esquema"`
	Referencia        string          `json:"referencia"`
	Destinos          []DestinoEgreso `json:"destinos"`
	MaximoConexiones  uint32          `json:"maximo_conexiones"`
	LimiteTiempoMS    uint64          `json:"limite_tiempo_ms"`
	LimiteSubidaBytes uint64          `json:"limite_subida_bytes"`
	LimiteBajadaBytes uint64          `json:"limite_bajada_bytes"`
}

type PlanLanzamiento struct {
	Esquema              string           `json:"esquema"`
	PlanRef              string           `json:"plan_ref"`
	RunRef               string           `json:"run_ref"`
	Cerca                uint64           `json:"cerca"`
	VCPU                 uint8            `json:"vcpu"`
	MemoriaMiB           uint32           `json:"memoria_mib"`
	KernelSHA256         string           `json:"kernel_sha256"`
	InitramfsSHA256      string           `json:"initramfs_sha256"`
	PerfilSHA256         *string          `json:"perfil_sha256"`
	LimiteTiempoMS       uint64           `json:"limite_tiempo_ms"`
	LimiteRAMPicoBytes   uint64           `json:"limite_ram_pico_bytes"`
	LimiteDiscoPicoBytes uint64           `json:"limite_disco_pico_bytes"`
	LimiteTokensAgente   uint64           `json:"limite_tokens_agente"`
	Servicios            []ServicioVsock  `json:"servicios"`
	Egreso               *ConcesionEgreso `json:"egreso,omitempty"`
}

type ContextoAutorizado struct {
	ProyectoRef          string `json:"proyecto_ref"`
	GoalRef              string `json:"goal_ref"`
	WorkItemRef          string `json:"work_item_ref"`
	EjecucionRef         string `json:"ejecucion_ref"`
	AutorizacionRef      string `json:"autorizacion_ref"`
	AprobacionEfectoRef  string `json:"aprobacion_efecto_ref"`
	IntentoEfectoRef     string `json:"intento_efecto_ref"`
	SesionRef            string `json:"sesion_ref"`
	AccesoArtefactosRef  string `json:"acceso_artefactos_ref"`
	AccesoMCPRef         string `json:"acceso_mcp_ref"`
	EndpointBuzonRef     string `json:"endpoint_buzon_ref"`
	DescriptorPerfilRef  string `json:"descriptor_perfil_ref"`
	EspecificacionSHA256 string `json:"especificacion_sha256"`
}

type contenidoConcesion struct {
	Esquema       string `json:"esquema"`
	Audiencia     string `json:"audiencia"`
	ConcesionRef  string `json:"concesion_ref"`
	RunRef        string `json:"run_ref"`
	Cerca         uint64 `json:"cerca"`
	PlanSHA256    string `json:"plan_sha256"`
	EmitidaUnixMS int64  `json:"emitida_unix_ms"`
	NoAntesUnixMS int64  `json:"no_antes_unix_ms"`
	ExpiraUnixMS  int64  `json:"expira_unix_ms"`
	ClaveID       string `json:"clave_id"`
	Algoritmo     string `json:"algoritmo"`
}

type concesionLanzamientoFirmadaV1 struct {
	Contenido   contenidoConcesion `json:"contenido"`
	FirmaBase64 string             `json:"firma_base64"`
}

var esquemaServicioVsockLanzamientoV1 = esquemaObjetoJSONEstricto{
	"papel": esquemaEscalarJSONEstricto, "servicio_ref": esquemaEscalarJSONEstricto,
	"puerto": esquemaEscalarJSONEstricto, "identidad_ref": esquemaEscalarJSONEstricto,
	"identidad_sha256": esquemaEscalarJSONEstricto,
}

var esquemaPlanLanzamientoSinEgresoV1 = esquemaObjetoJSONEstricto{
	"esquema": esquemaEscalarJSONEstricto, "plan_ref": esquemaEscalarJSONEstricto,
	"run_ref": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"vcpu": esquemaEscalarJSONEstricto, "memoria_mib": esquemaEscalarJSONEstricto,
	"kernel_sha256": esquemaEscalarJSONEstricto, "initramfs_sha256": esquemaEscalarJSONEstricto,
	"perfil_sha256":    esquemaEscalarJSONEstrictoOpcional,
	"limite_tiempo_ms": esquemaEscalarJSONEstricto, "limite_ram_pico_bytes": esquemaEscalarJSONEstricto,
	"limite_disco_pico_bytes": esquemaEscalarJSONEstricto, "limite_tokens_agente": esquemaEscalarJSONEstricto,
	"servicios": esquemaArrayEstricto(esquemaObjetoEstricto(esquemaServicioVsockLanzamientoV1)),
}

var esquemaPlanLanzamientoConEgresoV1 = func() esquemaObjetoJSONEstricto {
	esquema := make(esquemaObjetoJSONEstricto, len(esquemaPlanLanzamientoSinEgresoV1)+1)
	for nombre, campo := range esquemaPlanLanzamientoSinEgresoV1 {
		esquema[nombre] = campo
	}
	esquema["egreso"] = esquemaObjetoEstricto(esquemaConcesionEgresoV1)
	return esquema
}()

var esquemaContenidoConcesionLanzamientoV1 = esquemaObjetoJSONEstricto{
	"esquema": esquemaEscalarJSONEstricto, "audiencia": esquemaEscalarJSONEstricto,
	"concesion_ref": esquemaEscalarJSONEstricto, "run_ref": esquemaEscalarJSONEstricto,
	"cerca": esquemaEscalarJSONEstricto, "plan_sha256": esquemaEscalarJSONEstricto,
	"emitida_unix_ms": esquemaEscalarJSONEstricto, "no_antes_unix_ms": esquemaEscalarJSONEstricto,
	"expira_unix_ms": esquemaEscalarJSONEstricto, "clave_id": esquemaEscalarJSONEstricto,
	"algoritmo": esquemaEscalarJSONEstricto,
}

var esquemaConcesionLanzamientoFirmadaV1 = esquemaObjetoJSONEstricto{
	"contenido":    esquemaObjetoEstricto(esquemaContenidoConcesionLanzamientoV1),
	"firma_base64": esquemaEscalarJSONEstricto,
}

type ErrorConcesion struct{ Codigo string }

func (e *ErrorConcesion) Error() string { return e.Codigo }

// ExtraerAutoridadServiciosHostLanzamientoV1 deriva únicamente la autoridad
// canónica de una SolicitudLanzamiento producida por FirmanteConcesiones. No
// autentica la firma Ed25519 ni decide vigencia; esas decisiones pertenecen al
// verificador Rust antes del lanzamiento físico.
func ExtraerAutoridadServiciosHostLanzamientoV1(
	solicitud SolicitudLanzamiento,
) (AutoridadServiciosHostLanzamientoV1, error) {
	var plan PlanLanzamiento
	if !decodificarObjetoJSONEstricto(solicitud.Plan, esquemaPlanLanzamientoSinEgresoV1, &plan) &&
		!decodificarObjetoJSONEstricto(solicitud.Plan, esquemaPlanLanzamientoConEgresoV1, &plan) {
		return AutoridadServiciosHostLanzamientoV1{}, errorAutoridadServiciosHost("solicitud_invalida")
	}
	mensajePlanCanonico, err := mensajePlan(plan)
	if err != nil {
		return AutoridadServiciosHostLanzamientoV1{}, err
	}
	planDigest := sha256.Sum256(mensajePlanCanonico)
	planSHA256 := hex.EncodeToString(planDigest[:])

	var firmada concesionLanzamientoFirmadaV1
	if !decodificarObjetoJSONEstricto(
		solicitud.Concesion,
		esquemaConcesionLanzamientoFirmadaV1,
		&firmada,
	) || !contenidoConcesionValido(firmada.Contenido) {
		return AutoridadServiciosHostLanzamientoV1{}, errorAutoridadServiciosHost("concesion_invalida")
	}
	firma, err := base64.StdEncoding.DecodeString(firmada.FirmaBase64)
	if err != nil || len(firma) != ed25519.SignatureSize || base64.StdEncoding.EncodeToString(firma) != firmada.FirmaBase64 {
		return AutoridadServiciosHostLanzamientoV1{}, errorAutoridadServiciosHost("concesion_invalida")
	}
	contenido := firmada.Contenido
	if contenido.RunRef != plan.RunRef || contenido.Cerca != plan.Cerca || contenido.PlanSHA256 != planSHA256 {
		return AutoridadServiciosHostLanzamientoV1{}, errorAutoridadServiciosHost("vinculo_invalido")
	}
	mensajeConcesionCanonico := mensajeConcesion(contenido)
	concesionDigest := sha256.Sum256(mensajeConcesionCanonico)
	servicios := append([]ServicioVsock(nil), plan.Servicios...)
	sort.Slice(servicios, func(i, j int) bool { return servicios[i].Papel < servicios[j].Papel })
	if !serviciosAutoridadHostValidos(servicios) {
		return AutoridadServiciosHostLanzamientoV1{}, errorAutoridadServiciosHost("servicios_invalidos")
	}
	return AutoridadServiciosHostLanzamientoV1{
		RunRef: plan.RunRef, Cerca: plan.Cerca,
		PlanSHA256: planSHA256, ConcesionSHA256: hex.EncodeToString(concesionDigest[:]),
		Servicios: servicios,
	}, nil
}

func serviciosAutoridadHostValidos(servicios []ServicioVsock) bool {
	if len(servicios) < 1 || len(servicios) > 2 || servicios[0].Papel != "control_broker" ||
		(len(servicios) == 2 && servicios[1].Papel != "controlled_egress_proxy") {
		return false
	}
	vistos := make(map[string]struct{}, len(servicios))
	for _, servicio := range servicios {
		if _, duplicado := vistos[servicio.ServicioRef]; duplicado {
			return false
		}
		vistos[servicio.ServicioRef] = struct{}{}
	}
	return true
}

func contenidoConcesionValido(contenido contenidoConcesion) bool {
	return contenido.Esquema == EsquemaConcesionLanzamiento &&
		contenido.Audiencia == AudienciaLanzamiento &&
		referenciaValida(contenido.ConcesionRef, "concesion:", 160) &&
		referenciaExternaValida(contenido.RunRef) && contenido.Cerca != 0 &&
		sha256Valido(contenido.PlanSHA256) && contenido.EmitidaUnixMS > 0 &&
		contenido.NoAntesUnixMS >= contenido.EmitidaUnixMS &&
		contenido.ExpiraUnixMS > contenido.NoAntesUnixMS &&
		contenido.ExpiraUnixMS-contenido.EmitidaUnixMS <= VigenciaMaximaConcesion.Milliseconds() &&
		referenciaValida(contenido.ClaveID, "clave-publica:", 160) &&
		contenido.Algoritmo == AlgoritmoConcesion
}

func errorAutoridadServiciosHost(sufijo string) error {
	return &ErrorConcesion{Codigo: "concesion.autoridad_servicios_host_" + sufijo}
}

type FirmanteConcesiones struct {
	claveID string
	estado  *estadoFirmanteConcesiones
}

type estadoFirmanteConcesiones struct {
	mu      sync.RWMutex
	privada ed25519.PrivateKey
}

func NuevoFirmanteConcesiones(claveID string, privada ed25519.PrivateKey) (*FirmanteConcesiones, error) {
	if !referenciaValida(claveID, "clave-publica:", 160) || len(privada) != ed25519.PrivateKeySize {
		return nil, &ErrorConcesion{Codigo: "concesion.configuracion_invalida"}
	}
	copia := append(ed25519.PrivateKey(nil), privada...)
	return &FirmanteConcesiones{
		claveID: claveID,
		estado:  &estadoFirmanteConcesiones{privada: copia},
	}, nil
}

// Destruir borra la clave privada retenida por el firmante. Es idempotente y,
// cuando retorna, ninguna preparación concurrente conserva acceso a la clave.
func (f *FirmanteConcesiones) Destruir() {
	if f == nil || f.estado == nil {
		return
	}
	f.estado.mu.Lock()
	defer f.estado.mu.Unlock()
	for indice := range f.estado.privada {
		f.estado.privada[indice] = 0
	}
	f.estado.privada = nil
}

func (f *FirmanteConcesiones) Preparar(contexto ContextoAutorizado, plan PlanLanzamiento, emitida time.Time, vigencia time.Duration) (SolicitudLanzamiento, error) {
	if f == nil || f.estado == nil {
		return SolicitudLanzamiento{}, &ErrorConcesion{Codigo: "concesion.firmante_destruido"}
	}
	f.estado.mu.RLock()
	defer f.estado.mu.RUnlock()
	if len(f.estado.privada) != ed25519.PrivateKeySize {
		return SolicitudLanzamiento{}, &ErrorConcesion{Codigo: "concesion.firmante_destruido"}
	}
	contextoJSON, err := json.Marshal(contexto)
	if err != nil || !contextoValido(contexto) || emitida.UnixMilli() <= 0 || vigencia <= 0 || vigencia > VigenciaMaximaConcesion {
		return SolicitudLanzamiento{}, &ErrorConcesion{Codigo: "concesion.alcance_invalido"}
	}
	contextoSHA := sha256.Sum256(contextoJSON)
	sufijo := hex.EncodeToString(contextoSHA[:])
	plan.Esquema, plan.PlanRef, plan.RunRef = EsquemaPlanLanzamiento, "plan:"+sufijo, contexto.EjecucionRef
	mensajePlan, err := mensajePlan(plan)
	if err != nil {
		return SolicitudLanzamiento{}, err
	}
	planSHA := sha256.Sum256(mensajePlan)
	identidadConcesion := sha256.New()
	identidadConcesion.Write(contextoSHA[:])
	identidadConcesion.Write(planSHA[:])
	identidadConcesion.Write(binary.BigEndian.AppendUint64(nil, uint64(emitida.UnixMilli())))
	identidadConcesion.Write([]byte(f.claveID))
	concesionSufijo := hex.EncodeToString(identidadConcesion.Sum(nil))
	contenido := contenidoConcesion{
		Esquema: EsquemaConcesionLanzamiento, Audiencia: AudienciaLanzamiento,
		ConcesionRef: "concesion:" + concesionSufijo, RunRef: plan.RunRef, Cerca: plan.Cerca,
		PlanSHA256: hex.EncodeToString(planSHA[:]), EmitidaUnixMS: emitida.UnixMilli(),
		NoAntesUnixMS: emitida.UnixMilli(), ExpiraUnixMS: emitida.Add(vigencia).UnixMilli(),
		ClaveID: f.claveID, Algoritmo: AlgoritmoConcesion,
	}
	firma := ed25519.Sign(f.estado.privada, mensajeConcesion(contenido))
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return SolicitudLanzamiento{}, &ErrorConcesion{Codigo: "concesion.codificacion_fallida"}
	}
	concesionJSON, err := json.Marshal(struct {
		Contenido   contenidoConcesion `json:"contenido"`
		FirmaBase64 string             `json:"firma_base64"`
	}{contenido, base64.StdEncoding.EncodeToString(firma)})
	if err != nil {
		return SolicitudLanzamiento{}, &ErrorConcesion{Codigo: "concesion.codificacion_fallida"}
	}
	return SolicitudLanzamiento{Plan: planJSON, Concesion: concesionJSON}, nil
}

func mensajePlan(plan PlanLanzamiento) ([]byte, error) {
	if !planValido(plan) {
		return nil, &ErrorConcesion{Codigo: "concesion.plan_invalido"}
	}
	b := append([]byte("agentmicrovm.plan-lanzamiento.v1\x00"), campo(plan.Esquema)...)
	b = append(b, campo(plan.PlanRef)...)
	b = append(b, campo(plan.RunRef)...)
	b = binary.BigEndian.AppendUint64(b, plan.Cerca)
	b = append(b, plan.VCPU)
	b = binary.BigEndian.AppendUint32(b, plan.MemoriaMiB)
	b = append(b, campo(plan.KernelSHA256)...)
	b = append(b, campo(plan.InitramfsSHA256)...)
	if plan.PerfilSHA256 == nil {
		b = append(b, 0)
	} else {
		b = append(b, 1)
		b = append(b, campo(*plan.PerfilSHA256)...)
	}
	b = binary.BigEndian.AppendUint64(b, plan.LimiteTiempoMS)
	b = binary.BigEndian.AppendUint64(b, plan.LimiteRAMPicoBytes)
	b = binary.BigEndian.AppendUint64(b, plan.LimiteDiscoPicoBytes)
	b = binary.BigEndian.AppendUint64(b, plan.LimiteTokensAgente)
	servicios := append([]ServicioVsock(nil), plan.Servicios...)
	sort.Slice(servicios, func(i, j int) bool { return servicios[i].Papel < servicios[j].Papel })
	b = append(b, byte(len(servicios)))
	for _, s := range servicios {
		b = append(b, campo(s.Papel)...)
		b = append(b, campo(s.ServicioRef)...)
		b = binary.BigEndian.AppendUint32(b, s.Puerto)
		b = append(b, campo(s.IdentidadRef)...)
		b = append(b, campo(s.IdentidadSHA256)...)
	}
	if plan.Egreso != nil {
		b = append(b, 1)
		b = append(b, campo(plan.Egreso.Esquema)...)
		b = append(b, campo(plan.Egreso.Referencia)...)
		b = binary.BigEndian.AppendUint32(b, plan.Egreso.MaximoConexiones)
		b = binary.BigEndian.AppendUint64(b, plan.Egreso.LimiteTiempoMS)
		b = binary.BigEndian.AppendUint64(b, plan.Egreso.LimiteSubidaBytes)
		b = binary.BigEndian.AppendUint64(b, plan.Egreso.LimiteBajadaBytes)
		destinos := append([]DestinoEgreso(nil), plan.Egreso.Destinos...)
		sort.Slice(destinos, func(i, j int) bool { return destinos[i].Host < destinos[j].Host })
		b = append(b, byte(len(destinos)))
		for _, destino := range destinos {
			b = append(b, campo(destino.Host)...)
			puertos := append([]uint16(nil), destino.Puertos...)
			sort.Slice(puertos, func(i, j int) bool { return puertos[i] < puertos[j] })
			b = append(b, byte(len(puertos)))
			for _, puerto := range puertos {
				b = binary.BigEndian.AppendUint16(b, puerto)
			}
		}
	}
	return b, nil
}

func mensajeConcesion(c contenidoConcesion) []byte {
	b := append([]byte("agentmicrovm.concesion-lanzamiento.v1\x00"), campo(c.Esquema)...)
	b = append(b, campo(c.Audiencia)...)
	b = append(b, campo(c.ConcesionRef)...)
	b = append(b, campo(c.RunRef)...)
	b = binary.BigEndian.AppendUint64(b, c.Cerca)
	b = append(b, campo(c.PlanSHA256)...)
	b = binary.BigEndian.AppendUint64(b, uint64(c.EmitidaUnixMS))
	b = binary.BigEndian.AppendUint64(b, uint64(c.NoAntesUnixMS))
	b = binary.BigEndian.AppendUint64(b, uint64(c.ExpiraUnixMS))
	b = append(b, campo(c.ClaveID)...)
	return append(b, campo(c.Algoritmo)...)
}

func campo(valor string) []byte {
	b := binary.BigEndian.AppendUint32(nil, uint32(len(valor)))
	return append(b, valor...)
}
func sha256Valido(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && strings.ToLower(v) == v
}
func referenciaValida(v, prefijo string, max int) bool {
	s := strings.TrimPrefix(v, prefijo)
	if s == v || s == "" || len(v) > max {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func referenciaExternaValida(v string) bool {
	return v != "" && len(v) <= 512 && strings.TrimSpace(v) == v && !strings.ContainsAny(v, "\r\n\x00")
}
func contextoValido(c ContextoAutorizado) bool {
	for _, v := range []string{
		c.ProyectoRef, c.GoalRef, c.WorkItemRef, c.EjecucionRef,
		c.AutorizacionRef, c.AprobacionEfectoRef, c.IntentoEfectoRef,
		c.SesionRef, c.AccesoArtefactosRef, c.AccesoMCPRef, c.EndpointBuzonRef,
	} {
		if v == "" || strings.TrimSpace(v) != v || strings.ContainsAny(v, "\r\n\x00") {
			return false
		}
	}
	descriptorSHA := strings.TrimPrefix(c.DescriptorPerfilRef, prefijoDescriptorPerfilLanzamientoV1)
	return descriptorSHA != c.DescriptorPerfilRef && sha256Valido(descriptorSHA) &&
		sha256Valido(c.EspecificacionSHA256)
}
func planValido(p PlanLanzamiento) bool {
	if p.Esquema != EsquemaPlanLanzamiento || !referenciaValida(p.PlanRef, "plan:", 160) || !referenciaExternaValida(p.RunRef) || p.Cerca == 0 || p.VCPU < 1 || p.VCPU > 32 || p.MemoriaMiB < 64 || p.MemoriaMiB > 32768 || !sha256Valido(p.KernelSHA256) || !sha256Valido(p.InitramfsSHA256) || p.PerfilSHA256 != nil && !sha256Valido(*p.PerfilSHA256) || p.LimiteTiempoMS == 0 || p.LimiteRAMPicoBytes == 0 || p.LimiteDiscoPicoBytes == 0 || p.LimiteTokensAgente == 0 || len(p.Servicios) > 2 {
		return false
	}
	vistosP, vistosR := map[string]bool{}, map[uint32]bool{}
	tieneProxy := false
	for _, s := range p.Servicios {
		if s.Papel != "control_broker" && s.Papel != "controlled_egress_proxy" || vistosP[s.Papel] || s.Puerto == 0 || vistosR[s.Puerto] || !referenciaValida(s.ServicioRef, "servicio:", 160) || !referenciaValida(s.IdentidadRef, "identidad-servicio:", 160) || !sha256Valido(s.IdentidadSHA256) {
			return false
		}
		vistosP[s.Papel], vistosR[s.Puerto] = true, true
		tieneProxy = tieneProxy || s.Papel == "controlled_egress_proxy"
	}
	return egresoValido(p, tieneProxy)
}

func egresoValido(plan PlanLanzamiento, tieneProxy bool) bool {
	if plan.Egreso == nil {
		return !tieneProxy
	}
	egreso := plan.Egreso
	if !tieneProxy || egreso.Esquema != EsquemaConcesionEgreso || !referenciaValida(egreso.Referencia, "egreso:", 160) || len(egreso.Destinos) == 0 || len(egreso.Destinos) > maximoDestinosEgreso || egreso.MaximoConexiones == 0 || egreso.MaximoConexiones > maximoConexionesEgreso || egreso.LimiteTiempoMS == 0 || egreso.LimiteTiempoMS > plan.LimiteTiempoMS || egreso.LimiteSubidaBytes == 0 || egreso.LimiteSubidaBytes > maximoBytesEgreso || egreso.LimiteBajadaBytes == 0 || egreso.LimiteBajadaBytes > maximoBytesEgreso {
		return false
	}
	vistos := make(map[string]bool, len(egreso.Destinos))
	for _, destino := range egreso.Destinos {
		if !hostEgresoValido(destino.Host) || vistos[destino.Host] || len(destino.Puertos) == 0 || len(destino.Puertos) > maximoPuertosDestino {
			return false
		}
		vistos[destino.Host] = true
		puertos := make(map[uint16]bool, len(destino.Puertos))
		for _, puerto := range destino.Puertos {
			if puerto == 0 || puertos[puerto] {
				return false
			}
			puertos[puerto] = true
		}
	}
	return true
}

func hostEgresoValido(host string) bool {
	if host == "" || len(host) > 253 || host != strings.ToLower(host) || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	for _, etiqueta := range strings.Split(host, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || strings.HasPrefix(etiqueta, "-") || strings.HasSuffix(etiqueta, "-") {
			return false
		}
		for _, caracter := range []byte(etiqueta) {
			if !(caracter >= 'a' && caracter <= 'z' || caracter >= '0' && caracter <= '9' || caracter == '-') {
				return false
			}
		}
	}
	return true
}
