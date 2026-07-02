# Incidencia: goal activo con thread systemError por creditos agotados

Fecha: 2026-07-02.

## Sintoma

En el remoto aislado `127.0.0.1:18787`, Orquesta lanzo un goal de
automejora y lo proyecto como `running`, pero el thread de Codex habia
terminado en segundos con `status=systemError`, sin mensaje final de agente y
sin escrituras en el worktree. El JSONL de la sesion contenia `token_count` con
`has_credits=false` y `balance=0`.

## Causa

Cuando `thread/goal/get` seguia devolviendo `active`, el observador confiaba en
ese estado y solo buscaba resultados durables o marcadores finales. No cruzaba
la lectura activa con `thread/read`, aunque el thread ya estuviera en
`systemError`.

## Cierre

El observador de goals activos lee ahora el estado del thread. Si el thread esta
en `systemError`, devuelve `blocked` con la causa operable correspondiente. Si
la ruta JSONL del thread contiene senal de creditos agotados, la causa publica
es `codex_app_server_goal_provider_limited`.

Pruebas:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0BloqueaGoalActivoSiThreadTerminaSinCreditosV0|TestServerCodexAppServerGoalBackendV0ObservaUsageLimitedConCausaOperableV0|TestServerCodexAppServerGoalBackendV0DiagnosticaThreadSystemErrorSinResultadoDurableV0'
go test -count=1 ./cmd/orquesta-server
go test -count=1 ./...
```
