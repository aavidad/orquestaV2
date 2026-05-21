# Corte tests requeridos y smoke de programacion - 2026-05-21

## Objetivo

Avanzar Orquesta para que el Director Operativo pueda programar nucleo/director
con tests requeridos durables, sin meter runtime real dentro del nucleo y sin
mezclar OPES como producto base.

## Cambios subidos

- `6339115 Integrar runner de tests requeridos del director`
  - `orquesta-orchestration-core` incorpora `RequiredTestRunnerV0`.
  - El runner ejecuta por puerto `RequiredTestCommandExecutorPortV0`.
  - Guarda `RequiredTestEvidenceV0` causal e idempotente por
    `RequiredTestEvidenceWriterPortV0`.
  - `app-director-service` lo invoca desde `run_required_tests` si no existen
    evidencias causales ya persistidas.

- `7eaf574 Anadir adaptador local de tests requeridos`
  - Nuevo modulo `modulos/orquesta-runtime-required-test`.
  - Ejecuta comandos reales solo por allowlist inyectada.
  - No ejecuta shell, no hereda entorno y devuelve refs relativas de artefactos.
  - Queda fuera del nucleo y fuera de `orquesta-runtime` base para evitar ciclos
    de imports.

- `125b0ee Cablear tests requeridos opt-in en el stack`
  - `orquesta-app-codex-stack` acepta `RequiredTestRunner` inyectado.
  - `cmd/orquesta-server` lo cablea solo con
    `ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED=1`.
  - Requiere allowlist mediante `ORQUESTA_REQUIRED_TEST_GO_COMMAND` o
    `ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS`.
  - Apagado por defecto.

## Verificacion de codigo

Ejecutado antes de los commits:

```bash
git diff --check
go test -count=1 ./...
```

Resultado: verde.

## Smoke real de programacion con agentes y subagentes

Directorio temporal purgado antes de ejecutar:

```text
/tmp/orquesta-real-smoke-programacion-subagentes-20260521
```

Comando principal:

```bash
go run ./cmd/orquesta-server codex-launch-director-wave \
  --wave-ref real-programming-subagents-20260521 \
  --project-dir /tmp/orquesta-real-smoke-programacion-subagentes-20260521/project \
  --runtime-dir /tmp/orquesta-real-smoke-programacion-subagentes-20260521/runtime/real-programming-subagents-20260521 \
  --command "$(command -v codex)" \
  --agents 2 \
  --allow-recursive-delegation \
  --max-delegation-depth 1 \
  --max-subagents-per-agent 2 \
  --write-set "." \
  --required-tests "go test ./..." \
  --reasoning-effort high \
  --sandbox danger-full-access \
  --approval-policy never \
  --purge-runtime \
  --objective-file /tmp/orquesta-real-smoke-programacion-subagentes-20260521/objective.md
```

Resultado observado:

- 2 agentes padre lanzados.
- 4 subagentes hijos lanzados.
- 6 procesos Codex reales terminaron solos.
- No se corto ningun proceso.
- El repo Orquesta quedo limpio.
- El proyecto temporal genero codigo Go bajo `internal/smoke`.
- Se generaron 6 artefactos markdown bajo `smoke_outputs/`.

Validacion del proyecto temporal:

```bash
cd /tmp/orquesta-real-smoke-programacion-subagentes-20260521/project
go test -count=1 ./...
```

Resultado:

```text
ok smoke.local/orquesta-real-smoke/internal/smoke 0.002s
```

## Pendiente

- El smoke directo `codex-launch-director-wave` demuestra lanzamiento real de
  agentes/subagentes, pero no es todavia el ciclo durable completo
  `wait -> review -> run_required_tests -> replan_or_close`.
- Falta ejecutar un smoke de servidor/director donde el `RequiredTestRunner`
  opt-in genere `RequiredTestEvidenceV0` dentro del `PlanState` real.
- Falta replan negativo automatico ante `required-tests-failed` o runner invalido.
- Quedan duplicidades/historico en comandos directos de wave que conviene
  clasificar, no borrar.
