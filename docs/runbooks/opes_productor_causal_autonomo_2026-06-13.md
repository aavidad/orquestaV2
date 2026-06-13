# Productor Causal OPES Autonomo

Fecha: 2026-06-13.

## Problema

OPES podia lanzar agentes y recibir artefactos, pero el cierre no era autonomo:
un ACK con pendientes, rework o paquete incompleto quedaba registrado y un
operador tenia que convertirlo manualmente en la siguiente ola de trabajo.

## Solucion

El servidor residente ejecuta un productor causal OPES antes de drenar el
Director residente normal. El productor lee el ledger durable de entregas
`DomainWorkArtifactSubmissionRecordV0` y crea nuevos `DomainWorkJobRequestV0`
cuando encuentra:

- `document_plan` aceptado: expande trabajos derivados;
- `plan_temario` incompleto: pide rework del plan hasta cubrir toda la
  secuencia OPES;
- paquete final con `pendiente_continuar`: abre followup causal;
- `followup_refs`, `pending_followup_refs`, `rework_refs` o
  `missing_required_refs`: materializa la siguiente tarea;
- receipt rechazado: crea correccion conservando el trabajo recuperable.

La logica vive en `modulos/orquesta-opes-director`. El servidor solo cablea
puertos y wakeups. El nucleo no importa OPES.

## Idempotencia

Cada job causal usa `idempotency_key` estable con fuente, artefacto, followup y
fase. Si el backend de `domain-work` expone `DomainWorkJobRecordSourcePortV0`,
el productor consulta antes de crear. Si no lo expone, delega en la idempotencia
del backend remoto.

## Wakeup

El ledger de artefactos del composition root esta decorado para despertar:

- supervisor residente;
- Director residente.

La causa registrada es `domain_work_artifact_recorded`.

## No Cierre Falso

Un paquete final con estado `pendiente_continuar` no se interpreta como cierre.
Si no declara refs concretas, el productor crea el followup
`final-package-pendiente-continuar`.

## Pruebas

Comandos usados en la implementacion:

```bash
go test -count=1 ./modulos/orquesta-opes-director
go test -count=1 ./cmd/orquesta-server -run 'OPESCausal|WakeupDomainWork|ResidentDirector'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'DomainWork|ResidentDirector|DocumentPlan'
go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-file ./modulos/orquesta-document-plan-expander ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director
go test -count=1 ./cmd/orquesta-server -run 'DomainWork|OPES|Wakeup|ResidentDirector|Stack'
```
