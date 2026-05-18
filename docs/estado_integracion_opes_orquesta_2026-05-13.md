# Estado Integracion OPES-Orquesta

Fecha: 2026-05-13.

Nota 2026-05-18: documento historico. Para estado vigente de OPES como
consumidor de Orquesta, usar
`docs/corte_opes_como_consumidor_orquesta_2026-05-18.md` y el runbook
`docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`. REST OPES es
la ruta funcional actual; `orquesta.domain_work.v0` es MCP generico ya
existente; un cliente MCP especifico contra OPES queda opcional.

## Resumen

Desde Orquesta no queda un bloqueo conocido para que OPES pida trabajos
externos y reciba artefactos. OPES debe seguir siendo la aplicacion de dominio:
decide temario, capitulos, bloques, visuales, persistencia y exportacion.
Orquesta solo orquesta agentes, modelos, sesiones, reintentos y entrega por
contratos publicos.

## Cerrado En Orquesta

- Contrato hexagonal `orquesta-domain-work` para `create_job` y
  `submit_artifact`.
- Conector REST OPES opt-in por `ORQUESTA_OPES_BASE_URL`.
- Endpoint REST/MCP fino `/api/v0/domain-work`.
- Flujo `/api/v0/external-work/run` para convertir un job OPES en run de
  Orquesta con microtarea externa.
- Entrega de `draft_content_block` como `content_block`.
- Entrega de revisiones como `block_revision`.
- Entrega de fuentes como `source`.
- Entrega de `generate_visual_asset` como `visual_asset`.
- Entrega de `summarize_*` como `topic_summary`.
- Entrega de `expand_topic_from_summary` como `topic_expansion_package`.
- Bridge opt-in `orquesta-server opes-drain-once` para leer ventanas pequenas
  de jobs OPES pendientes y convertirlos en runs por `/api/v0/external-work/run`.
- Hidratacion de `summarize_topic` por API publica
  `GET /api/topics/{topic_id}/blocks`; si no hay bloques, no se crea run.
- Contexto externo acotado pero ampliable por perfil `compact`, `standard` o
  `large`, con `large` por defecto para trabajos largos OPES.
- Estadisticas por `external_job_ref` para que OPES o web consulten progreso.
- Ledger de entregas para no reenviar artefactos ya enviados.
- Smokes reales protegidos por confirmacion explicita:
  - `scripts/smoke_opes_domain_work_real.sh`
  - `scripts/smoke_opes_external_work_agent_real.sh`
  - `scripts/smoke_opes_visual_asset_real.sh`

## Probado

- `go test -count=1 ./...` en Orquesta.
- Smoke REST directo OPES-Orquesta para `content_block`.
- Smoke con agente real para `draft_content_block`.
- Smoke REST directo OPES-Orquesta para `visual_asset`.
- Dry-run real contra cola OPES `summarize_topic` en
  `http://127.0.0.1:18080`, sin crear runs ni artefactos, hidratando 3 bloques
  para el primer job y 2 para el segundo.
- Guardas de smoke: los scripts no crean datos si no se exporta la variable de
  confirmacion correspondiente.

## Pendiente No Bloqueante

- Repetir prueba real completa de OPES creando un tema cuando OPES termine el
  trabajo actual o usando una instancia separada. No lanzar smokes contra la
  instancia que crea el temario real.
- Probar `generate_visual_asset` con agente real, no solo con entrega REST
  directa. Debe hacerse contra una instancia OPES de smoke.
- Revisar calidad de artefactos OPES generados por agentes reales: estructura,
  citas, continuidad editorial, ausencia de placeholders y utilidad para
  exportacion.
- Si OPES decide consumir MCP en vez de REST, probar el transporte MCP real con
  `orquesta.domain_work.v0`. El contrato ya existe, pero la integracion OPES
  actual usa REST.
- Mantener OPES separado como app de dominio. No importar codigo OPES dentro de
  Orquesta ni leer su DB o ficheros internos.

## No Hacer Mientras OPES Trabaja

- No parar procesos OPES.
- No ejecutar smokes contra la instancia que crea el temario real.
- No crear topics, chapters, jobs ni artifacts de prueba en la instancia activa.
- No modificar ficheros de OPES desde esta sesion de Orquesta.

## Drenar Cola OPES

Dry-run seguro:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_DRY_RUN=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=2 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=summarize_topic \
go run ./cmd/orquesta-server opes-drain-once
```

Ejecucion real, solo con Orquesta server levantado:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=3 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=summarize_topic \
go run ./cmd/orquesta-server opes-drain-once
```

El bridge no cambia OPES directamente: solo convierte jobs pendientes en runs.
OPES se completa cuando Orquesta entrega el artefacto por el conector publico.

## Siguiente Paso Recomendado

Esperar a que OPES termine su temario actual y revisar sus artefactos desde la
frontera publica. Si falla, diagnosticar primero si el problema es:

- contrato OPES-Orquesta;
- calidad del paquete de contexto que OPES entrega;
- seleccion de modelo/esfuerzo;
- perdida o timeout del agente;
- validacion editorial posterior de OPES.

Solo despues de aislar la causa se debe programar. No parchear un fallo puntual
sin entender que frontera lo produjo.
