# Registro de rails 2026-05-24

Objetivo: que los rails esten ordenados y revisables, sin listas copiadas por
comando ni ficheros sueltos sin propietario.

## Politicas comunes

- `modulos/orquesta-rails/text_policy_v0.go`: rail comun de texto sensible.
  Permite vocabulario operativo opaco (`runtime`, `provider`, `model`, `git`,
  `db`, `sql`, `codex`) y corta solo patrones que parecen llevar valor sensible
  (`api_key=`, `client_secret=`, `authorization:`, `bearer `, `-----BEGIN`).
- `modulos/orquesta-rails/filesystem_policy_v0.go`: rail estricto comun de
  filesystem. Rechaza rutas absolutas, `..`, HOME/variables, URLs, rutas
  Windows ambiguas, caracteres de control y comandos destructivos conocidos.
  Estado vigente 2026-06-02: queda dormido como el resto de `orquesta-rails`;
  no veta rutas ni comandos mientras `ORQUESTA_RAILS_MODE` este normalizado a
  `offline`.
- 2026-05-24: los rails de `detalle_prohibido`,
  `detalle_operacional_prohibido` y `secreto_detectado` quedan desactivables por
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`. El servidor Orquesta usa `off` por
  defecto si el operador no fija un valor explicito, para no bloquear
  autoprogramacion por falsos positivos. Las listas y tests estrictos se
  conservaban como reactivables con `ORQUESTA_DETAIL_PROHIBITED_RAILS=on`; esta
  foto queda superada por la decision del 2026-06-02.
- 2026-06-02: la regla vigente cambia para produccion: los rails de detalle
  operativo/local quedan offline por defecto y no bloquean entregas aunque
  existan listas, scopes o `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` heredado.
  Referencias a runtime, provider, modelo, ficheros de control, HOME usado como
  diagnostico, receipts o evidencia `ref_only` quedan como evidencia util.
  No hay reactivacion por env: `ORQUESTA_RAILS_MODE=enforced` se normaliza a
  `offline` hasta nueva orden. Cualquier vuelta a bloqueo exige tarea futura,
  owner claro, cambio explicito de codigo/configuracion y prueba viva.
  La autoridad de decision queda en el Director o agente orquestador: el rail
  produce senal, no veto. Si una regla blanda entorpece trabajo valido, el
  Director/orquestador puede ignorarla, retirarla del flujo o convertirla en
  tarea de mejora.

## Adaptadores actuales

- `modulos/orquesta-core-workflow/sensitive_detail_rails_v0.go`: wrapper local
  del core hacia `orquesta-rails`, para no copiar listas dentro de los
  validadores de workflow.
- `modulos/orquesta-director-agent/director_decision_helpers_v0.go`: usa
  `orquesta-rails` para el mismo rail de texto sensible en decisiones JSON del
  director-agent.
- `modulos/orquesta-runtime-worktree/verify_v0.go`: verifica efectos reales por
  snapshot y bloquea `removed_path` como rail estricto de no borrado aunque el
  fichero estuviera dentro del write-set.
- `modulos/orquesta-runtime-codex/codex_profile_v0.go`: exige sandbox
  `workspace-write`; `danger-full-access` queda fuera de la ejecucion normal
  porque rompe el rail estricto de no salir del workdir. Tambien rechaza
  `extra_args` capaces de reabrir `--add-dir`, `--sandbox` o `-C`.

## Rails especificos pendientes de ordenar

Estos rails pueden ser legitimos porque protegen fronteras concretas, pero hay
que revisarlos con matriz externa antes de endurecer o relajar:

- `modulos/orquesta-context/context_bundle_validation_v0.go`
- `modulos/orquesta-persistence/outbox_ledger_helpers_v0.go`
- `modulos/orquesta-persistence/persistence_repository_helpers_v0.go`
- `modulos/orquesta-director-scheduler/scheduler_tick_validation_v0.go`
- `modulos/orquesta-director/outbox_dispatch_cycle_validation_v0.go`
- `modulos/orquesta-core-replanner/replan_proposal_validation_v0.go`
- `modulos/orquesta-app-change-director-source/readiness_v0.go`
- `modulos/orquesta-runtime-required-test/local_command_executor_v0.go`
- `modulos/orquesta-observability/orquesta_event_types_v0.go`

## Regla de orden

- rail comun reutilizable: paquete `orquesta-rails`, fichero por familia;
- rail de frontera: queda en su modulo, pero documentado aqui;
- rail que bloquee ejecucion real por falso positivo: primero test externo,
  luego apertura permisiva, despues ajuste fino;
- nunca copiar la misma lista en varios comandos.

## Pendiente de automejora

- Crear matrices externas por rail antes de volver a endurecer: runtime,
  runtime-codex ACK, core-workflow, context, scheduler, director, persistence,
  capacity, observability, leases y concurrency.
- Reintroducir cortes solo por tarea explicita y despues de comprobar que no
  cortan refs operativas razonables, alias, datos reutilizables ni diagnostico
  necesario.
- Mantener evidencia cruda en estado/log local privado; no sustituirla por
  `detalle_prohibido` sin guardar el detalle original.
