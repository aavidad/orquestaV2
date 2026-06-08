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

## Fronteras

- El nucleo no conoce Ollama, Codex, Claude, Gemini, OPES, OAuth, HOME, token,
  DB, rutas locales ni URLs de proveedor.
- La peticion publica de modelos no acepta `base_url`; la composicion configura
  endpoint, token y timeout.
- La gestion de modelos no decide que modelo usa una tarea. Esa decision queda
  en capacidad/scheduler/Director o en la composicion consumidora.
- El debate/voto no es mayoria simple: el Director consolida criterios,
  evidencias, coste, riesgo y calidad antes de aceptar o pedir rework.

## Pendiente Para Director Autonomo

- Ampliar `DirectorAutonomousOpsSnapshotV0` con runtime models, waits, olas,
  cohortes y arbol recursivo cuando esas fuentes entren por puertos publicos.
- Timeline causal completa con decisiones, replan/rework, cierres, bloqueos y
  arbol recursivo de agentes.
- Politica de seleccion/escalado de modelos por perfil de tarea y presupuesto,
  sin rails por palabras ni cortes automaticos recuperables.
