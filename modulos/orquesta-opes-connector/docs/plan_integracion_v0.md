# Plan de integracion V0

Fecha: 2026-05-13.

## Corte vigente

La integracion sigue siendo opt-in. El corte vigente implementa REST para
consultar jobs, leer bloques publicos de tema, crear jobs y enviar artefactos
sin acoplar OPES al nucleo de Orquesta.

## Cortes y estado

1. Cerrado: documentar contrato OPES-Orquesta en Orquesta.
2. Cerrado: mantener `orquesta-domain-work` como contrato generico sin
   semantica OPES.
3. Cerrado: crear el modulo `orquesta-opes-connector`.
4. Cerrado: implementar adaptador REST configurable para jobs, bloques publicos
   de tema
   y artefactos.
5. Cerrado para stack/HTTP actual: cablear el adaptador por
   `ORQUESTA_OPES_BASE_URL` y el tool generico `orquesta.domain_work.v0`.
6. Pendiente opcional: anadir cliente MCP especifico contra OPES si aporta
   ventaja frente a REST.

## Criterios de aceptacion del conector

- No accede a DB, ficheros ni rutas internas de OPES.
- No hardcodea SQLite, Postgres ni ningun backend.
- Solo usa REST/MCP publicados por OPES.
- Requiere `correlation_id`, `idempotency_key`, `requested_by=orquesta` y
  `external_refs`.
- Registra `job.id` y artefactos OPES como `evidence_refs`.
- No envia a OPES decisiones de agente, modelo, runtime, lease, sesion ni
  `tmux`.
- Puede funcionar apagado sin afectar a Orquesta.
- Tests de conector no llaman OPES real; usan `httptest`.

## Riesgos abiertos

- REST queda elegido como ruta productiva actual; MCP especifico de OPES queda
  pendiente solo si hace falta frente al tool generico `orquesta.domain_work.v0`.
- La URL y timeout REST se configuran por composicion con
  `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_OPES_TIMEOUT_SECONDS` y
  `ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS`.
- `RegisterDelivery` se representa hacia OPES mediante
  `DomainWorkArtifactSubmissionV0` y `POST /api/jobs/{id}/artifacts`, con
  ledger idempotente en el stack.
- Falta validar el tratamiento i18n si el conector expone texto visible en UI o
  CLI.

## No hacer en este corte

- No tocar OPES.
- No tocar persistencia.
- No modificar seleccion de agentes, modelos, capacidad, leases ni runtimes.
