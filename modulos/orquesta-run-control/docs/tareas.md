# Tareas

## Hecho

- Definir `RunControlStateV0` y catalogo de estados.
- Definir puertos reader/writer para control de runs.
- Definir comandos puros `PauseRunV0`, `ResumeRunV0`, `StopRunV0` y `CancelRunV0`.
- Implementar `EvaluateRunControlV0`.
- Cubrir reglas de scheduling, dispatch, checkpoint, parada de agentes y terminalidad con tests unitarios.

## Pendiente fuera de alcance

- Adaptadores de almacenamiento.
- Adaptadores HTTP o MCP.
- Integracion con scheduler interno.
- Integracion con runtime o procesos de agentes.
- Mutacion real del workflow core.
