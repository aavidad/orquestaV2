# Contexto Codex: orquesta-agent-process-registry

## Reglas comunes

- Contexto pequeno: este modulo solo define el contrato neutral del registro de procesos de agentes.
- Hexagonal: expone puertos y DTOs, no adaptadores operativos.
- No depende de nucleo, persistence, runtime, DB, proveedor, HOME ni OAuth.
- Si una decision afecta a otro modulo, emite `CONSULTA AL DIRECTOR`.

## Alcance

Contrato neutral para `run_id + agent_request_id -> process_ref`.

## Prohibido

- Importar `orquestacionnucleoapp`.
- Importar adaptadores concretos de persistence o runtime.
- Guardar rutas, PID, HOME, OAuth, provider, prompt, transcript, DB o DSN.
