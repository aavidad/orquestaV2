// Este fichero traduce la lectura oficial de cuota sin decidir persistencia.
package codex

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type ventanaCuotaCodex struct {
	Porcentaje *int   `json:"usedPercent"`
	Duracion   *int64 `json:"windowDurationMins"`
	Reinicio   *int64 `json:"resetsAt"`
}
type lecturaCuotaCodex struct {
	Creditos              json.RawMessage    `json:"credits"`
	LimiteIndividual      json.RawMessage    `json:"individualLimit"`
	IDLimite              *string            `json:"limitId"`
	NombreLimite          *string            `json:"limitName"`
	Plan                  *string            `json:"planType"`
	Primaria              *ventanaCuotaCodex `json:"primary"`
	TipoAlcanzado         *string            `json:"rateLimitReachedType"`
	Secundaria            *ventanaCuotaCodex `json:"secondary"`
	ControlGastoAlcanzado *bool              `json:"spendControlReached"`
}
type respuestaCuotaCodex struct {
	Limites          *lecturaCuotaCodex `json:"rateLimits"`
	CreditosReinicio json.RawMessage    `json:"rateLimitResetCredits"`
	LimitesPorID     json.RawMessage    `json:"rateLimitsByLimitId"`
}
type traduccionCuotaCodex struct {
	Observacion application.AgentQuotaObservation
	Evidencia   json.RawMessage
}

func traducirLecturaCuotaCodex(datos []byte, colocacion ports.AgentPlacementRef, vigencia time.Duration, ahora func() time.Time) (traduccionCuotaCodex, error) {
	if colocacion.String() == "" || vigencia <= 0 || ahora == nil {
		return traduccionCuotaCodex{}, &Error{Code: CodeStateInvalid}
	}
	var respuesta respuestaCuotaCodex
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	if decodificador.Decode(&respuesta) != nil || requireJSONEOF(decodificador) != nil ||
		respuesta.Limites == nil {
		return traduccionCuotaCodex{}, &Error{Code: CodeOutputInvalid}
	}
	instante := ahora().Round(0).UTC()
	if instante.IsZero() || !instante.Add(vigencia).After(instante) {
		return traduccionCuotaCodex{}, &Error{Code: CodeClockInvalid}
	}
	limites, estado, reinicio := *respuesta.Limites, application.AgentQuotaUnknown, time.Time{}
	idLimite, invalido, agotado, validas := "", false, false, 0
	if limites.IDLimite != nil && *limites.IDLimite != "" {
		idLimite = *limites.IDLimite
		if !validAccountProfileID(idLimite) {
			idLimite, invalido, limites.IDLimite = "", true, nil
		}
	}
	if limites.TipoAlcanzado != nil {
		if tipoAlcanzadoValido(*limites.TipoAlcanzado) {
			agotado = true
		} else {
			invalido = true
			limites.TipoAlcanzado = nil
		}
	}
	if limites.ControlGastoAlcanzado != nil {
		agotado = agotado || *limites.ControlGastoAlcanzado
	}
	for _, ventana := range []*ventanaCuotaCodex{limites.Primaria, limites.Secundaria} {
		if ventana == nil {
			continue
		}
		if ventana.Porcentaje == nil || *ventana.Porcentaje < 0 || *ventana.Porcentaje > 100 || ventana.Duracion == nil ||
			*ventana.Duracion <= 0 || ventana.Reinicio == nil {
			invalido = true
			continue
		}
		candidata := time.Unix(*ventana.Reinicio, 0).UTC()
		if !candidata.After(instante) {
			invalido = true
			continue
		}
		validas++
		if reinicio.IsZero() || candidata.Before(reinicio) {
			reinicio = candidata
		}
		agotado = agotado || *ventana.Porcentaje == 100
	}
	if !invalido && agotado {
		estado = application.AgentQuotaExhausted
	} else if !invalido && validas > 0 &&
		(limites.ControlGastoAlcanzado == nil || !*limites.ControlGastoAlcanzado) {
		estado = application.AgentQuotaAvailable
	}
	reintento := time.Time{}
	if estado == application.AgentQuotaExhausted {
		reintento = reinicio
	}
	limites.Creditos, limites.LimiteIndividual, limites.NombreLimite, limites.Plan = nil, nil, nil, nil
	if idLimite == "" {
		limites.IDLimite = nil
	}
	evidencia, _ := json.Marshal(limites)
	identidad := fmt.Sprintf("%s\x00%s\x00%s\x00%s", colocacion.String(), idLimite,
		identidadVentana(limites.Primaria), identidadVentana(limites.Secundaria))
	digest := sha256.Sum256([]byte(identidad))
	observacion := application.AgentQuotaObservation{PlacementRef: colocacion,
		WindowRef: application.AgentQuotaWindowRef(fmt.Sprintf("codex-quota:v1:sha256:%x", digest)),
		Status:    estado, Quality: "measured", ObservedAt: instante, ExpiresAt: instante.Add(vigencia),
		ResetAt: reinicio, RetryAt: reintento}
	return traduccionCuotaCodex{observacion, evidencia}, nil
}

func tipoAlcanzadoValido(valor string) bool {
	return strings.Contains("|rate_limit_reached|workspace_owner_credits_depleted|workspace_member_credits_depleted|workspace_owner_usage_limit_reached|workspace_member_usage_limit_reached|", "|"+valor+"|")
}
func identidadVentana(ventana *ventanaCuotaCodex) string {
	if ventana == nil || ventana.Duracion == nil || ventana.Reinicio == nil {
		return "-"
	}
	return fmt.Sprintf("%d:%d", *ventana.Duracion, *ventana.Reinicio)
}
