# Bugs vivos de Orquesta - indice canonico

Actualizado: 2026-07-11 (auditoria real del attestor 208AA).
Mantenedor: Claude (revisor). Regla: UNA fila por bug vivo con su residual
exacto; el historial completo vive en
`docs/inventario_bugs_orquesta_2026-06-30.md` y NO se cuenta desde alli.
No reutilizar IDs. Al cerrar o abrir un bug, actualizar este indice en el
mismo commit.

## Vivos (nucleo)

| ID | Residual exacto que lo mantiene vivo | Siguiente accion |
| --- | --- | --- |
| BUG-ORQ-20260710-208A-D | patches focales verdes en local; un repro local 2026-07-11 elimino el backend pero el primer control no propago terminalidad, observe tuvo que reconciliar a `blocked` y shutdown publico `ready` sin salir | cerrar propagacion/actuador causal y probar por API que control + shutdown terminan sin segundo intento ni SIGINT propietario; repetir tras deploy |
| BUG-ORQ-20260710-208E | tooling drain/harness listo (F3-R2); falta receipt real `clean` de drain + dos pases amplios verdes. El perfil aislado exige `ORQUESTA_TEST_CACHE_ROOT` y `ORQUESTA_TEST_BATCH_ROOT` fuera de `/srv`; una revision local 2026-07-10 confirmo que Go 1.25 deja `GOMODCACHE` readonly y un runner desligado puede dejar helpers Unix bajo esa raiz | operador ejecuta drain + `orquesta_test_batches.sh` con ambas rutas aisladas; consolidar config y restaurar `chmod -R u+w` antes del cleanup, verificando que no quedan helpers por identidad de runtime |
| F5/identidad runtime | integrado en `2fe12f658` y completado localmente por D1 con `5f30973d7`: identidad de servidor y proyecto externo ya son distintos; closure accepted y shutdown listo | ejecutar drain/deploy gobernados y verificar por API el binario remoto, sin tocar `uso-app` |
| BUG-ORQ-20260710-208I | timeout parcial de observe coexistio con `invalid` durable; causa raiz no demostrada. F1 YA adoptado en observe/status/stats (etapa A, 2026-07-10): la superficie local ya no puede publicar running contradicho; la evidencia del incidente es del servidor remoto | repro por API contra servidor desplegado y correlacionar refs/tiempos con el veredicto causal publicado; no cerrable en local |
| BUG-ORQ-20260710-208S | goal-first residente ya cubre bloqueo/rework/parada por falta de progreso; el residual es `codex-launch-wave`, utilidad breakglass cuyo presupuesto depende de `codex-wave-status` | no usar esa utilidad como flujo productivo; si se conserva para operacion real, conectarla a una tarea/observador gobernado; evidencia en la [incidencia 208S](incidencias/incidencia_orquesta_ola_codex_sin_progreso_diagnostico_excesivo_2026-07-10.md) |
| BUG-ORQ-20260701-079 | solo frontera proveedor: cap duro pre-tool ante stdout crudo sin redireccion | esperar enforcement del proveedor o probe adversarial nuevo; no bloquea local |
| BUG-ORQ-20260704-165 / 20260701-065 | residual amplio de observabilidad/control lento con proveedor real; nucleo local cerrado | se paga con la adopcion completa del veredicto F1 + repro 208 tras deploy |
| BUG-ORQ-20260711-208Z | una instancia con `degraded_identity` bloquea correctamente trabajo, pero tambien rechaza `POST /api/v0/server/shutdown` con `server_work_launch_degraded_identity` | separar shutdown seguro del guard de lanzamiento; conservar idempotencia y demostrar que prepare-run sigue bloqueado. Evidencia en la [incidencia 208Z](incidencias/incidencia_orquesta_degraded_identity_bloquea_shutdown_2026-07-11.md) |
| BUG-ORQ-20260711-208AA | cerrado funcionalmente: un goal real termino con codigo y tests focales verdes, pero el attestor carecia de toolchain/cache y despues de `PATH`; Orquesta trataba errores operativos del attestor como test rojo y lanzaba rework inutil | `b9205ca71` integra preflight, snapshot verificado, `PATH` minimo y fallo de infraestructura sin rework; E2E `autoprog-attestor-e2e-autonomous-20260711` cerro `accepted` con dos receipts independientes `passed`. Evidencia en la [incidencia 208AA](incidencias/incidencia_orquesta_attestor_toolchain_infra_lanza_rework_2026-07-11.md) |
| BUG-ORQ-20260711-208AC | cerrado funcionalmente: el observador residente reutilizaba un timeout de dos segundos y cancelaba la atestacion independiente antes de que un test Go real terminase | `4a4ce945d` separa la ventana durable del residente (15 minutos) de la respuesta HTTP acotada; el E2E final cerro `accepted` sin intervencion ni rework. Residual documentado: scheduler concurrente de atestaciones para escala. Evidencia en la [incidencia 208AC](incidencias/incidencia_orquesta_attestor_cancelado_por_timeout_observador_2026-07-11.md) |

## Vivos (operativos, no de codigo)

| ID | Residual | Siguiente accion |
| --- | --- | --- |
| S14/deploy remoto | binario remoto vivo `9541e2f0...` anterior al codigo; flujo nuevo: remoto = solo destino de deploy | drain gobernado + `orquesta_server_deploy.sh` cuando el operador decida subir |
| S13/F4 artefactos versionados | conteo vivo: 61 con nombre de checkpoint/resultado; 68 al incluir cuatro fixtures eval y tres artefactos auxiliares de la familia. Codex app-server, Claude y Gemini ya escriben/leen primero recibos runtime por goal; el guard Git conserva el rechazo de nuevos no clasificados. Quedan proyeccion comun de procedencia, smoke real aislado y migracion de 58 movibles. Evidencia en la [incidencia S13](incidencias/incidencia_orquesta_s13_destino_recibos_runtime_2026-07-10.md). | probar proveedor real aislado y migrar los 58 movibles con commits gobernados |
| S12 limpieza envs remota | perfil remoto/secretos/defaults sin corte gobernado | corte separado tras deploy; no mezclar con drain |
| CODEX-HOME-TOKEN-INVALIDADO | auth Codex remota caducada | reauth del operador en servidor; hoy ademas cuota local agotada |
| BUG-ORQ-20260710-208T | retencion runtime sin cuota: `.orquesta-runtime/codex-waves` ocupa 16 GB y conserva 399 homes aislados; no se puede purgar sin clasificar evidencia | retenedor gobernado con inventario, dry-run, export/compresion, cuota y recibo; evidencia en la [incidencia 208T](incidencias/incidencia_orquesta_retencion_runtime_codex_waves_2026-07-10.md) |
| BUG-ORQ-20260701-058/066/075 (familia OPES) | solo residuales de campo: OPES temporal/preproduccion y proveedor real; local/fake cerrado | field test OPES temporal cuando se retome ese frente |

## Cerrados hoy (referencia rapida)

- BUG-ORQ-20260711-208AB: cerrado localmente por `97d1d913a`. Los tests
  derivados del grafo ya salen con `CommandRef` y hashes congelados; el repro
  API `autoprog-attestor-contract-repro-20260711` lanzo un goal real sin spec
  invalido. Evidencia en la [incidencia 208AB](incidencias/incidencia_orquesta_goal_spec_generado_invalido_2026-07-11.md).
- Incidencia lease generation conflict primer lanzamiento: cerrada con
  `01cb27d77` (selector tmux `=sesion:` + verificacion tri-estado +
  cleanup/shutdown degradan a evidencia residual). Smoke real local verde
  end-to-end verificado por revisor.
- BUG-ORQ-20260710-210 (guard textual stale del smoke compuesto).
- 208G (perdida causal de `checkpoint_started` en materializador) - reducido
  local, pendiente confirmacion independiente.
- BUG-ORQ-20260710-208L: cerrado por `5f30973d7` y D1 local retenido en
  `/tmp/orquesta-goal-first-app-server.ZAorvf`; el target externo no Git ya
  no se usa como identidad del binario.
- BUG-ORQ-20260710-208K: cerrado localmente por `4faf2fd2f`. La configuracion
  ausente ya materializa aliases tipados, la parcial falla cerrada y `xhigh`
  exige evidencia causal. Revisor: capacity, runtimes, stack completo y
  focales del servidor verdes; evidencia en la
  [incidencia 208K](incidencias/incidencia_orquesta_model_routing_fail_closed_legacy_2026-07-10.md).
- BUG-ORQ-20260710-208N: cerrado localmente. `orquesta-server` sin subcomando
  devuelve `comando requerido` con codigo 2, sin panic ni arranque implicito;
  cobertura en `TestRunMainV0WithoutCommandReturnsUsageError`.
- BUG-ORQ-20260710-208J: cerrado localmente por `08a993a3e`. El fallo del
  atestador persiste claim `failed` y `blocked/rework` sin reintento por
  polling; la activacion de atestador real queda como evidencia de 208H, no
  como bug duplicado.
- BUG-ORQ-20260710-208M: cerrado localmente. El lanzamiento rechaza runtime no
  observable y la ola permitida se observa `running` por status/tail y termina
  `stopped` por CLI; evidencia en la
  [incidencia 208M](incidencias/incidencia_orquesta_codex_wave_registry_untrusted_2026-07-10.md).
- BUG-ORQ-20260710-208O/208P/208Q: cerrados localmente. Se endurece la frontera
  Codex, se recupera Claude process y se redacciona argv sensible; evidencia en
  la [incidencia de regresiones D3](incidencias/incidencia_orquesta_d3_batch_regresiones_2026-07-10.md).
- BUG-ORQ-20260710-208R: cerrado localmente. Las expectativas de effort global
  se alinean con routing tipado por tarea, sin reintroducir herencia global.
- BUG-ORQ-20260710-208H: cerrado localmente. El atestador configurado se
  ejerce desde composicion y los dos pases aislados pasan sobre seis paquetes.
- Cobertura local Claude/Gemini goal-first 20260711: cerrada con
  `TestGoalFirstProcessBackendsE2EV0ReworkThenClose`. Ambos adaptadores de
  proceso fake recorren `StartAppDirectorV0`, resultado durable, rework por
  tests requeridos fallidos y cierre causal accepted. No equivale a smoke con
  proveedor real: Claude conserva ese smoke remoto pendiente y Gemini depende
  además de tier/autenticación operativos.
- BUG-ORQ-20260710-208U: cerrado localmente. La primera consolidacion de
  guardian introdujo un helper en `config_file_v0.go` y aliases de timeout;
  R4 demostro que volvian a romper T90 y la regla de unidad. `0b564992a`
  extrae el helper a fichero propio y deja los timeouts solo en
  `orquesta.config.json` tipado. T90, registry, guardian y presupuesto verdes;
  no hubo servidor, guardian, agente ni proveedor real.
- BUG-ORQ-20260711-208V: cerrado localmente. El primer test de `mcp-stdio`
  pidió `resources/read` con el nombre de una tool (`orquesta.status.v0`) en
  vez del URI canónico del recurso; el transporte devolvió correctamente
  `mcp_resource_not_found`, aunque el worker lo había declarado verde. El
  fixture usa ahora `MCPOperatorOperationsResourceURIV0`; focales de stdio,
  JSON-RPC HTTP y montaje MCP verdes. No hubo servidor, proveedor ni efecto
  externo.
- BUG-ORQ-20260711-208W: cerrado localmente. El primer adaptador CSV/JSON de
  ingesta validaba el array JSON pero no comprobaba datos posteriores al cierre
  del array. Se anadio rechazo `data_file_json_trailing_data` con prueba focal;
  se preservan hash, snapshot y limites. No hubo acceso fuera del root ni
  efecto externo.
- BUG-ORQ-20260711-208X: cerrado localmente. El guard MEJ-106 quedo rojo por
  dos overrides `*_TIMEOUT_MS` del guardian y tres nombres de IPC de fixture
  bajo el prefijo global, y el script de metricas dependia del locale heredado.
  `cec2848f3` fija `LC_ALL=C` y cubre el caso portable; `31ecba309` reduce la
  superficie y `0b564992a` termina la consolidacion tipada en 423/103. Durante la
  revalidacion se detecto que Bash puede avisar por locale invalido antes de
  ejecutar el script; el guard raiz fija su propio hijo a `LC_ALL=C`. Focales
  guardian/tool-file, `TestEnvVarsBudgetMEJ106V0` en C y no-C, y el test del
  script verdes. No se arranco servidor, guardian, agente ni proveedor.
- BUG-ORQ-20260711-208Y: cerrado localmente sin parche tmux especifico. El
  primer R4 vio `codex_app_server_tmux_generation_observation_transient`; 20
  focales secuenciales y el lote nuevo de dos pases no lo reprodujeron. Receipt
  acreditado: `/tmp/orquesta-r4-retry-batches/receipt.json`, 14 ejecuciones y
  `two_consecutive_passes_passed`. Reabrir con evidencia de identidad/lease y
  log del hijo si reaparece; detalle en la [incidencia R4](incidencias/incidencia_orquesta_r4_lotes_config_y_tmux_transient_2026-07-11.md).
- D3 / BUG-ORQ-20260710-208H: cierre formal local renovado por el receipt R4
  acreditado de 2026-07-11 sobre siete paquetes y dos pases. No equivale a
  smoke de proveedor, deploy remoto ni drain real.
- D3 configuracion/envs 20260710: cerrado localmente. La metrica separa
  produccion `425/425` y fixtures exclusivos `103/103`, con dos pases verdes.
- D3 local 2026-07-10: detenido tras el primer lote determinista para no
  gastar un segundo pase imposible. `TestEnvVarsBudgetMEJ106V0` fallo con
  538 variables frente a 513; recibo y log retenidos en
  `/tmp/orquesta-test-batches/receipt.json` y
  `/tmp/orquesta-test-batches/logs/pass-001-batch-001.log`. No hubo cambios
  de fuente ni procesos residuales.
- Revision routing 2026-07-10: los focales de capacity, runtimes y stack
  fueron verdes, pero la bateria amplia del worktree WIP se desligo antes de
  devolver resultado y dejo dos helpers Unix bajo
  `/tmp/orquesta-routing-review-env`; se pararon por identidad de esa ruta y
  se limpio despues de restaurar permisos de `GOMODCACHE`. No acredita el
  test amplio ni la integracion del routing; pertenece a 208E/F3.

## Regla de conteo

Bugs vivos de codigo del nucleo: los de la primera tabla (6 entradas, de las
cuales 208A-E son el mismo frente con residuales distintos). Todo lo demas es
operativo o de campo. Si una lectura antigua del inventario historico
contradice este indice, prevalece este indice.
