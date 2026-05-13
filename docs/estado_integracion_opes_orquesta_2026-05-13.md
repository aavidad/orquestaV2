# Estado Integracion OPES-Orquesta

Fecha: 2026-05-13.

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
- Guardas de smoke: los scripts no crean datos si no se exporta la variable de
  confirmacion correspondiente.

## Pendiente No Bloqueante

- Repetir prueba real completa de OPES creando un tema cuando OPES termine el
  trabajo actual. No lanzar mientras haya un temario en curso.
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
