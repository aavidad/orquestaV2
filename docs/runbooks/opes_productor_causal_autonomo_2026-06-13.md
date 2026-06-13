# Productor Causal OPES Autónomo

Fecha: 2026-06-13.

## Problema

OPES podía lanzar agentes y recibir artefactos, pero el cierre no era autónomo:
un ACK con pendientes, rework o paquete incompleto quedaba registrado y un
operador tenía que convertirlo manualmente en la siguiente ola de trabajo.

## Solución

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
- entrega aceptada con `course_id` y `topic_id`: abre `update_topic_registry`
  para que el conector oficial actualice el registro global de temas sin
  ampliar el write-set del agente de contenido;
- receipt rechazado: crea correccion conservando el trabajo recuperable.

La lógica vive en `modulos/orquesta-opes-director`. El servidor solo cablea
puertos y wakeups. El núcleo no importa OPES.

## Idempotencia

Cada job causal usa `idempotency_key` estable con fuente, artefacto, followup y
fase. Si el backend de `domain-work` expone `DomainWorkJobRecordSourcePortV0`,
el productor consulta antes de crear. Si no lo expone, delega en la idempotencia
del backend remoto.

## Wakeup

El ledger de artefactos del composition root está decorado para despertar:

- supervisor residente;
- Director residente.

La causa registrada es `domain_work_artifact_recorded`.

## No Cierre Falso

Un paquete final con estado `pendiente_continuar` no se interpreta como cierre.
Si no declara refs concretas, el productor crea el followup
`final-package-pendiente-continuar`.

## Registro Por Tema

El registro OPES se trata como trabajo causal propio. El productor crea
`update_topic_registry` cuando una entrega aceptada trae `course_id` y
`topic_id`; el job debe devolver `topic_registry_update` con `registry_action`,
`proposed_status`, `done_refs`, `pending_refs` y evidencias. Si el conector o la
herramienta oficial no está disponible, el job debe devolver bloqueo público y
comando exacto necesario, no cerrar el tema como listo.

El propio artefacto `topic_registry_update` queda excluido para evitar bucles.

## Pruebas

Comandos usados en la implementación:

```bash
go test -count=1 ./modulos/orquesta-opes-director
go test -count=1 ./cmd/orquesta-server -run 'OPESCausal|WakeupDomainWork|ResidentDirector'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'DomainWork|ResidentDirector|DocumentPlan'
go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-file ./modulos/orquesta-document-plan-expander ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director
go test -count=1 ./cmd/orquesta-server -run 'DomainWork|OPES|Wakeup|ResidentDirector|Stack'
```
