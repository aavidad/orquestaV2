package main

// serverProjectConfigCouncilV0 activa el gate de creacion del consejo. Va en la
// config canonica y NO en una env nueva: el presupuesto de envs esta en 426 y no
// se sube por esto.
//
// gate_required=false por defecto, y no es un descuido: sin fuente real de
// miembros el gate obligatorio bloquearia toda creacion. Se activa cuando esa
// fuente exista.
type serverProjectConfigCouncilV0 struct {
	GateRequired bool `json:"gate_required,omitempty"`
}
