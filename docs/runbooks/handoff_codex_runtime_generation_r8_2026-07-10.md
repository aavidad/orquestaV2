# Handoff Codex runtime generation R8 — 2026-07-10

## Alcance

Corrección local de los rojos independientes posteriores a R7. No se usó el
servidor remoto, `uso-app` ni otras aplicaciones, y no hubo push o deploy.

## Causa causal y parche

El helper arrancaba `sh -c "sleep 30"` y escribía en `owner.pid` el PID recién
creado de `sh`. Ese wrapper puede retener el FD mientras su hijo `sleep` es la
hoja que lo conserva; el resolver seleccionaba correctamente esa hoja causal.
El fixture arranca ahora `sleep` directamente y vuelve a exigir igualdad exacta
con el PID registrado, sin relajar el resolver productivo.

El paquete completo encontró además que `tmux` sin servidor puede responder
`error connecting to ... (No such file or directory)`. La clasificación acepta
solo esa forma exacta como sesión ausente; errores de permisos u otros fallos
siguen cerrando con `codex_app_server_tmux_has_session_failed`.

## Verificación

Pasaron sin `SKIP`:

```text
go test -race -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestSocketOwnerV0|TestSocketOwnerProcV0|TestEnsureV0AdoptaGeneracionExactaCuandoTmuxDesapareceV0'
go test -count=50 ./modulos/orquesta-runtime-codex-appserver -run '^TestSocketOwnerV0AceptaDescendienteDeWrapperV0$'
go test -count=20 ./modulos/orquesta-runtime-codex-appserver -run 'TestTmuxHasSessionV0|TestCodexAppServerTmuxBackendV0ForcedStopPreservaProcesoSinIdentidadPersistidaV0|TestSocketOwnerV0AceptaDescendienteDeWrapperV0'
go test -count=1 -json ./modulos/orquesta-runtime-codex-appserver
```

La salida JSON del paquete completo no contiene acciones `skip`. También
pasaron `gofmt` y `git diff --check` del write-set.

estado_final: ready_for_operator_runtime_generation_r8
