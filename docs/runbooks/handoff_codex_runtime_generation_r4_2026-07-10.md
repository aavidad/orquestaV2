# Handoff Codex runtime generation R4 — 2026-07-10

## Alcance

Reparación previa a deploy del commit `4a97c37e3a`, limitada a
`modulos/orquesta-runtime-codex-appserver`. No se operaron procesos, sesiones
tmux ni runtimes reales; los procesos hijos y el tmux usados por los tests son
fixtures efímeras propiedad de la propia suite.

## Contrato R4 cerrado

- El verde de una generación enlaza causalmente la identidad inmutable de la
  sesión (`session_id`, `session_created`, pane PID+starttime), el árbol de
  procesos y el inode del listener. El owner real se resuelve desde
  `/proc/net/unix` y `/proc/<pid>/fd`, se exige como descendiente del pane y se
  persiste como `socket_owner_pid` + `socket_owner_start_ref`.
- `waitForTmuxSocketV0` vuelve a comprobar sesión, token generacional, owner del
  socket, starttime y preflight después del CAS del marker antes de devolver
  verde. Un pane wrapper y un listener nativo descendiente no se confunden.
- `new-session` publica la identidad mediante `-P/-F` y recibe
  `ORQUESTA_CODEX_APP_SERVER_GENERATION_REF` como entorno de sesión observable.
  Un marker provisional puede recuperar por ese token una sesión creada antes
  de persistir la identidad completa y cerrarla por su ID exacto.
- Toda mutación de marker/socket exige una capacidad `flock` estable ligada al
  lease exacto, con parent privado, UID/modo e identidad inode verificadas. El
  CAS ya no depende de un mutex local y el rollback legacy no toca el marker
  generacional exacto.
- La limpieza usa quarantine/rename en el mismo parent privado, valida el inode
  tras el rename y restaura o conserva el residual privado si no puede probar
  identidad. No hace `os.Remove(socketPath)` ni borra sockets por glob.
- El único marker generacional válido es `socketPath + ".owner.json"`. Los
  markers legacy quedan como evidencia conservadora de conflicto, nunca como
  autorización para inferir `s.sock` o borrar `*.sock`.
- `kill-session` recibe exclusivamente el `session_id` persistido. Antes de
  matar revalida ID, created, pane PID+starttime; una sustitución por nombre no
  alcanza la sesión nueva.
- `has-session` solo traduce a ausente el `ExitError` de código 1 con mensaje
  tmux de sesión/servidor inexistente. Cualquier otro `ExitError` se propaga.

## Evidencia adversarial

- Procfs sintético exacto: wrapper pane -> wrapper intermedio -> owner nativo;
  también owner no descendiente rechazado.
- Sustitución inyectada inmediatamente antes de quarantine: la identidad nueva
  se conserva y la mutación devuelve conflicto.
- Sustitución entre revalidación y `kill-session`: se usa el ID viejo y la
  sesión sustituta permanece.
- Dos procesos Go reales compiten por el mismo CAS: exactamente un ganador y
  un conflicto. Stress: 40 repeticiones.
- Recuperación de marker provisional por token generacional y cierre por
  `session_id` exacto.
- `has-session` adversarial con `permission denied` no se convierte en
  not-found.

## Verificación ejecutada

Con `GOMODCACHE=/srv/orquesta-self/runtime/server-latest/go-mod-cache` en modo
lectura y caches efímeras dentro del módulo:

```text
go test -count=1 ./modulos/orquesta-runtime-codex-appserver
ok orquesta/modulos/orquesta-runtime-codex-appserver

go test -race -count=1 ./modulos/orquesta-runtime-codex-appserver \
  -run 'Test(MarkerCASV0MultiprocesoRealTieneUnSoloGanadorV0|QuarantineMarkerV0DetectaSustitucionAntesDeRenameV0|TmuxKillSessionV0SustitucionConservaSesionNuevaV0|SocketOwnerProcV0)'
ok orquesta/modulos/orquesta-runtime-codex-appserver

go test -count=40 ./modulos/orquesta-runtime-codex-appserver \
  -run '^TestMarkerCASV0MultiprocesoRealTieneUnSoloGanadorV0$'
ok orquesta/modulos/orquesta-runtime-codex-appserver

git diff --check
sin salida
```

`gofmt` se aplicó a todos los Go modificados. La ruta solicitada
`/srv/orquesta-self/runtime/test-cache` es de solo lectura para esta sesión; no
se alteraron permisos ni runtime. `/tmp` estaba al 98% y agotó su cuota durante
la primera compilación, por lo que no se borraron temporales ajenos y se usó
una cache efímera dentro del write-set, retirada al cierre.

El sandbox devuelve `EPERM` al crear listeners Unix. Las integraciones que
requieren `listen(AF_UNIX)` quedan explícitamente `SKIP` aquí y siguen activas
para un host Linux normal; el contrato de causalidad se cubre sin falso verde
mediante fixtures procfs y los tests de CAS/TOCTOU sí se ejecutaron.

No se creó commit ni se tocó `.git`.

estado_final: ready_for_operator_runtime_r4
