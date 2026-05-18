# orquesta-app-codex-stack

Stack exterior de composicion para arrancar apps reales con Codex desde
`/nueva-app`, API REST o MCP en modo opt-in.

Este modulo no sustituye a `orquesta-app-gateway` productivo. Prepara una raiz
de composicion separada para inyectar `StartAppDirectorPortsV0` reales en el
servicio de director y conectar observacion de ACK/progreso de Codex usando los
adaptadores existentes.

Flujo previsto:

```text
web /nueva-app | REST | MCP
  -> orquesta-app-gateway
  -> orquesta-app-director-service.StartAppDirectorV0
  -> StartAppDirectorPortsV0 configurados
  -> orquesta-runtime-codex-delivery
  -> runtime Codex real opt-in
```

Principios:

- hexagonal: web/API/MCP solo ven puertos y DTOs publicos;
- composition externa: este modulo puede conocer conectores reales, el core no;
- opt-in estricto: sin variables/configuracion explicitas no arranca nada real;
- sin defaults de proveedor/modelo/DB: todo viene de configuracion;
- review gate por puerto explicito: la evidencia real de ficheros se inyecta
  desde el borde y el core solo recibe observaciones compactas;
- sin tocar app-gateway productivo: la ruta estable sigue siendo neutra;
- smokes reales acotados: se reutiliza la experiencia de
  `orquesta-runtime-codex-delivery`, con ACK, write-set y progreso por corte.

Estado actual: composition Go opt-in con test fake, smoke real desactivado por
defecto y supervisor Codex unitario para `launch -> sigue -> done` sobre el ciclo
normal de agentes.

El supervisor Codex vive en `codex_supervisor_v0.go`. Expone
`CodexSupervisorAgentLifecyclePortV0`: el adaptador real debe avanzar
`RunGlobalSupervisorV0`/`DrainRunV0`/`ContinueAppDirectorV0` y usar el outbox
`LaunchRuntimeAgent` existente si necesita arrancar otro Codex.

Arranque manual de un smoke configurado:

```bash
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealOptInV0 -count=1 -timeout 300s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=240 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-real/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/app-codex-stack-real/project/.orquesta-runtime \
./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

El ejemplo muestra el tipo de configuracion esperada, pero el stack no fija el
modelo, proveedor, DB ni directorios productivos. Si se usan, deben llegar por
configuracion de operador.

Validacion de este subtrabajo:

```bash
git diff --check -- modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexSupervisorV0'
```
