# Handoff cierre de sesion Orquesta 2026-07-02

Fecha: 2026-07-02

## Estado Git

- Rama remota de trabajo: `origin/trabajo/plataforma-agentes`.
- Local integrado y empujado hasta: `777e027c Separa code-home tmux app-server`.
- Worktree local usado para integracion: `/tmp/orquesta-rootfix.y9E0W9`.
- El arbol local queda limpio en `777e027c`.
- Agente remoto sigue vivo en
  `/srv/orquesta-self/runtime/audit-13611445`.

## Commits principales cerrados en esta tanda

- `7f12de3f Detecta artefactos OPES fuera de write-set`.
- `19922f4d Centraliza cleanup de smokes aislados`.
- `24de427c Extiende cleanup comun a wrappers restantes`.
- `0567e977 Evita runtime manual en inicio agente`.
- `777e027c Separa code-home tmux app-server`.

## Verificacion

- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
  verde para el detector OPES fuera de write-set.
- `go test -count=1 ./cmd/orquesta-server` verde tras el split T90.
- `go test -count=1 ./...` verde en `777e027c`.
- T90 ya no falla por `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go`:
  el fichero bajo de 994 a 760 lineas y el bloque `codex-home` quedo en
  `cmd/orquesta-server/codex_goal_app_server_tmux_code_home_v0.go`.

## Numeros actuales del inventario

- Total bugs funcionales inventariados: 134.
- Cerrados: 124.
- Abiertos: 10.

Abiertos:

- `BUG-ORQ-20260701-058` OPES/contrato de calidad.
- `BUG-ORQ-20260701-065` Shutdown/Goal backend.
- `BUG-ORQ-20260701-066` Goal-first/OPES cierre y observacion.
- `BUG-ORQ-20260701-073` Goal-first/timeout tras checkpoint.
- `BUG-ORQ-20260701-075` Goal-first/QA de artefactos parciales.
- `BUG-ORQ-20260701-076` Shutdown forzado/app-server residual.
- `BUG-ORQ-20260701-077` Startup/base_url y app-server residual.
- `BUG-ORQ-20260701-079` Goal-first/sin checkpoint temprano y salidas gigantes.
- `BUG-ORQ-20260701-085` Goal-first/write-set y recibo de dominio.
- `BUG-ORQ-20260701-088` Goal-first/status/shutdown alto consumo.

## Agente remoto

El agente remoto quedaba trabajando al cierre de esta sesion. Estado observado:

- HEAD remoto/origin: `777e027c`.
- Cambio vivo remoto: `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`.
- Proposito del cambio remoto: anadir guarda automatica para que scripts que
  arrancan servidor temporal con `server_pid="$!"` y `ORQUESTA_SERVER_ADDR`
  usen `smoke_shutdown_orquesta_server`.
- No cortar ese agente ni borrar sus untracked generados; debe terminar,
  probar, commitear y hacer fetch/rebase antes de push si origin avanza.

## Siguiente accion recomendada

1. Dejar terminar el cambio remoto de guarda de wrappers.
2. Traer su commit localmente y repetir `go test -count=1 ./cmd/orquesta-server`
   y, si no hay prisa, `go test -count=1 ./...`.
3. Recalcular inventario; si BUG-077 queda cubierto solo por wrappers pero
   siguen residuales reales de startup/backend, mantenerlo abierto con avance.
4. Continuar con los abiertos de runtime alto consumo/shutdown
   (`065`, `073`, `076`, `079`, `088`) y los de OPES largo (`058`, `066`,
   `075`, `085`) sin tocar produccion.
