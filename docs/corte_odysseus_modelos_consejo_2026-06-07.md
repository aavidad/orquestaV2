# Corte Odysseus Modelos Consejo 2026-06-07

## Decision

Se asimilan de Odysseus dos ideas utiles, sin importar su arquitectura ni
convertir Orquesta en una app clonica:

- debate estructurado entre agentes/modelos, con voto y decision final del
  Director;
- gestion operativa de modelos locales/cloud desde una UI o API, por puerto
  opt-in.

## Implementacion Vigente

- Consejo/voto queda expresado como skill transversal:
  `skills/orquesta-revision-consejo-votacion/SKILL.md`.
- Direccion de agentes queda expresada como skill:
  `skills/orquesta-director-agentes/SKILL.md`.
- Ordenacion de trabajo queda expresada como skill:
  `skills/orquesta-ordenacion-trabajo/SKILL.md`.
- Gestion de modelos queda expresada como puerto neutral:
  `RuntimeModelManagerPortV0`.
- Ollama entra solo por adaptador:
  `modulos/orquesta-runtime-ollama`.
- MCP/HTTP expone:
  `orquesta.runtime.models.v0` y `POST /api/v0/runtime/models`.
- Primer corte de cockpit operativo:
  `DirectorAutonomousOpsSnapshotV0` viaja en `orquesta.director.stats.v0` y
  `orquesta.autoprogramming.status.v0`; `/ops` lo consume para la llamada
  visible del Director.
- Director residente opt-in cableado en servidor el 2026-06-08:
  `cmd/orquesta-server` inyecta una implementacion real de
  `ResidentDirectorPortV0` sobre el stack Codex en los commits `8944ca9f` y
  `f619e899`. Esto deja servidor, reentrada y cierre externo conectados; no
  declara todavia smoke largo real.

## Fronteras

- El nucleo no conoce Ollama, Codex, Claude, Gemini, OPES, OAuth, HOME, token,
  DB, rutas locales ni URLs de proveedor.
- La peticion publica de modelos no acepta `base_url`; la composicion configura
  endpoint, token y timeout.
- La gestion de modelos no decide que modelo usa una tarea. Esa decision queda
  en capacidad/scheduler/Director o en la composicion consumidora.
- El debate/voto no es mayoria simple: el Director consolida criterios,
  evidencias, coste, riesgo y calidad antes de aceptar o pedir rework.
- `SkillRefs` debe ser contrato neutral de refs opacas. No debe transportar
  rutas locales, proveedor, modelo, HOME, OAuth, token ni contenido completo de
  una skill en el core.

## Pendiente Para Director Autonomo

- Primer loop offline probado el 2026-06-08:
  `RunResidentDirectorBriefingLoopV0` itera briefing, ejecuta acciones seguras y
  vuelve a pedir briefing hasta idle/cierre, accion externa pendiente, falta de
  progreso, necesidad de Director o presupuesto operativo. No es daemon ni
  proveedor real.
- Primer alojamiento residente conectado el 2026-06-08 en `orquesta-server` por
  `ResidentDirectorPortV0`: opt-in explicito, anti-solape, coalescing, panic
  durable, status publico, self-watchdog consciente del Director y adaptador
  real en `cmd/orquesta-server` sobre el stack Codex. Pendiente: smoke largo
  real del residente con servidor temporal y evidencias durables.
- Materializar `SkillRefs` como contrato Go neutral y transportar esas refs de
  perfil/tarea hasta runtime/packet/prompt sin meter proveedores en el nucleo.
- Activar consejo/votacion en el ciclo residente cuando existan varias opciones
  comparables: lanzar rondas, esperar por cohorte/ola, recoger votos/evidencias
  y dejar que el Director acepte, pida rework o replantee. La skill existe; el
  wiring live no debe darse por probado hasta tener test/smoke propio.
- Ampliar `DirectorAutonomousOpsSnapshotV0` con runtime models, waits, olas,
  cohortes y arbol recursivo cuando esas fuentes entren por puertos publicos.
- Timeline causal completa con decisiones, replan/rework, cierres, bloqueos y
  arbol recursivo de agentes.
- Politica de seleccion/escalado de modelos por perfil de tarea y presupuesto,
  sin rails por palabras ni cortes automaticos recuperables.
