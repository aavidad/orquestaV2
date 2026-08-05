package agentmicrovm

import (
	"context"
	"net"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	CodeCredentialBrokerInvalid         = "agentmicrovm.credential_broker_invalid"
	CodeCredentialBrokerAuthorityDenied = "agentmicrovm.credential_broker_authority_denied"
	CodeCredentialBrokerUnavailable     = "agentmicrovm.credential_broker_unavailable"
	CodeCredentialBrokerReplayRejected  = "agentmicrovm.credential_broker_replay_rejected"
	CodeCredentialBrokerAmbiguous       = "agentmicrovm.credential_broker_ambiguous"
)

// CredentialBroker handles exactly one already-accepted control-broker
// connection. Listener lifecycle and peer authentication stay outside this
// adapter; the authenticated aperture is still crossed against durable launch
// authority before any credential claim is consumed.
type CredentialBroker struct {
	authorities ports.MicroVMHostLaunchAuthorityRegistry
	credentials credentials.OneShotStore
	timeout     time.Duration
}

// NewCredentialBroker requires an explicit finite timeout. A composition must
// choose the operational value; this security boundary has no hidden default.
func NewCredentialBroker(
	authorities ports.MicroVMHostLaunchAuthorityRegistry,
	credentialStore credentials.OneShotStore,
	timeout time.Duration,
) (*CredentialBroker, error) {
	if nilInterface(authorities) || nilInterface(credentialStore) || timeout <= 0 {
		return nil, fail(CodeCredentialBrokerInvalid, nil)
	}
	return &CredentialBroker{
		authorities: authorities,
		credentials: credentialStore,
		timeout:     timeout,
	}, nil
}

// Handle serves one complete one-shot exchange on conn. Once UseOnce enters
// its callback, every failure is terminal and ambiguous: the durable claim is
// never released and this method never writes a second response.
func (broker *CredentialBroker) Handle(ctx context.Context, conn net.Conn) error {
	if broker == nil || nilInterface(ctx) || nilInterface(conn) ||
		nilInterface(broker.authorities) || nilInterface(broker.credentials) || broker.timeout <= 0 {
		return fail(CodeCredentialBrokerInvalid, nil)
	}

	handleCtx, cancel := context.WithTimeout(ctx, broker.timeout)
	defer cancel()
	deadline, present := handleCtx.Deadline()
	if !present || handleCtx.Err() != nil || conn.SetDeadline(deadline) != nil {
		return fail(CodeCredentialBrokerUnavailable, nil)
	}
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	aperture, err := microvm.DecodificarAperturaServicioHostV1(conn)
	if err != nil || aperture.Papel != microvm.PapelServicioHostControlBroker {
		return fail(CodeCredentialBrokerAuthorityDenied, nil)
	}
	runRef, err := goal.NewExecutionRef(aperture.RunRef)
	if err != nil {
		return fail(CodeCredentialBrokerAuthorityDenied, nil)
	}
	authority, err := broker.authorities.Resolve(handleCtx, ports.MicroVMHostLaunchAuthorityKey{
		RunRef: runRef, ActionFence: aperture.Cerca,
	})
	if err != nil || !credentialBrokerApertureMatches(authority, aperture) {
		return fail(CodeCredentialBrokerAuthorityDenied, nil)
	}

	request, err := microvm.DecodificarSolicitudAuthJSONCodexV1(conn)
	if err != nil {
		broker.writeRejected(conn, microvm.CodigoErrorAuthJSONCodexSolicitudInvalida)
		return fail(CodeCredentialBrokerAuthorityDenied, nil)
	}
	if request.SesionRef != authority.SessionRef.String() || request.Cerca != authority.Key.ActionFence {
		broker.writeRejected(conn, microvm.CodigoErrorAuthJSONCodexAutoridadInvalida)
		return fail(CodeCredentialBrokerAuthorityDenied, nil)
	}

	callbackStarted := false
	result, useErr := broker.credentials.UseOnce(handleCtx, authority.OneShotClaim, func(secret credentials.Secret) error {
		callbackStarted = true
		material := secret.Bytes()
		secret.Destroy()
		if len(material) == 0 || len(material) > microvm.MaximoMaterialAuthJSONCodexV1 {
			clear(material)
			return fail(CodeCredentialBrokerAmbiguous, nil)
		}
		writeErr := microvm.EscribirRespuestaAuthJSONCodexV1(conn, microvm.CabeceraRespuestaAuthJSONCodexV1{
			Protocolo:        microvm.ProtocoloAuthJSONCodexOneShotV1,
			Estado:           microvm.EstadoRespuestaAuthJSONCodexDisponible,
			LongitudMaterial: uint32(len(material)),
		}, material)
		clear(material)
		if writeErr != nil {
			return fail(CodeCredentialBrokerAmbiguous, nil)
		}
		ack, err := microvm.DecodificarAcuseAuthJSONCodexV1(conn)
		if err != nil || !credentialBrokerAckMatches(ack, authority, microvm.EtapaAcuseAuthJSONCodexProyectada) {
			return fail(CodeCredentialBrokerAmbiguous, nil)
		}
		return nil
	})

	if callbackStarted {
		if useErr != nil || result.Replayed || credentials.ValidateOneShotUseResult(authority.OneShotClaim, result) != nil {
			return fail(CodeCredentialBrokerAmbiguous, nil)
		}
	} else {
		if credentials.HasErrorCode(useErr, credentials.ErrorAlreadyConsumed) && result.Replayed &&
			credentials.ValidateOneShotUseResult(authority.OneShotClaim, result) == nil {
			broker.writeRejected(conn, microvm.CodigoErrorAuthJSONCodexRepeticionRechazada)
			return fail(CodeCredentialBrokerReplayRejected, nil)
		}
		broker.writeRejected(conn, microvm.CodigoErrorAuthJSONCodexNoDisponible)
		return fail(CodeCredentialBrokerUnavailable, nil)
	}

	ack, err := microvm.DecodificarAcuseAuthJSONCodexV1(conn)
	if err != nil || !credentialBrokerAckMatches(ack, authority, microvm.EtapaAcuseAuthJSONCodexPurgada) {
		return fail(CodeCredentialBrokerAmbiguous, nil)
	}
	return nil
}

func credentialBrokerApertureMatches(
	authority ports.MicroVMHostLaunchAuthorityV1,
	aperture microvm.AperturaServicioHostV1,
) bool {
	if ports.ValidateMicroVMHostLaunchAuthorityBoundV1(authority) != nil || len(authority.Services) == 0 {
		return false
	}
	service := authority.Services[0]
	return authority.Key.RunRef.String() == aperture.RunRef &&
		authority.Key.ActionFence == aperture.Cerca &&
		authority.ExternalRef == aperture.EjecucionRef &&
		authority.PlanSHA256 == aperture.PlanSHA256 &&
		authority.ConcessionSHA256 == aperture.ConcesionSHA256 &&
		service.Role == ports.MicroVMHostServiceControlBroker &&
		service.ServiceRef == aperture.ServicioRef &&
		service.IdentityRef == aperture.IdentidadRef &&
		service.IdentitySHA256 == aperture.IdentidadSHA256
}

func credentialBrokerAckMatches(
	ack microvm.AcuseAuthJSONCodexV1,
	authority ports.MicroVMHostLaunchAuthorityV1,
	stage microvm.EtapaAcuseAuthJSONCodexV1,
) bool {
	return ack.SesionRef == authority.SessionRef.String() &&
		ack.Cerca == authority.Key.ActionFence && ack.Etapa == stage && ack.Correcto && ack.CodigoError == nil
}

func (*CredentialBroker) writeRejected(conn net.Conn, code microvm.CodigoErrorAuthJSONCodexV1) {
	responseCode := code
	_ = microvm.EscribirRespuestaAuthJSONCodexV1(conn, microvm.CabeceraRespuestaAuthJSONCodexV1{
		Protocolo:   microvm.ProtocoloAuthJSONCodexOneShotV1,
		Estado:      microvm.EstadoRespuestaAuthJSONCodexRechazada,
		CodigoError: &responseCode,
	}, nil)
}
