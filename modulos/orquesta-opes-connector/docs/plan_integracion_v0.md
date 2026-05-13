# Plan de integracion V0

Fecha: 2026-05-13.

## Propuesta de corte

La integracion debe seguir siendo opt-in. El primer corte implementa REST para
crear jobs y enviar artefactos sin acoplar OPES al nucleo de Orquesta.

## Cortes propuestos

1. Documentar contrato OPES-Orquesta en Orquesta.
2. Mantener `orquesta-domain-work` como contrato generico sin semantica OPES.
3. Crear el modulo `orquesta-opes-connector`.
4. Implementar adaptador REST configurable para jobs y artefactos.
5. En corte posterior, cablear el adaptador desde stack/MCP/web si procede.
6. En corte posterior, anadir MCP si aporta ventaja frente a REST.

## Criterios de aceptacion del futuro conector

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

- REST queda elegido para la primera implementacion; MCP queda pendiente.
- Falta definir donde se configuraran URL, credenciales y timeouts.
- Falta decidir como se representara `RegisterDelivery` en el flujo actual.
- Falta validar el tratamiento i18n si el conector expone texto visible en UI o
  CLI.

## No hacer en este corte

- No tocar OPES.
- No tocar persistencia.
- No modificar seleccion de agentes, modelos, capacidad, leases ni runtimes.
