# Tareas

## T12 smoke OPES real opt-in

Estado: bloqueado verificable 2026-05-27 para ejecucion real contra OPES
temporal.

El conector REST ya cubre la frontera publica necesaria para T12: listar jobs,
crear jobs y enviar artefactos con receipts validados. Los intentos cerrados
demostraron los tests focales y el recorrido fake historico del bridge hasta
`assemble_topic -> assembled_topic`; la secuencia vigente anade
`generate_audio_asset -> audio_asset` para accesibilidad auditiva.

Pendiente real:

- OPES temporal vivo;
- Orquesta temporal viva;
- `ORQUESTA_OPES_BASE_URL` y `ORQUESTA_BASE_URL` explicitos;
- `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de efectos;
- cuota/modelo confirmados para agente real.

Sin esas precondiciones, no crear nueva implementacion padre ni drenar colas
OPES. Registrar bloqueo verificable y conservar el comando del runbook.
