package orquestacouncil_test

import (
	"errors"
	"strings"
	"testing"

	council "orquesta/modulos/orquesta-council"
)

func miembrosV0() []council.MemberV0 {
	return []council.MemberV0{
		{MemberRef: "autor", FamilyRef: "familia-a", BudgetRemaining: 0.90, CapabilityRank: 2},
		{MemberRef: "sobrado", FamilyRef: "familia-a", BudgetRemaining: 0.80, CapabilityRank: 3},
		{MemberRef: "justito", FamilyRef: "familia-b", BudgetRemaining: 0.05, CapabilityRank: 1},
		{MemberRef: "medio", FamilyRef: "familia-b", BudgetRemaining: 0.50, CapabilityRank: 5},
	}
}

func asientoV0(t *testing.T, assignment council.AssignmentV0, role council.RoleV0) council.SeatV0 {
	t.Helper()
	for _, seat := range assignment.Seats {
		if seat.Role == role {
			return seat
		}
	}
	t.Fatalf("no se asigno el rol %s", role)
	return council.SeatV0{}
}

// El rol se asigna por presupuesto EN CALIENTE, no por modelo: el de mas cuota
// traga diffs crudos, el de menos decide sobre material masticado.
func TestAsignacionAutomaticaPorPresupuestoV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-1",
		AuthorRef:  "autor",
		Members:    miembrosV0(),
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}

	revisor := asientoV0(t, assignment, council.RoleRevisorV0)
	if revisor.MemberRef != "sobrado" {
		t.Fatalf("revisor = %s, quiero el de mas presupuesto (sobrado)", revisor.MemberRef)
	}
	if revisor.Material != council.MaterialDiffCrudoV0 {
		t.Fatalf("el revisor debe recibir diffs crudos, recibe %s", revisor.Material)
	}

	consultor := asientoV0(t, assignment, council.RoleConsultorV0)
	if consultor.MemberRef != "justito" {
		t.Fatalf("consultor = %s, quiero el de menos presupuesto (justito)", consultor.MemberRef)
	}
	if consultor.Material != council.MaterialMasticadoV0 {
		t.Fatalf("el consultor decide sobre material masticado, no lee diffs; recibe %s", consultor.Material)
	}
	if consultor.BudgetWarning == "" {
		t.Fatal("un consultor al 5% de cuota debe avisar")
	}

	adversario := asientoV0(t, assignment, council.RoleAdversarioV0)
	if adversario.FamilyRef == "familia-a" {
		t.Fatalf("el adversario no puede ser de la familia del autor: %s", adversario.FamilyRef)
	}
}

// El operador manda en el reparto: el override gana a la asignacion automatica, y
// queda registrado quien forzo que y que habria salido solo.
func TestOverrideManualGanaYDejaEvidenciaV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-2",
		AuthorRef:  "autor",
		Members:    miembrosV0(),
		Overrides: []council.RoleOverrideV0{{
			Role:      council.RoleRevisorV0,
			MemberRef: "medio",
			ForcedBy:  "operador",
			Reason:    "quiero a medio revisando esto",
		}},
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}

	revisor := asientoV0(t, assignment, council.RoleRevisorV0)
	if revisor.MemberRef != "medio" {
		t.Fatalf("el override no gano: revisor = %s", revisor.MemberRef)
	}
	if !revisor.Forced || revisor.ForcedBy != "operador" || revisor.ForcedReason == "" {
		t.Fatalf("el override no dejo evidencia de quien forzo y por que: %+v", revisor)
	}
	if revisor.AutomaticMemberRef != "sobrado" {
		t.Fatalf("no se conservo lo que habria elegido el automatico: %q", revisor.AutomaticMemberRef)
	}
}

// Forzar a alguien sin cuota AVISA pero OBEDECE: es decision del operador.
func TestOverrideSinCuotaAvisaPeroObedeceV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-3",
		AuthorRef:  "autor",
		Members:    miembrosV0(),
		Overrides: []council.RoleOverrideV0{{
			Role:      council.RoleRevisorV0,
			MemberRef: "justito",
			ForcedBy:  "operador",
		}},
	})
	if err != nil {
		t.Fatalf("forzar a un miembro sin cuota debe obedecer, no bloquear: %v", err)
	}
	revisor := asientoV0(t, assignment, council.RoleRevisorV0)
	if revisor.MemberRef != "justito" {
		t.Fatalf("no obedecio el override: %s", revisor.MemberRef)
	}
	if revisor.BudgetWarning == "" || len(assignment.Warnings) == 0 {
		t.Fatal("obedecio pero no aviso de la falta de cuota")
	}
}

// El override manda en el REPARTO, no en las REGLAS DE INTEGRIDAD.
func TestOverrideNoPuedeSaltarseLaIntegridadV0(t *testing.T) {
	_, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-4",
		AuthorRef:  "autor",
		Members:    miembrosV0(),
		Overrides: []council.RoleOverrideV0{{
			Role:      council.RoleRevisorV0,
			MemberRef: "autor",
			ForcedBy:  "operador",
		}},
	})
	if !errors.Is(err, council.ErrAutorNoSeRevisaV0) {
		t.Fatalf("el autor no puede revisarse a si mismo ni forzandolo: %v", err)
	}

	_, err = council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-5",
		AuthorRef:  "autor",
		Members:    miembrosV0(),
		Overrides: []council.RoleOverrideV0{{
			Role:      council.RoleAdversarioV0,
			MemberRef: "sobrado",
			ForcedBy:  "operador",
		}},
	})
	if !errors.Is(err, council.ErrAdversarioMismaFamiliaV0) {
		t.Fatalf("el adversario no puede ser de la familia del autor ni forzandolo: %v", err)
	}
}

func TestSeguridadSoloSeConvocaEnDecisionCriticaV0(t *testing.T) {
	normal, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-6", AuthorRef: "autor", Members: miembrosV0(),
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}
	for _, seat := range normal.Seats {
		if seat.Role == council.RoleSeguridadV0 {
			t.Fatal("seguridad no debe convocarse en una decision no critica")
		}
	}

	critico, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-7", AuthorRef: "autor", Members: miembrosV0(), SecurityCritical: true,
	})
	if err != nil {
		t.Fatalf("AssignRolesV0 critico: %v", err)
	}
	seguridad := asientoV0(t, critico, council.RoleSeguridadV0)
	if seguridad.MemberRef == "" {
		t.Fatal("una decision critica debe cubrir el rol de seguridad")
	}
}

func TestUmbralDeDosTerciosYEmpateEsReworkV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-8", AuthorRef: "autor", Members: miembrosV0(),
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}

	aceptado, err := council.DecideV0(assignment, votosV0(assignment, council.VoteApproveV0, council.VoteApproveV0, council.VoteApproveV0))
	if err != nil {
		t.Fatalf("DecideV0: %v", err)
	}
	if aceptado.Outcome != council.OutcomeAcceptedV0 {
		t.Fatalf("tres aprobaciones de tres deben aceptar: %+v", aceptado)
	}

	// 2 de 3 son exactamente dos tercios: alcanza el umbral.
	justo, err := council.DecideV0(assignment, votosV0(assignment, council.VoteApproveV0, council.VoteReworkV0, council.VoteApproveV0))
	if err != nil {
		t.Fatalf("DecideV0: %v", err)
	}
	if justo.Outcome != council.OutcomeAcceptedV0 {
		t.Fatalf("2 de 3 son dos tercios exactos y deben aceptar: %+v", justo)
	}

	// 1 de 3 se queda corto.
	rework, err := council.DecideV0(assignment, votosV0(assignment, council.VoteApproveV0, council.VoteReworkV0, council.VoteReworkV0))
	if err != nil {
		t.Fatalf("DecideV0: %v", err)
	}
	if rework.Outcome != council.OutcomeReworkV0 {
		t.Fatalf("1 de 3 no alcanza dos tercios: %+v", rework)
	}
	if !strings.Contains(rework.Rationale, "no alcanzan el umbral") {
		t.Fatalf("el rationale no explica el rechazo: %q", rework.Rationale)
	}
}

// El empate no se resuelve por sorteo ni por voto de calidad: vuelve el trabajo.
func TestEmpateEsReworkNoAceptacionV0(t *testing.T) {
	assignment := council.AssignmentV0{
		CouncilRef: "consejo-empate",
		Seats: []council.SeatV0{
			{Role: council.RoleRevisorV0, MemberRef: "a"},
			{Role: council.RoleConsultorV0, MemberRef: "b"},
			{Role: council.RoleAdversarioV0, MemberRef: "c"},
			{Role: council.RoleSeguridadV0, MemberRef: "d"},
		},
	}
	decision, err := council.DecideV0(assignment, []council.BallotV0{
		{MemberRef: "a", Vote: council.VoteApproveV0},
		{MemberRef: "b", Vote: council.VoteApproveV0},
		{MemberRef: "c", Vote: council.VoteReworkV0},
		{MemberRef: "d", Vote: council.VoteReworkV0},
	})
	if err != nil {
		t.Fatalf("DecideV0: %v", err)
	}
	if decision.Outcome != council.OutcomeReworkV0 {
		t.Fatalf("un empate 2-2 esta por debajo de dos tercios y es rework: %+v", decision)
	}
}

// El veto de seguridad no se compensa con mayoria: ninguna cantidad de
// aprobaciones lo levanta.
func TestVetoDeSeguridadNoSeLevantaConMayoriaV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-9", AuthorRef: "autor", Members: miembrosV0(), SecurityCritical: true,
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}

	// Un miembro puede llevar dos sombreros (aqui adversario y seguridad), pero
	// vota UNA vez. Si tiene el rol de seguridad, puede vetar.
	votos := map[string]council.VoteV0{}
	for _, seat := range assignment.Seats {
		if _, ya := votos[seat.MemberRef]; !ya {
			votos[seat.MemberRef] = council.VoteApproveV0
		}
		if seat.Role == council.RoleSeguridadV0 {
			votos[seat.MemberRef] = council.VoteBlockV0
		}
	}
	ballots := make([]council.BallotV0, 0, len(votos))
	for member, vote := range votos {
		ballots = append(ballots, council.BallotV0{MemberRef: member, Vote: vote})
	}

	decision, err := council.DecideV0(assignment, ballots)
	if err != nil {
		t.Fatalf("DecideV0: %v", err)
	}
	if decision.Outcome != council.OutcomeBlockedV0 {
		t.Fatalf("el veto de seguridad debe bloquear pese a la mayoria: %+v", decision)
	}
	if decision.Approvals < 2 {
		t.Fatal("el escenario no tenia mayoria que vencer; no prueba nada")
	}
}

// Solo seguridad puede vetar: un revisor no puede bloquear por su cuenta.
func TestSoloSeguridadPuedeVetarV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-10", AuthorRef: "autor", Members: miembrosV0(),
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}
	ballots := votosV0(assignment, council.VoteBlockV0, council.VoteApproveV0, council.VoteApproveV0)
	if _, err := council.DecideV0(assignment, ballots); !errors.Is(err, council.ErrVetoSinSeguridadV0) {
		t.Fatalf("un rol distinto de seguridad no puede vetar: %v", err)
	}
}

func TestQuorumIncompletoNoDecideV0(t *testing.T) {
	assignment, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "consejo-11", AuthorRef: "autor", Members: miembrosV0(),
	})
	if err != nil {
		t.Fatalf("AssignRolesV0: %v", err)
	}
	parcial := []council.BallotV0{{
		MemberRef: assignment.Seats[0].MemberRef,
		Role:      assignment.Seats[0].Role,
		Vote:      council.VoteApproveV0,
	}}
	if _, err := council.DecideV0(assignment, parcial); !errors.Is(err, council.ErrQuorumIncompletoV0) {
		t.Fatalf("sin quorum no hay decision: %v", err)
	}
}

func votosV0(assignment council.AssignmentV0, votes ...council.VoteV0) []council.BallotV0 {
	ballots := make([]council.BallotV0, 0, len(assignment.Seats))
	for idx, seat := range assignment.Seats {
		vote := council.VoteApproveV0
		if idx < len(votes) {
			vote = votes[idx]
		}
		ballots = append(ballots, council.BallotV0{MemberRef: seat.MemberRef, Role: seat.Role, Vote: vote})
	}
	return ballots
}

// HALLAZGO CRITICO (Codex): contar FILAS en vez de IDENTIDADES permitia repetir
// el mismo member_ref y fabricar un consejo de UNA sola persona que se aprobaba
// a si misma: outcome=accepted, total=1, approvals=1.
func TestConsejoDeUnoNoPuedeAprobarseASiMismoV0(t *testing.T) {
	_, err := council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "c", AuthorRef: "autor",
		Members: []council.MemberV0{
			{MemberRef: "autor", FamilyRef: "f", BudgetRemaining: 0.9},
			{MemberRef: "x", FamilyRef: "g", BudgetRemaining: 0.5},
			{MemberRef: "x", FamilyRef: "g", BudgetRemaining: 0.5},
		},
	})
	if !errors.Is(err, council.ErrMiembroDuplicadoV0) {
		t.Fatalf("un member_ref duplicado debe rechazarse: %v", err)
	}

	// Y sin duplicados, un solo revisor tampoco es un consejo.
	_, err = council.AssignRolesV0(council.ConvocationV0{
		CouncilRef: "c", AuthorRef: "autor",
		Members: []council.MemberV0{
			{MemberRef: "autor", FamilyRef: "f", BudgetRemaining: 0.9},
			{MemberRef: "x", FamilyRef: "g", BudgetRemaining: 0.5},
		},
	})
	if !errors.Is(err, council.ErrConsejoDeUnoV0) {
		t.Fatalf("una sola identidad revisora no es un consejo: %v", err)
	}
}

// Ultima linea de defensa: aunque la asignacion venga fabricada de fuera, un
// consejo de uno no puede aceptar nada.
func TestDecideRechazaUnConsejoDeUnaSolaIdentidadV0(t *testing.T) {
	fabricado := council.AssignmentV0{
		CouncilRef: "c",
		Seats: []council.SeatV0{
			{Role: council.RoleRevisorV0, MemberRef: "x"},
			{Role: council.RoleConsultorV0, MemberRef: "x"},
			{Role: council.RoleAdversarioV0, MemberRef: "x"},
		},
	}
	_, err := council.DecideV0(fabricado, []council.BallotV0{
		{MemberRef: "x", Vote: council.VoteApproveV0},
	})
	if !errors.Is(err, council.ErrConsejoDeUnoV0) {
		t.Fatalf("un consejo de una identidad no puede decidir: %v", err)
	}
}
