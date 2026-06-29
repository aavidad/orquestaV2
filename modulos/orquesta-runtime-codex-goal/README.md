# orquesta-runtime-codex-goal

Adaptador opt-in para usar Codex Goal como Director operativo de un trabajo
Orquesta.

Entrada:

```text
GoalWorkSpecV0
  -> CodexGoalStartPacketV0
  -> CodexGoalStarterPortV0
  -> GoalLaunchReceiptV0

GoalObservationRequestV0
  -> CodexGoalObservationRequestV0
  -> CodexGoalObserverPortV0
  -> GoalWorkResultV0
```

Este modulo no lanza procesos ni conoce la herramienta interna concreta para
crear u observar un goal. Solo prepara paquetes estables y llama a puertos
inyectados. La composicion que tenga acceso real a Codex Goal implementa esos
puertos.

La composicion actual `cmd/orquesta-server` debe usar
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux` para trabajo goal-first normal.
Ese wiring usa `codex app-server` y queda fuera de este modulo: arranca
`codex app-server --listen unix://<socket>` en una sesion `tmux` y observa por
WebSocket sobre socket Unix privado. `app_server_proxy` queda como
compatibilidad diagnostica condicionada a daemon/socket compatible; no es ruta
normal de self-programming ni de smokes vigentes.
El paquete conserva el gobierno externo de Orquesta: contexto, reglas,
write-set, tests, criterios de aceptacion, artefactos, evidencias, presupuesto y
politicas de cierre/rework.

Regla de cierre: un goal `complete` es una entrega del runtime, no cierre
aceptado de Orquesta. El cierre pasa por `GoalWorkClosureValidatorPortV0` o por
un validador de composicion equivalente.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal
```
