# Auditoria de limpieza de ramas y worktrees - 2026-07-06

## Contexto

- Rama de prueba vigente: `trabajo/plataforma-agentes`.
- HEAD verificado: `0188739c3 docs: cola D1-D7 cerrada; D4 con dedupe causal en origen y auditoria justificada`.
- `git fetch --all --prune` ejecutado sin errores.
- `GOFLAGS=-buildvcs=false go build ./...` pasa en `0188739c3`.
- Binario de prueba generado fuera del repo:
  `/tmp/orquesta-builds/orquesta-server-0188739c3`.

## Limpieza segura ejecutada

- `git worktree prune` elimino solo metadatos de worktrees inexistentes:
  - `/tmp/orquesta-remote-dirty-review`
  - `/tmp/orquesta-sync.ZTSAb4`
- No se borraron ramas ni worktrees con contenido real.
- `.claude/settings.local.json` ya esta ignorado por la configuracion global
  del usuario. `.claude/worktrees/` se ignora ahora en `.gitignore` porque es
  un contenedor de worktrees auxiliares, no codigo fuente del repo principal.

## Ramas y worktrees clasificados

### Principal valida

- `trabajo/plataforma-agentes` esta alineada con `origin/trabajo/plataforma-agentes`.
- Es la rama que compila y debe usarse para la prueba del nuevo Orquesta.

### Conservar como evidencia o respaldo

- `claude/fix3-parking-runs-envenenados` (`cac171a3d`):
  - Aporta una variante del fix de parking de runs envenenados por presupuesto
    de eventos.
  - La rama principal ya contiene la solucion activa en torno a
    `run_oversized`, `run_oversized_events_budget` y la evidencia
    `evidence-ref-codex-supervisor-run-oversized-parked`.
  - `docs/auditoria_diseno_estructural_2026-07-06.md` cita esta rama como
    respaldo de Claude.
  - Recomendacion: no fusionar a ciegas; conservar hasta que Claude confirme
    que la implementacion principal cubre todo el caso.

- `pilot-t297` (`d47a3ec9d`):
  - Aporta un piloto G1 del wizard de programacion.
  - La rama principal ya contiene un wizard posterior y mas amplio.
  - Recomendacion: revisar solo ideas/fixtures utiles; no hacer merge directo,
    porque la rama esta muy por detras de la principal.

- `backup/pre-sync-trabajo-plataforma-20260703` (`0c59a88ed`):
  - Tiene 13 commits locales no contenidos en `trabajo/plataforma-agentes`.
  - Recomendacion: conservar/revisar como respaldo de sincronizacion previa; no
    borrar dentro de una limpieza automatica.

- `/home/alberto/Trabajo/orquesta-goal-worktree-180f5631`:
  - Worktree detached con muchos cambios sin commit.
  - Auditoria paralela de ramas lo clasifica como HEAD ya contenido en la base,
    pero con `89 M`, `1 D`, `9 ??` y diff aproximado de `90 files changed,
    1490 insertions, 937 deletions`.
  - Auditoria focal del worktree lo clasifica como mayoritariamente integrado,
    movido o superado. En particular, el codigo antiguo de
    `cmd/orquesta-server/codex_goal_app_server*` vive ahora sobre todo en
    `modulos/orquesta-runtime-codex-appserver`, por lo que no debe copiarse tal
    cual.
  - Aprovechable puntual detectado:
    - `modulos/orquesta-external-work-run`: respetar
      `input_fields.expected_artifact_type` antes de caer a `work_kind`.
    - `modulos/orquesta-goal`: conservar `LastResult` cuando el observador
      falla con issues.
    - `modulos/orquesta-app-codex-stack`: evitar reconciliar `running_stale` si
      hay marker goal-first vivo/ambiguo.
    - `modulos/orquesta-app-codex-stack`: endurecer receipt con `artifact_ref`
      exacto y tipo.
    - `modulos/orquesta-app-codex-stack`: evitar retry de `prepare_run` si hay
      proceso vivo/ambiguo registrado.
    - `modulos/orquesta-runtime-codex-goal`: revisar prompt de subagentes para
      regla `fork_context/full-history` y omision de overrides no necesarios.
  - Recomendacion: integrar solo esos fixes/testcases en una rama limpia contra
    `trabajo/plataforma-agentes`, archivar evidencias MD/JSON si siguen
    importando y borrar el worktree solo despues.

### Candidatas a archivar/eliminar tras confirmacion

- `trabajo/nueva-app-ayuda-opciones-tablas` (`889dfea9a`):
  - Tiene una punta local no contenida por SHA.
  - En la comprobacion local, `git log --right-only --cherry-pick` frente a
    `trabajo/plataforma-agentes` no mostro patch unico, lo que sugiere que el
    cambio podria estar absorbido por commits posteriores.
  - Recomendacion: conservar hasta una ultima comprobacion focal de docs/tests
    del wizard; no borrar en esta pasada.

- `autoprogramacion/orquesta-self-20260512-224047`:
  - Aparece como rama ya contenida en `trabajo/plataforma-agentes`.
  - Recomendacion: archivar o borrar worktree cuando no se necesite conservar
    la evidencia historica local.

- `backup/pre-platform-core-2026-05-13`, `backup/pre-remote-integration-20260630074332`
  y `master`:
  - Aparecen contenidas en `trabajo/plataforma-agentes`.
  - Recomendacion: conservar solo si se quieren como anclas historicas.

### No limpiar automaticamente

- `autoprogramacion/orquesta-20260512-120537`:
  - No esta fusionada en `trabajo/plataforma-agentes` y ademas tiene upstream
    remoto propio.
  - Recomendacion: no borrar desde esta limpieza.

## Proxima accion recomendada

### Extraccion selectiva realizada por Codex

- `modulos/orquesta-external-work-run`: integrado
  `input_fields.expected_artifact_type` como prioridad del contrato de
  artefacto, conservando tipos especificos de dominio y normalizando aliases
  conocidos como `completed_syllabus_package -> final_domain_package`.
- `modulos/orquesta-goal`: integrado guardado best-effort de `LastResult`
  cuando el observer falla pero devuelve issues accionables.
- `modulos/orquesta-app-codex-stack`: integrado match estricto de receipt:
  si el contrato declara `artifact_ref` y `artifact_type`, ambos deben coincidir.
- `modulos/orquesta-app-codex-stack`: integrado guard para no reconciliar
  `running_stale` cuando el run tiene disposicion goal-first viva o recuperable.
- `modulos/orquesta-app-codex-stack`: integrado guard para no crear retry de
  `prepare_run` si existe proceso registrado vivo o ambiguo del run anterior.
- `modulos/orquesta-runtime-codex-goal`: ya estaba integrado en la rama actual:
  el prompt incluye la regla `fork_context/full-history` y omision de
  `agent_type`, `model` y `reasoning_effort` cuando esos valores se heredan.

### Verificacion focal ejecutada

- `go test -count=1 ./modulos/orquesta-external-work-run`
- `go test -count=1 ./modulos/orquesta-goal`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-runtime-codex-goal`
- `go test -count=1 ./...`

### Limpieza pendiente

1. Confirmar con Claude si `claude/fix3-parking-runs-envenenados` queda como
   respaldo historico o si falta algun caso no cubierto por la rama principal.
2. Borrar o mover fuera del flujo activo las ramas ya
   supersedidas.
3. Borrar `/home/alberto/Trabajo/orquesta-goal-worktree-180f5631` solo despues
   de backup o confirmacion explicita: los fixes aprovechables ya estan
   extraidos, pero el borrado del worktree dirty es destructivo.
