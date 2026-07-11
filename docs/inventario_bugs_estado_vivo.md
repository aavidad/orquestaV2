# Bugs vivos de Orquesta - indice canonico

Actualizado: 2026-07-11 (cierre local de nucleo R4).
Mantenedor: Claude (revisor). Regla: UNA fila por bug vivo con su residual
exacto; el historial completo vive en
`docs/inventario_bugs_orquesta_2026-06-30.md` y NO se cuenta desde alli.
No reutilizar IDs. Al cerrar o abrir un bug, actualizar este indice en el
mismo commit.

## Vivos (nucleo)

No quedan bugs de codigo local conocidos tras R4. Esto no declara desplegado el
remoto ni cierra validaciones de proveedor/campo.

## Vivos (conectores)

| Residual | Siguiente accion |
| --- | --- |
| El reconciliador external-work ya falla cerrado y usa autoridad causal, pero el caller productivo actual solo alimenta la ruta de run stale; ACK, artefacto y estado de dominio siguen cubiertos por contrato/test directo, no por wiring real | al retomar conectores, cablear las señales desde la composicion external-work y añadir smoke temporal; no reabrir el nucleo mientras siga fail-closed |

## Vivos (operativos, no de codigo)

| ID | Residual | Siguiente accion |
| --- | --- | --- |
| BUG-ORQ-20260710-208A-D | codigo local cerrado por R4: control, cleanup, replay parcial y shutdown pasan focales, suite completa y `-race`; falta smoke API local post-commit y repeticion tras deploy | ejecutar una sola orden de control y un solo shutdown contra backend aislado; despues validar remoto gobernado |
| BUG-ORQ-20260710-208E | tooling drain/harness listo (F3-R2); falta receipt real `clean` y dos pases amplios en el entorno de despliegue | operador ejecuta drain y lotes aislados cuando se retome remoto |
| F5/identidad runtime | codigo local cerrado; falta verificar identidad del binario desplegado | ejecutar drain/deploy gobernados sin tocar `uso-app` |
| BUG-ORQ-20260710-208I | evidencia original solo remota; local ya falla cerrado | repro por API tras desplegar el binario vigente |
| BUG-ORQ-20260710-208S | residual de utilidad breakglass, fuera del flujo productivo | no usarla como flujo principal; gobernarla si se conserva |
| BUG-ORQ-20260701-079 | frontera de enforcement del proveedor | esperar cap del proveedor o probe adversarial nuevo |
| BUG-ORQ-20260704-165 / 20260701-065 | residual con proveedor lento real; nucleo local cerrado | revalidar tras deploy |
| S14/deploy remoto | binario remoto vivo `9541e2f0...` anterior al codigo; flujo nuevo: remoto = solo destino de deploy | drain gobernado + `orquesta_server_deploy.sh` cuando el operador decida subir |
| S13/F4 artefactos versionados | conteo vivo: 61 con nombre de checkpoint/resultado; 68 al incluir cuatro fixtures eval y tres artefactos auxiliares de la familia. Codex app-server, Claude y Gemini ya escriben/leen primero recibos runtime por goal; el guard Git conserva el rechazo de nuevos no clasificados. Quedan proyeccion comun de procedencia, smoke real aislado y migracion de 58 movibles. Evidencia en la [incidencia S13](incidencias/incidencia_orquesta_s13_destino_recibos_runtime_2026-07-10.md). | probar proveedor real aislado y migrar los 58 movibles con commits gobernados |
| S12 limpieza envs remota | perfil remoto/secretos/defaults sin corte gobernado | corte separado tras deploy; no mezclar con drain |
| CODEX-HOME-TOKEN-INVALIDADO | auth Codex remota caducada | reauth del operador en servidor; hoy ademas cuota local agotada |
| BUG-ORQ-20260711-208AD | smoke local de autoprogramacion: `prepare-run` acepto el goal, pero el receipt quedo `invalid` por `codex_app_server_tmux_has_session_failed` y la observacion acabo `blocked/goal_backend_gone_without_result`; la causa sigue en investigacion | reproducir la carrera tmux con target exacto que contiene `:` y retener argv/rc/stdout/stderr antes de proponer cambio; evidencia en la [incidencia 208AD](incidencias/incidencia_orquesta_autoprogramacion_tmux_target_colon_race_2026-07-11.md) |
| BUG-ORQ-20260710-208T | retencion runtime sin cuota: `.orquesta-runtime/codex-waves` ocupa 16 GB y conserva 399 homes aislados; no se puede purgar sin clasificar evidencia | retenedor gobernado con inventario, dry-run, export/compresion, cuota y recibo; evidencia en la [incidencia 208T](incidencias/incidencia_orquesta_retencion_runtime_codex_waves_2026-07-10.md) |
| BUG-ORQ-20260701-058/066/075 (familia OPES) | solo residuales de campo: OPES temporal/preproduccion y proveedor real; local/fake cerrado | field test OPES temporal cuando se retome ese frente |

## Cerrados hoy (referencia rapida)

- BUG-ORQ-20260710-208A-D: cierre de codigo local R4. Una confirmacion de
  cleanup atribuible supera snapshot stale; fallos parciales son reintentables;
  shutdown no publica ready con trabajo; ausencia no prueba parada. Evidencia
  en la [incidencia 208C](incidencias/incidencia_orquesta_208c_control_shutdown_primera_orden_2026-07-11.md).
- BUG-ORQ-20260711-215/216/217: cerradas la carrera de contadores del harness HTTP
  y la observacion transitoria posterior a `tmux kill-session`, sin ampliar
  timeouts ni relajar identidad; el guard de latencia conserva 60 s normal y
  no se aplica bajo `-race`, manteniendo todas las aserciones. Evidencia en la
  [incidencia race/shutdown](incidencias/incidencia_orquesta_race_harness_shutdown_2026-07-11.md).
- BUG-ORQ-20260711-208Z: cerrado localmente. La identidad degradada sigue
  bloqueando todo lanzamiento, pero `POST /api/v0/server/shutdown` termina el
  servidor cooperativamente e idempotente, sin señal externa. Evidencia en la
  [incidencia 208Z](incidencias/incidencia_orquesta_degraded_identity_bloquea_shutdown_2026-07-11.md).
- BUG-ORQ-20260711-211/212/213/214: cerrados localmente. El reconciliador
  external-work usa la autoridad causal comun; un terminal exige referencia
  durable; la supersesion de intentos no depende del orden; y el substream no
  amplia review fuera del wait. Evidencia en la
  [incidencia de invariantes causales](incidencias/incidencia_orquesta_invariantes_causales_nucleo_2026-07-11.md).
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

Bugs vivos de codigo local del nucleo: 0 conocidos tras R4. Residuales de
conectores: 1 wiring fail-closed. El resto es operativo, remoto, proveedor o de
campo. Si una lectura historica contradice este indice, prevalece este indice.
