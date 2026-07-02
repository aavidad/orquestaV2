# Incidencia: AFTER_SECONDS=0 bloqueaba capacidad libre

Fecha: 2026-07-02.

## Sintoma

El agente remoto propuso cambiar el contrato de
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`: la intencion correcta
era desactivar solo el disparador por reloj idle, pero conservar el relleno de
cola cuando hay capacidad libre y `target_queue` lo pide.

El cambio remoto tocaba el modulo puro, pero no cerraba el contrato completo:
`cmd/orquesta-server/config.go` seguia convirtiendo `AFTER_SECONDS=0` en
`IdleSelfImprovementDisabled=true`, lo que apagaba tambien el disparo
`capacity_free`.

## Riesgo

La automejora goal-first podia quedar sin trabajo precisamente en entornos donde
el operador queria evitar el disparo por tiempo, pero si permitir mantener una
cola acotada de reparaciones seguras.

## Cambio

- `ConfigV0` separa `IdleSelfImprovementDisabled` global de
  `IdleSelfImprovementIdleDisabled`.
- `AFTER_SECONDS=0` marca solo el trigger idle como apagado.
- Los contextos OPES/dominio sin workdir separado siguen apagando la automejora
  globalmente.
- Si el supervisor esta idle pero el trigger por reloj esta apagado, el runtime
  evalua `capacity_free`.
- El modulo puro alinea la misma semantica.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming
go test -count=1 ./modulos/orquesta-server -run 'Test(RuntimeV0SupervisorPreparaCapacidadAunqueRelojIdleEsteDesactivado|RuntimeV0SupervisorPreparaAutomejoraConColaVacia|RuntimeV0IdleSelfImprovementAfterZeroDesactivaPlanificacion|NormalizeConfigV0)'
go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0(ConfiguraAutomejoraIdle|AceptaAliasLegacyDeAutomejoraIdle|AutomejoraIdleDefaultYApagado|DesactivaAutomejoraIdle|DetectaOPES|CanonicaGana)'
```

## Residual

El apagado global sigue existiendo por `IdleSelfImprovementDisabled`, usado para
sesiones de dominio/OPES no aisladas. No se abre produccion ni se modifica OPES
productivo.
