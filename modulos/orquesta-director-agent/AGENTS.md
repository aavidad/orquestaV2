# Contexto local: orquesta-director-agent

Lee este archivo antes de tocar este mini-proyecto.

## Reglas

- Este modulo define el contrato del agente director externo; no implementa el nucleo.
- El agente director es reemplazable: Codex, Claude, Gemini, local u otro conector deben poder producir el mismo DTO.
- El director gasta el minimo contexto posible: decide, enruta y delega; no lleva el peso de analisis, programacion, documentacion ni revision.
- Si una decision requiere trabajo pesado, el director debe pedir agentes/grupos especializados con contexto propio pequeno.
- No introducir referencias a proveedor, HOME, OAuth, credenciales, modelos concretos, transcripts ni prompts completos.
- El core no importa este modulo; los adaptadores validan el DTO y lo convierten a comandos publicos.
- Contexto pequeno: `AGENTS.md`, `README.md`, `docs/contratos.md`, `docs/tareas.md`, `docs/pruebas.md` y `docs/decisiones.md`.
- Funciones pequenas y ficheros manejables.

## Entrega

- Validar con `go test -count=1 ./modulos/orquesta-director-agent`.
- Si se cruza con workflow/runtime, la prueba debe vivir en `orquesta-e2e`.
