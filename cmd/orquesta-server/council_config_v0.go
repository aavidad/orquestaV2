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
	// DoubleReviewRequired exige dos revisiones independientes antes de cerrar una
	// entrega material. Va aqui y NO en una env nueva: el presupuesto esta en 426.
	DoubleReviewRequired bool                                 `json:"double_review_required,omitempty"`
	Members              []serverProjectConfigCouncilMemberV0 `json:"members,omitempty"`
}

// serverProjectConfigCouncilMemberV0 declara QUIEN puede sentarse en el consejo y
// con que familia. El PRESUPUESTO no se declara aqui: se observa en caliente. Si
// se declarara, cualquiera podria fabricar el reparto de roles escribiendo un
// numero en un fichero.
type serverProjectConfigCouncilMemberV0 struct {
	MemberRef      string `json:"member_ref"`
	FamilyRef      string `json:"family_ref,omitempty"`
	CapabilityRank int    `json:"capability_rank,omitempty"`
}
