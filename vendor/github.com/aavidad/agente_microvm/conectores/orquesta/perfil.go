package microvm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
)

const (
	EsquemaDescriptorPerfilLanzamientoV1 = "agentmicrovm.descriptor-perfil-lanzamiento.v1"
	prefijoDescriptorPerfilLanzamientoV1 = "perfil-lanzamiento:sha256:"
	prefijoEjecutorPerfilV1              = "ejecutor:perfil-sha256:"
)

// DescriptorPerfilLanzamientoV1 describe un perfil físico sin publicar rutas.
type DescriptorPerfilLanzamientoV1 struct {
	Esquema              string          `json:"esquema"`
	DescriptorRef        string          `json:"descriptor_ref"`
	EjecutorRef          string          `json:"ejecutor_ref"`
	VCPU                 uint8           `json:"vcpu"`
	MemoriaMiB           uint32          `json:"memoria_mib"`
	KernelSHA256         string          `json:"kernel_sha256"`
	InitramfsSHA256      string          `json:"initramfs_sha256"`
	PerfilSHA256         string          `json:"perfil_sha256"`
	ServiciosDisponibles []ServicioVsock `json:"servicios_disponibles"`
}

type ErrorDescriptorPerfilLanzamientoV1 struct{ Codigo string }

func (e *ErrorDescriptorPerfilLanzamientoV1) Error() string { return e.Codigo }

func ConstruirEjecutorRefPerfilV1(perfilSHA256 string) (string, error) {
	if !sha256Valido(perfilSHA256) {
		return "", errorDescriptorPerfil("descriptor_perfil.activo_invalido")
	}
	return prefijoEjecutorPerfilV1 + perfilSHA256, nil
}

// MensajeCanonicoDescriptorPerfilLanzamientoV1 excluye DescriptorRef, derivado del digest.
func MensajeCanonicoDescriptorPerfilLanzamientoV1(descriptor DescriptorPerfilLanzamientoV1) ([]byte, error) {
	if err := validarContenidoDescriptorPerfil(descriptor); err != nil {
		return nil, err
	}
	b := append([]byte("agentmicrovm.descriptor-perfil-lanzamiento.v1\x00"), campo(descriptor.Esquema)...)
	b = append(b, campo(descriptor.EjecutorRef)...)
	b = append(b, descriptor.VCPU)
	b = binary.BigEndian.AppendUint32(b, descriptor.MemoriaMiB)
	b = append(b, campo(descriptor.KernelSHA256)...)
	b = append(b, campo(descriptor.InitramfsSHA256)...)
	b = append(b, campo(descriptor.PerfilSHA256)...)
	servicios := append([]ServicioVsock(nil), descriptor.ServiciosDisponibles...)
	sort.Slice(servicios, func(i, j int) bool { return servicios[i].Papel < servicios[j].Papel })
	b = append(b, byte(len(servicios)))
	for _, servicio := range servicios {
		b = append(b, campo(servicio.Papel)...)
		b = append(b, campo(servicio.ServicioRef)...)
		b = binary.BigEndian.AppendUint32(b, servicio.Puerto)
		b = append(b, campo(servicio.IdentidadRef)...)
		b = append(b, campo(servicio.IdentidadSHA256)...)
	}
	return b, nil
}

func CalcularSHA256DescriptorPerfilLanzamientoV1(descriptor DescriptorPerfilLanzamientoV1) (string, error) {
	mensaje, err := MensajeCanonicoDescriptorPerfilLanzamientoV1(descriptor)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(mensaje)
	return hex.EncodeToString(digest[:]), nil
}

func ValidarDescriptorPerfilLanzamientoV1(descriptor DescriptorPerfilLanzamientoV1) error {
	digest, err := CalcularSHA256DescriptorPerfilLanzamientoV1(descriptor)
	if err != nil {
		return err
	}
	if descriptor.DescriptorRef != prefijoDescriptorPerfilLanzamientoV1+digest {
		return errorDescriptorPerfil("descriptor_perfil.digest_no_coincide")
	}
	return nil
}

// DecodificarDescriptorPerfilLanzamientoV1 rechaza extensiones y valores JSON concatenados.
func DecodificarDescriptorPerfilLanzamientoV1(datos []byte) (DescriptorPerfilLanzamientoV1, error) {
	var descriptor DescriptorPerfilLanzamientoV1
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&descriptor); err != nil {
		return DescriptorPerfilLanzamientoV1{}, errorDescriptorPerfil("descriptor_perfil.json_invalido")
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return DescriptorPerfilLanzamientoV1{}, errorDescriptorPerfil("descriptor_perfil.json_invalido")
	}
	if err := ValidarDescriptorPerfilLanzamientoV1(descriptor); err != nil {
		return DescriptorPerfilLanzamientoV1{}, err
	}
	return descriptor, nil
}

// ValidarPlanConDescriptorPerfilLanzamientoV1 exige hechos físicos publicados.
func ValidarPlanConDescriptorPerfilLanzamientoV1(descriptor DescriptorPerfilLanzamientoV1, plan PlanLanzamiento) error {
	if err := ValidarDescriptorPerfilLanzamientoV1(descriptor); err != nil {
		return err
	}
	if !planValido(plan) {
		return errorDescriptorPerfil("descriptor_perfil.plan_invalido")
	}
	if plan.VCPU != descriptor.VCPU || plan.MemoriaMiB != descriptor.MemoriaMiB ||
		plan.KernelSHA256 != descriptor.KernelSHA256 || plan.InitramfsSHA256 != descriptor.InitramfsSHA256 ||
		plan.PerfilSHA256 == nil || *plan.PerfilSHA256 != descriptor.PerfilSHA256 {
		return errorDescriptorPerfil("descriptor_perfil.plan_no_coincide")
	}
	for _, servicioPlan := range plan.Servicios {
		disponible := false
		for _, servicioDescriptor := range descriptor.ServiciosDisponibles {
			if servicioPlan == servicioDescriptor {
				disponible = true
				break
			}
		}
		if !disponible {
			return errorDescriptorPerfil("descriptor_perfil.plan_no_coincide")
		}
	}
	return nil
}

func validarContenidoDescriptorPerfil(descriptor DescriptorPerfilLanzamientoV1) error {
	if descriptor.Esquema != EsquemaDescriptorPerfilLanzamientoV1 {
		return errorDescriptorPerfil("descriptor_perfil.esquema_incompatible")
	}
	if descriptor.VCPU < 1 || descriptor.VCPU > 32 || descriptor.MemoriaMiB < 64 || descriptor.MemoriaMiB > 32768 {
		return errorDescriptorPerfil("descriptor_perfil.recursos_invalidos")
	}
	if !sha256Valido(descriptor.KernelSHA256) || !sha256Valido(descriptor.InitramfsSHA256) || !sha256Valido(descriptor.PerfilSHA256) {
		return errorDescriptorPerfil("descriptor_perfil.activo_invalido")
	}
	ejecutorRef, _ := ConstruirEjecutorRefPerfilV1(descriptor.PerfilSHA256)
	if descriptor.EjecutorRef != ejecutorRef {
		return errorDescriptorPerfil("descriptor_perfil.ejecutor_invalido")
	}
	if len(descriptor.ServiciosDisponibles) > 2 {
		return errorDescriptorPerfil("descriptor_perfil.demasiados_servicios")
	}
	papeles, puertos, referencias := map[string]bool{}, map[uint32]bool{}, map[string]bool{}
	for _, servicio := range descriptor.ServiciosDisponibles {
		if servicio.Papel != "control_broker" && servicio.Papel != "controlled_egress_proxy" ||
			!referenciaValida(servicio.ServicioRef, "servicio:", 160) || servicio.Puerto == 0 ||
			!referenciaValida(servicio.IdentidadRef, "identidad-servicio:", 160) || !sha256Valido(servicio.IdentidadSHA256) {
			return errorDescriptorPerfil("descriptor_perfil.servicio_invalido")
		}
		if papeles[servicio.Papel] || puertos[servicio.Puerto] || referencias[servicio.ServicioRef] {
			return errorDescriptorPerfil("descriptor_perfil.servicios_duplicados")
		}
		papeles[servicio.Papel], puertos[servicio.Puerto], referencias[servicio.ServicioRef] = true, true, true
	}
	return nil
}

func errorDescriptorPerfil(codigo string) error {
	return &ErrorDescriptorPerfilLanzamientoV1{Codigo: codigo}
}
