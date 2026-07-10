# Handoff F1: reconciliador causal de estado vivo

Fecha: 2026-07-10.

## Alcance y excepcion operativa

Se completo F1 con Codex directo por el bloqueo declarado del binario Orquesta
viejo. No se lanzaron goals, runtimes, deploys ni procesos persistentes. No se
uso `codebase-memory-mcp`. El trabajo quedo limitado al write-set autorizado y
no se tocaron ni revirtieron cambios ajenos, `checkpoint_started`,
`orquesta_goal_result`, `orquesta-goal` o proveedores/runtime.

## Cambios

- `orquesta-estado-vivo` contiene una unica autoridad pura,
  `DerivarVeredictoCausalV0`, que reconcilia estado persistido, resultado
  durable e identidad/liveness runtime.
- El contrato distingue `running_confirmed`, `terminal_by_artifact`,
  `process_dead_state_stale`, `divergent_needs_repair` e `indeterminate`.
  Timeout u observacion incompleta no significan proceso muerto, terminal ni
  running; identidad ausente/no coincidente no cae a precedencia legacy.
- `NodoCicloVidaV0` transporta el `VeredictoCausalV0` completo y conserva
  `Fase` como compatibilidad aditiva. Su JSON incluye `veredicto_causal`
  tipado, clase, razon, flags, identidades y evidence refs.
- La evidencia real-forma de procesos aporta scope `goal_execution`, identidad
  de proceso, generacion, intento/resultado de observacion y mismatch tipado.
  Un error de snapshot se conserva como observacion indeterminada sin filtrar
  el error privado.
- MCP consume el veredicto del nodo: `observe` y `director/stats` bloquean
  divergencia/indeterminacion y eliminan cualquier `running` heredado que el
  reconciliador no confirme. Server deriva sus contadores desde el veredicto y
  expone por API contadores aditivos para las clases accionables
  `indeterminate` y `divergent_needs_repair`, dentro del presupuesto vigente
  del diagnostico compacto.
- Las pruebas antiguas con `ProcesoVivo=true` sin scope, identidad ni snapshot
  se migraron a la forma causal o pasaron a esperar indeterminacion, evitando
  conservar falsos verdes.

## Cobertura

Quedan cubiertos:

- snapshot real-forma vivo y muerto;
- store/resultado terminal con runtime vivo;
- timeout/observacion incompleta indeterminada incluso junto a terminal;
- identidad ausente, generacion no coincidente y varios procesos validos del
  mismo run;
- replay determinista/idempotente;
- transporte JSON tipado, mapeo MCP y contadores de API server;
- fronteras neutrales del modulo y superficies de status.

## Verificacion

Todas las ejecuciones finales usaron `TMPDIR`, `GOTMPDIR`, `GOCACHE`,
`GOMODCACHE` y `GOPATH` aislados bajo
`/srv/orquesta-self/runtime/tmp-deploy/orquesta-f1-estado-vivo`, con
`GOPROXY=off`. La cache de modulos aislada se sembro desde la cache local
existente porque el sandbox no permite DNS.

Comandos finales requeridos:

```text
go test -count=1 ./modulos/orquesta-estado-vivo
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-mcp
go test -count=1 ./modulos/orquesta-server
go test -count=1 . -run 'Test(EstadoVivoStatusSurfacesDoNotImportStateStoresDirectly|NeutralOrchestrationPackagesDoNotImportProductAdapters)$'
git diff --check
```

Resultado final: las cinco pruebas quedaron `ok` (`estado-vivo`,
`app-codex-stack`, `mcp`, `server` y las dos fronteras root) y
`git diff --check` no produjo errores. La suite server se repitio verde despues
de ajustar los contadores accionables al presupuesto existente.

## Riesgos y limites

- No hubo repro remoto ni runtime vivo por prohibicion expresa de este corte;
  la evidencia es offline/fake-runtime con formas reales de snapshot.
- El arbol estaba sucio y recibio cambios ajenos concurrentes fuera del
  write-set. Se preservaron y se excluyeron de esta implementacion y revision.
- La primera prueba aislada en `/tmp` fallo antes de compilar por cuota de
  disco; se retiro solo el temporal propio y las verificaciones finales se
  ejecutaron en la ruta aislada indicada arriba.

estado_final: ready_for_operator_f1

## Revalidacion local independiente

Tras aislar F1 sobre la rama local integrada, pasaron de nuevo los cuatro
paquetes, las dos fronteras raiz y los tests F1 de server con `-race -count=20`.
El intento `-race` sobre todo `orquesta-server` descubrio una carrera previa y
reproducible tambien en main entre `memoryStateStoreV0` y el test async de
presupuesto idle. Queda en un frente separado; no pertenece al write-set F1 ni
se declara verde aqui.
