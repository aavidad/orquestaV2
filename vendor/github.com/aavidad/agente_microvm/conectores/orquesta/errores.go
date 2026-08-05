package microvm

import "fmt"

type ErrorConfiguracion struct {
	Causa string
}

func (e *ErrorConfiguracion) Error() string {
	return "microvm.configuracion_invalida:" + e.Causa
}

type ErrorProtocolo struct {
	Recibido string
}

func (e *ErrorProtocolo) Error() string {
	return "microvm.protocolo_incompatible"
}

type ErrorRespuestaGrande struct{}

func (e *ErrorRespuestaGrande) Error() string {
	return "microvm.respuesta_demasiado_grande"
}

type ErrorRespuesta struct {
	Estado  int
	Codigo  string
	Detalle string
}

func (e *ErrorRespuesta) Error() string {
	return fmt.Sprintf("microvm.respuesta_rechazada:%d:%s", e.Estado, e.Codigo)
}
