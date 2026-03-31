# Diario del orquestador — 2026-03-30

## 2026-03-31 17:5x aprox. — el `provider_backoff` ya deja telemetría canónica de ventana agotada

Hallazgo:

- la app ya mostraba diario, semanal y cuenta observada si existian artefactos persistidos, pero el caso mas importante seguia cojo: cuando `codex exec resume` devolvia `usage limit / try again at`, Orquesta solo pausaba al agente y reencolaba la orden
- eso dejaba el presupuesto real de proveedor fuera del modelo canonico; `status` seguia dependiendo de diario/semanal aunque el bloqueo efectivo fuese una ventana del proveedor
- ademas, el propio stderr del proveedor ya podia traer identidad util (`perfil activo`, `login`, correo), pero no se promovia a claves canonicas

Decision:

- `provider_backoff` pasa a convertirse en `presupuesto_sesion` persistido, no en texto efimero
- cuando el proveedor indique agotamiento o `try again at`, Orquesta registra una ventana `provider` agotada (`remaining_messages=0`) con `reset_at` observado
- si el stderr trae identidad util, se promueve a `account_user` / `account_email` dentro del snapshot canonico

Cambios:

- `db/controlplane_entities.go`
  - `gestionarBackoffProveedorRuntimeOrderSendInstruction(...)` ahora registra `presupuesto_sesion` tras pausar al agente
  - nuevos helpers:
    - `registrarPresupuestoSesionProviderBackoff(...)`
    - `resolverSesionYModeloRuntimeOrder(...)`
    - `identidadObservadaDesdeErrorProveedor(...)`
- `db/controlplane_entities_test.go`
  - la regresion de `session_resume` con `usage limit` ahora verifica tambien:
    - `budget_source=provider_backoff`
    - `window_kind=provider`
    - `remaining_messages=0`
    - `reset_at` futuro
    - identidad observada `account_user=Codex1`
    - `GetAgente()` enriquecido con `PresupuestoSesionPct=0` y `PresupuestoVentana=provider`

Conclusion:

- la cuota efectiva ya no depende solo del diario/semanal cuando el proveedor ha cerrado la ventana real
- la identidad de cuenta sigue sin inventarse, pero si el proveedor la deja en el stderr o snapshot ahora Orquesta la conserva y la proyecta por API/web/CLI

## 2026-03-31 17:52 aprox. — `agente cuentas` ya enseña el perfil operativo vivo aunque no haya correo

Hallazgo:

- incluso con la superficie de cuentas ya expuesta, los agentes activos seguian apareciendo como `—` si no habia correo observado en snapshot de presupuesto ni en metadata explicita
- en la practica el runtime ya dejaba una identidad operativa util en el propio `rendered_command`: el perfil real de `codex-perfil`

Decision:

- cuando no haya `account_email` ni `account_user` explicitos, Orquesta puede usar como `account_user` el perfil observado en `rendered_command`
- eso da identidad viva util sin inventar correo ni abrir otra fuente de verdad

Cambios:

- `db/sesiones.go`
  - `identidadCuentaDesdeHandle(...)` ahora cae a `perfilCuentaDesdeRenderedCommandHandle(...)`
  - nuevo `CuentaFuente=runtime_handle_profile`
- `db/presupuestos_sesion_test.go`
  - nueva regresion `TestGetAgenteExtraePerfilDesdeRenderedCommandHandle`

Validacion viva:

- con `ORQUESTA_SERVER_URL=http://127.0.0.1:16543`, `./orquesta agente cuentas --activos --json` ya devuelve:
  - `Codex1` `cuenta_usuario=Codex1`
  - `Codex2` `cuenta_usuario=Codex2`
  - `Codex3` `cuenta_usuario=Codex3`
  - `Codex4` `cuenta_usuario=Codex4`
  - `Codex5` `cuenta_usuario=Codex5`
- `./orquesta status` ya muestra `usuario CodexN` en los cinco agentes activos

Conclusion:

- el correo sigue vacio cuando el runtime no lo expone
- la identidad operativa del perfil ya no queda oculta y el orquestador deja de ver “agentes anonimos”

## 2026-03-31 17:55 aprox. — `agente cuentas/presupuesto` vuelven al camino server-first normal

Hallazgo:

- el daemon estaba sano y `server doctor` lo descubria, pero `agente cuentas` seguia cayendo con el mensaje legacy de “arranca serve”
- la causa no era la API ni el modelo de datos, sino una fuga en la tabla canonica `commandSupportsServerMode(...)`: los subcomandos `agente cuentas` y `agente presupuesto` no estaban declarados como server-first

Cambios:

- `cmd/cliente_servidor.go`
  - `commandSupportsServerMode(...)` ahora incluye `agente cuentas` y `agente presupuesto`
- `cmd/cliente_servidor_test.go`
  - regresion explicita para ambos subcomandos

Validacion:

- `go test ./cmd -run 'Test(CommandSupportsServerMode|AgenteCuentasCmdRenderizaListado|AgentePresupuestoCmdRenderizaListado)' -count=1` => OK
- `./orquesta agente cuentas --activos` vuelve a funcionar sin `ORQUESTA_SERVER_URL`

Conclusion:

- la familia `agente` recupera coherencia con el resto de superficies server-first
- la telemetria nueva de cuentas/presupuesto deja de depender de una variable manual para ser usable

## 2026-03-31 17:58 aprox. — `agente cuentas` deja de mostrar formato tosco cuando solo hay usuario

Hallazgo:

- tras exponer el perfil operativo vivo, la CLI seguia pintando `— (CodexN)` cuando no habia correo pero si `cuenta_usuario`
- eso funcionaba, pero seguia pareciendo una ausencia parcial en vez de una identidad operativa valida

Cambios:

- `cmd/agente_telemetria.go`
  - nuevo helper `resumenCuentaObservadaAgente(...)`
  - si hay solo usuario, la salida pasa a `usuario CodexN`
- `cmd/agente_telemetria_test.go`
  - nueva regresion para impedir el formato legado

Validacion viva:

- `./orquesta agente cuentas --activos` ya devuelve:
  - `Codex1 usuario Codex1`
  - `Codex2 usuario Codex2`
  - `Codex3 usuario Codex3`
  - `Codex4 usuario Codex4`
  - `Codex5 usuario Codex5`

## 2026-03-31 12:0x aprox. — la metadata viva del handle deja de arrastrar prompts crudos

Hallazgo:

- incluso despues de cortar `runtime_panic` por `supervisor_local`, la API seguia devolviendo handles enormes
- en el caso vivo de `Codex1` el problema visible venia de `bootstrap_prompt` y `continuity_prompt` embebidos completos en `metadata_json`
- eso infla respuestas, ensucia diagnosticos y aumenta el riesgo de errores o timeouts por cargar en cada vista texto que solo hace falta en el arranque

Decision:

- el handle vivo no es un almacén de prompts completos
- `bootstrap_prompt` y `continuity_prompt` deben salir de la metadata persistida del handle
- si hace falta trazabilidad, se guarda solo resumen compacto o longitud; el prompt completo pertenece al arranque efectivo, al transcript o al manifest del runtime

Cambios:

- `db/controlplane_entities.go`
  - `actualizarHandleRuntimeArranque(...)` deja de persistir los prompts completos
  - nuevos helpers:
    - `compactarMetadataRuntimeHandle(...)`
    - `asignarResumenPromptHandle(...)`
    - `resumirPromptHandle(...)`
    - `compactarMetadataHandleRuntimePersistida(...)`
  - `SincronizarRuntimeHandleSupervisado(...)` compacta tambien handles viejos al sincronizarlos
- tests ajustados:
  - `db/controlplane_entities_test.go`
  - `cmd/controlplane_e2e_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderStartIntegraBootstrapDeHandoffMailboxYCheckpoint|RuntimeOrderStartEmbebeLaunchPromptMultilineaCuandoConectorLoDeclara)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestAPIControlPlaneArranqueRealConBootstrapMultilinea' -count=1` => OK

Conclusion:

- el arranque sigue recibiendo el bootstrap completo donde toca
- el handle persistido deja de cargar esa mochila en cada lectura viva

## 2026-03-31 16:05 aprox. — la recuperacion local ya no relanza por estado fosilizado

Hallazgo:

- la recuperacion autonoma de runtime local seguia decidiendo `local_runtime_failed` a partir de `runtime_instances.logical_state/process_state` y `runtime_handles.estado` persistidos
- eso puede disparar un `start` nuevo aunque el proceso local supervisado siga vivo y solo se haya quedado degradado en la BD

Decision:

- antes de reencolar `start` para un runtime local, el daemon debe observar el proceso real
- si el proceso sigue vivo, la ruta correcta es revivir `handle/runtime`, no relanzar otro runtime

Cambios:

- `cmd/controlplane_support.go`
  - `procesarRecuperacionRuntimeDegradadoSesion()` sincroniza primero el handle local por `db.SincronizarRuntimeHandleSupervisado(..., "autonomia_runtime_recovery")`
- `cmd/controlplane_support_test.go`
  - nueva regresion `TestProcesarAutonomiaAgentesBatchNoRelanzaRuntimeLocalSiSigueVivo`
  - fuerza `handle/runtime` a `fallido` con `PID` vivo y verifica que no aparece ninguna `runtime_order` nueva

Validacion:

- la nueva regresion cubre justo el caso que estaba faltando: estado degradado en persistencia pero proceso local vivo

Conclusion:

- `local_runtime_failed` deja de ser una decision basada solo en estado fosilizado
- la recuperacion local pasa a apoyarse primero en evidencia viva del proceso antes de relanzar

## 2026-03-31 16:20 aprox. — `runtime_panic` deja de arrastrar el comando completo de `script(1)`

Hallazgo:

- la API de `runtime-events` seguia devolviendo algunos `runtime_panic` enormes
- el caso real mezclaba en la misma linea el preambulo de `script(1)` (`Script started on ... [COMMAND=...]`) con el texto de panic
- eso ensucia observabilidad, hincha respuestas y mete bootstrap completo donde solo deberia verse el fallo util

Decision:

- si una linea empieza por `Script started on ` y luego contiene el panic real, Orquesta debe recortarla al marcador de fallo
- el transcript y el `runtime_event` deben conservar el panic, no el comando gigantesco que lo precede

Cambios:

- `db/runtime_transcript.go`
  - nuevo helper `recortarPreambuloSistemaHastaFallo(...)`
  - `limpiarLineaTranscript(...)` recorta el preambulo de `script(1)` antes de persistir
- `db/runtime_transcript_test.go`
  - nueva regresion `TestIngestarRuntimeTranscriptHandleRecortaPreambuloScriptEnRuntimePanic`

Validacion:

- el test nuevo verifica que el transcript y el `runtime_event` ya no contienen `Script started on ...` cuando la misma linea incluye `The application panicked (crashed).`

Conclusion:

- la observabilidad del panic deja de arrastrar bootstrap completo y vuelve a ser legible y barata

## 2026-03-31 16:35 aprox. — `/api/runtime-events` deja de servir megatexto histórico

Hallazgo:

- aunque el ingest nuevo de transcript ya corta mejor los `runtime_panic`, la API seguia devolviendo eventos historicos gigantes
- eso cargaba dashboard/CLI con `message` y `payload_json` enormes, incluso cuando el cliente solo necesita una proyeccion operativa

Decision:

- el raw se conserva en persistencia, pero la API de lectura debe ser compacta por defecto
- `runtime-events` debe truncar `message`, `payload_json` y strings profundas del `payload` a tamaños razonables

Cambios:

- `cmd/api.go`
  - `apiHandlerRuntimeEvents()` pasa por `compactarRuntimeEventsAPI(...)`
  - nuevos helpers:
    - `compactarValorRuntimeEventAPI(...)`
    - `truncarTextoAPI(...)`
- `cmd/api_runtimes_test.go`
  - nueva regresion `TestAPIRuntimeEventsCompactaPayloadGigante`

Conclusion:

- la observabilidad histórica sigue disponible en BD/trazas
- la superficie server-first deja de atascarse por eventos viejos desproporcionados

## 2026-03-31 16:55 aprox. — `status` y la API ya enseñan presupuesto restante por agente

Hallazgo:

- el modelo de `presupuestos_sesion` ya existia, pero `status` y la proyeccion de agentes no mostraban cuanto quedaba por agente
- eso dejaba al orquestador ciego justo en el punto que decide pausas, handoff y reparto de trabajo cuando los runtimes se quedan sin cuota

Decision:

- la vista server-first de agentes debe exponer presupuesto restante de forma operativa
- prioridad de datos:
  - snapshot real de `presupuestos_sesion`
  - si falta, fallback al porcentaje derivable de consumo diario

Cambios:

- `db/sesiones.go`
  - `Agente` incorpora presupuesto visible: porcentaje restante, estado, fuente y `remaining_*`
  - `ListarAgentes*` y `GetAgente` enriquecen cada agente con el ultimo presupuesto evaluado
- `cmd/status_remoto_compat.go`
  - `status` muestra `restante %`, créditos y estado relevante junto al agente activo
- tests:
  - `db/presupuestos_sesion_test.go`
  - `cmd/status_test.go`

Conclusion:

- ya se puede ver desde la app cuanto margen real le queda a cada agente
- el siguiente paso es hacer que los runtimes reporten ese presupuesto con más regularidad, no solo al chocar contra `usage limit`

## 2026-03-31 18:55 aprox. — el porcentaje visible ya distingue ventana efectiva y reset

Hallazgo:

- mostrar solo un `%` suelto seguia siendo ambiguo
- un agente podia salir bien en diario y estar casi agotado en semanal, o tener una ventana real de `5h` mas restrictiva que ambas

Decision:

- el `%` visible debe ser el presupuesto efectivo mas restrictivo
- junto al porcentaje, Orquesta debe mostrar que `ventana` ha ganado (`5h`, `daily`, `weekly`, etc.) y su `reset_at` cuando exista

Cambios:

- `db/sesiones.go`
  - el enriquecimiento de `Agente` ya elige el minimo entre presupuesto de sesion, diario y semanal
  - se exponen `PresupuestoVentana` y `PresupuestoResetAt`
- `cmd/status_remoto_compat.go`
  - `status` muestra `restante % · ventana ... · reset ...`
- tests:
  - `db/presupuestos_sesion_test.go`
  - `cmd/status_test.go`

Conclusion:

- el orquestador ya no ve una cifra aislada, sino el limite efectivo que realmente manda

## 2026-03-31 11:3x aprox. — fusible permanente para `supervisor_local` tras `runtime_panic` real en Codex

Hallazgo:

- el caso vivo de `Codex1` ya no era el bootstrap largo ni el banner de `script(1)`; esos frentes estaban cerrados
- aun con bootstrap compactado, `Codex1` seguia paniqueando en `tui_app_server/src/wrapping.rs:52` justo despues de una entrega por `stdin`
- el problema ya no era de texto concreto sino del canal `supervisor_local` sobre ciertos handles Codex

Decision:

- no seguir reintentando `stdin` caliente sobre un handle que ya ha demostrado inestabilidad
- degradar ese handle de forma persistente para el resto de su ciclo de vida
- dejar que las entregas futuras caigan a `session_resume` o permanezcan en `runtime_mailbox`, que son caminos mas seguros

Cambios:

- `db/controlplane_entities.go`
  - `RuntimeHandlePermiteEntregaCalienteSupervisada(...)` ya devuelve `false` si el handle tiene `disable_supervisor_hot_input=true`
  - nuevo helper `DeshabilitarEntregaCalienteSupervisadaHandle(...)`
- `cmd/controlplane_support.go`
  - `enfriarAgentePorRuntimePanic(...)` ahora degrada primero el handle supervisado y luego aplica el cooldown del agente
- tests nuevos o reforzados:
  - `db/controlplane_entities_test.go`
    - `TestRuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin`
    - `TestRuntimeOrderSendInstructionCodexSupervisadoDegradadoCaeASessionResume`
  - `cmd/controlplane_support_test.go`
    - `TestProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic` ahora verifica tambien la degradacion del handle

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|RuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal|RuntimeOrderSendInstructionCodexSupervisadoDegradadoCaeASessionResume)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic' -count=1` => OK

Conclusion:

- Orquesta deja de tratar todos los handles Codex supervisados como iguales
- un handle que revienta por `stdin` ya no vuelve a recibir guidance caliente como si nada
- el siguiente paso vivo es observar el siguiente ciclo de `Codex1` con este fusible activo

## 2026-03-31 08:1x aprox. — `status` deja de contar agentes pausados o en enfriamiento como activos

Hallazgo:

- tras corregir `provider_backoff`, `Codex2` quedaba bien pausado por cuota:
  - handle `369` en `pausado`
  - `agente tick` devolviendo `pausar_por_cuota`
- aun asi, `status` seguia mostrandolo dentro de “Agentes: 5 activos ahora”
- la causa estaba en `aplicarEstadoVisibleAgente(...)`: marcaba `Activo=true` con solo tener sesion operativa

Decision:

- un agente en `estado_cuota=enfriamiento` no puede contarse como activo visible
- una sesion `pausada` tampoco debe sumar presencia activa, aunque conserve heartbeat o handle reciente

Cambios:

- `db/sesiones.go`
  - `aplicarEstadoVisibleAgente(...)` deja `Activo=false` para:
    - `estado_cuota=enfriamiento`
    - `sesion.Estado=pausada`
- `db/sesiones_test.go`
  - `TestListarAgentesOcultaAgenteEnEnfriamientoAunqueTengaSesionOperativa`
  - `TestListarAgentesOcultaAgentePausadoAunqueMantengaHeartbeat`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ListarAgentesOcultaAgenteEnEnfriamientoAunqueTengaSesionOperativa|ListarAgentesOcultaAgentePausadoAunqueMantengaHeartbeat|ListarAgentesOcultaSesionZombiPeroMantieneHandleActivo|ListarAgentesIgnoraSesionesConHeartbeatObsoleto)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva tras reiniciar daemon:
  - `curl /api/runtime-handles?agente=Codex2` => handle `369`, estado `pausado`
  - `./orquesta agente tick Codex2 --proyecto orquestador` => `pausar_por_cuota`
  - `./orquesta status` pasa de `5 activos` a `4 activos` y deja fuera a `Codex2`

## 2026-03-31 08:0x aprox. — `session_resume` detecta cuota aunque Codex anteponga banner

Hallazgo:

- el caso vivo de `Codex2` seguia quedando como `session_resume fallo: exit status 1`
- la reproduccion exacta del comando `codex-perfil Codex2 exec resume ...` devolvio el error real:
  - `ERROR: You've hit your usage limit ...`
- el problema no era el control plane, sino el resumen de stderr:
  - Codex imprime antes `Perfil activo`, `CODEX_HOME`, `Consejo`
  - `trimmedCommandOutput(...)` truncaba desde el principio y el detector de `provider_backoff` no llegaba a ver `usage limit`

Decision:

- el resumen de salida debe priorizar el error relevante frente al banner
- si el stderr contiene `usage limit`, `rate limit`, `purchase more credits`, `try again at` o `ERROR:`, la salida resumida debe recortarse desde ese punto

Cambios:

- `internal/controlruntime/codex_resume.go`
  - `trimmedCommandOutput(...)` prioriza marcadores de error relevantes antes de truncar
- `internal/controlruntime/codex_resume_test.go`
  - nueva regresion `TestTrimmedCommandOutputPrefiereErrorRelevanteAlBanner`
- `db/controlplane_entities_test.go`
  - `TestRuntimeOrderSendInstructionSessionResumePausaPorCuotaProveedor` pasa a cubrir stderr con banner previo de Codex

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(CodexResumeTimeoutDefaultYOverride|TrimmedCommandOutputPrefiereErrorRelevanteAlBanner|EnviarInstruccionSesionResumeUsaCodexPerfil|EnviarInstruccionSesionResumeAceptaMetadataDeSupervisorLocal)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestRuntimeOrderSendInstructionSessionResumePausaPorCuotaProveedor' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva tras reiniciar daemon:
  - `#80324` y `#80327` dejan de quedar como `session_resume fallo: exit status 1`
  - pasan a `agente en enfriamiento por usage limit de proveedor`
  - `./orquesta agente tick Codex2 --proyecto orquestador` devuelve `pausar_por_cuota`

## 2026-03-31 07:5x aprox. — la lease de `runtime_orders` deja de expirar dos veces

Hallazgo:

- las `send_instruction` que parecian "colgadas" no estaban necesariamente perdidas; el problema era que la reconciliacion stale trataba `lease_expires_at` como si aun necesitara pasar otro `stale_seconds`
- la query hacia `COALESCE(lease_expires_at, started_at, updated_at, created_at) <= cutoff`
- como `lease_expires_at` ya es una fecha futura de caducidad, compararla contra `cutoff=now-stale_seconds` retrasa la recuperacion aproximadamente otro ciclo completo

Decision:

- una lease vencida se reconcilia contra `now`
- el `cutoff` solo aplica a ordenes sin lease explicita

Cambios:

- `db/controlplane_entities.go`
  - `ReconciliarRuntimeOrdersStale()` separa:
    - `lease_expires_at <= now`
    - o, si no hay lease, `started_at/updated_at/created_at <= cutoff`
  - `reconciliarRuntimeOrderStale(...)` aplica la misma regla al `UPDATE`
- `db/controlplane_entities_test.go`
  - `TestReconciliarRuntimeOrdersStaleRecuperaLeaseExpiradaSinEsperarOtroCutoff`
  - ajuste de `TestReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ReconciliarRuntimeOrdersStaleRecuperaBasicasYExpiraNoSoportadas|ReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado|ReconciliarRuntimeOrdersStaleRecuperaLeaseExpiradaSinEsperarOtroCutoff|ReconciliarRuntimeOrderStaleNoPisaOrdenYaCompletada|ReconciliarRuntimeOrderStaleNoPisaOrdenReclamadaDeNuevo)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva tras reiniciar daemon:
  - `Codex1 #80312` pasa por fin a `completada`
  - `Codex2 #80323` pasa a `completada` con `mailbox_only=true`
  - `Codex2 #80324` deja de quedarse secuestrada y vuelve a `pendiente` al fallar `session_resume`

## 2026-03-31 07:4x aprox. — timeout de `session_resume` para autonomia deja de reencolar la misma orden

Hallazgo:

- el siguiente cuello real tras bajar el timeout seguia concentrado en `autonomia` larga sobre `session_resume`
- una `send_instruction` nacida de mailbox durable podia fallar por `session_resume timeout` y volver a `pendiente`
- eso no mejoraba la entrega: solo hacia girar la misma orden mientras el mailbox ya era la fuente durable correcta

Decision:

- para mailbox autonomo (`autonomia`, `nudge`, `watchdog`, `governance_refresh`, `skills_refresh`), un `session_resume timeout` no se reintenta sobre la misma orden
- la orden derivada se completa como `mailbox_only` con trazabilidad del timeout
- el mensaje durable permanece pendiente en `runtime_mailbox`, que sigue siendo la unica verdad de continuidad

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper `runtimeOrderSendInstructionDebeDegradarseAMailboxPorError(...)`
  - `ejecutarRuntimeOrderSendInstruction(...)` deja de reencolar el mismo intento cuando el fallo es `session_resume timeout` en kinds autonomos
- `db/controlplane_entities_test.go`
  - nueva regresion `TestRuntimeOrderSendInstructionAutonomiaTimeoutDegradaAMailboxDurable`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderSendInstructionAutonomiaTimeoutDegradaAMailboxDurable|RuntimeOrderSendInstructionReencolaMailboxTrasFalloSessionResume)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - el contrato nuevo ya queda cubierto por test dirigido
  - al intentar reproducirlo en vivo sobre `Codex1` y `Codex2` aparecio otro problema distinto y previo: varias `send_instruction` quedan en `ejecutando` sin cerrar (`#80312`, `#80323`, `#80324`)
  - ese leak de ciclo de vida del runner queda separado como siguiente frente; no se mezcla en este commit con el cambio de contrato del timeout

## 2026-03-31 03:xx aprox. — el bootstrap pasa a poseer de verdad su mailbox y deja de duplicarse por `session_resume`

Hallazgo:

- el caso vivo de `Codex2` ya no era un problema generico de `send_instruction`
- el `start #80154` habia arrancado con `bootstrap.mailbox_count=1`, es decir, el mensaje `64424` ya estaba dentro de la continuidad pendiente
- aun asi, el batch `procesarRuntimeMailboxSessionResumeBatch()` materializo una segunda `send_instruction` (`#80157`) para ese mismo `mailbox_id`
- eso demostraba una fuga de contrato: la `bootstrap lease` y el batch de mailbox competian por la misma verdad

Decision:

- si un `mailbox_id` ya esta cubierto por una `bootstrap lease` pendiente, ese mensaje pertenece al bootstrap hasta su `ack`
- ningun batch puede crear otra `send_instruction` para ese mismo `mailbox_id`
- si aun existe una `send_instruction` vieja para ese mailbox, debe cerrarse como `superseded` por `covered_by_bootstrap_lease`
- ademas, una `send_instruction` creada desde mailbox deja de ser cola durable propia: si no hay runtime entregable inmediato, la orden se completa y la verdad queda solo en el mailbox

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper exportado `RuntimeMailboxCubiertoPorBootstrapPendiente(...)`
  - nuevo cierre `completarRuntimeOrderSendInstructionCubiertaPorBootstrap(...)`
  - nuevo cierre `completarRuntimeOrderSendInstructionDiferidaAMailbox(...)`
  - `ejecutarRuntimeOrderSendInstruction(...)` deja de reencolar cuando la verdad ya esta cubierta por bootstrap o cuando el mailbox ya es la fuente durable suficiente
- `cmd/controlplane_support.go`
  - `procesarRuntimeMailboxSessionResumeBatch()` salta el mensaje si ya esta cubierto por la lease de bootstrap
- tests nuevos:
- `db/controlplane_entities_test.go`
  - `TestRuntimeOrderSendInstructionMailboxCubiertaPorBootstrapSeCompletaSinReintento`
  - `TestRuntimeOrderSendInstructionMailboxSeCompletaDejandoLaVerdadEnMailboxSiNoHayHandleEntregable`
  - `cmd/controlplane_support_test.go`
    - `TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderSendInstructionMailboxCubiertaPorBootstrapSeCompletaSinReintento|RuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|GuardarSesionActivaPreservaMetadataRicaDelHandle|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(ProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstruction|ProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionParaWatchdog|ProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente|ProcesarRuntimeMailboxSessionResumeBatchCoalesceNudgeAunqueHayaOrdenAbierta)' -count=1` => OK

## 2026-03-31 00:xx aprox. — decision de no reescritura sobre framework externo y cierre de doctrina del nucleo

Objetivo:

- cortar el riesgo de seguir reprogramando sintomas sin una direccion fija
- fijar por escrito si Orquesta se reescribe o no sobre LangChain/LangGraph/CrewAI/AutoGen
- dejar una doctrina estable del nucleo de orquestacion para las siguientes pasadas

Conclusion tomada:

- no se reescribe Orquesta sobre LangChain, LangGraph, CrewAI ni otro framework externo
- si se copiaran patrones, se copiaran como arquitectura y semantica, no como sustitucion del producto

Motivo:

- el problema real no es de prompts ni de grafo conversacional
- el problema real es de control operativo duradero:
  - daemon unico
  - cola durable
  - lease y ack
  - handoff con continuidad
  - supervisor de runtime
  - reconciliacion tras reinicio o caida
- cambiar de framework ahora destruiria continuidad y abriria otra transicion a medias

Referencias estudiadas:

- AutoGen Core como patron de supervisor/manager y mensajes entre agentes
- OpenAI Agents para handoff filtrado y resumen de continuidad
- Kubernetes controller/operator como referencia de reconciliacion declarativa
- Celery/Temporal como referencia de cola durable con retry/backoff/ack

Decision doctrinal fijada en la biblia:

- `orquesta server` sigue siendo el unico plano de control
- `runtime_orders` debe evolucionar a cola formal con `pending/leased/running/completed/failed/...`
- `runtime_mailbox` debe tipificarse con claridad y no mezclar señales internas con mensajes entregables al agente
- si un `kind` de mailbox puede quedar pendiente, debe tener una ruta oficial de entrega o consumo

Hallazgo operativo ya visible antes de tocar codigo:

- `Codex1` ya no tenia la orden `#79976` colgada en `ejecutando`; el daemon la reencolo en `pendiente`
- aun asi seguian quedando `8` mensajes pendientes en mailbox para `Codex1` sin runtime vivo ni tarea activa
- esto confirma que el siguiente frente correcto ya no es el arranque del daemon sino la semantica y reconciliacion de mailbox

## 2026-03-31 01:xx aprox. — reconciliacion de watchdog pendiente sin ruta real de entrega

Objetivo:

- corregir una incoherencia concreta del control plane sin abrir otro refactor ciego
- evitar que `watchdog` quede pendiente indefinidamente en `runtime_mailbox` cuando ya no existe handle activo para el agente

Hallazgo:

- `watchdog` si tiene sentido como señal persistente cuando existe un runtime activo al que sondar
- pero tambien puede quedar pendiente mucho despues de que el runtime/handle haya desaparecido
- en ese estado deja de ser una señal entregable y pasa a ser ruido operativo

Cambio aplicado:

- `procesarRuntimeMailboxBatch()` ahora empieza por reconciliar `watchdog` pendientes sin handle activo
- esos mensajes se marcan como `entregado` + `consumido`
- se deja auditoria explicita `runtime_mailbox_watchdog_sin_handle`

Criterio doctrinal que queda reforzado:

- un mensaje pendiente solo puede seguir vivo si existe una ruta oficial de entrega o consumo
- si `watchdog` ya no tiene runtime/handle activo al que aplicarse, debe consumirse como señal obsoleta
- esto no elimina `watchdog` como primitive; solo evita su acumulacion como mentira operativa

## 2026-03-31 02:xx aprox. — unificacion oficial de `orquestador` -> `orquesta`

Objetivo:

- cortar otra fuente de confusion recurrente entre nombre de app, slug de proyecto y ruta fisica del repo
- evitar que agentes futuros sigan mezclando `orquesta` y `orquestador` como si fueran dos proyectos vivos distintos

Hallazgo:

- el estado vivo seguia usando `orquestador` como slug del proyecto activo aunque la ruta real ya era `/home/alberto/Trabajo/orquesta`
- no existian dos proyectos raiz activos distintos para la misma ruta; era el mismo registro persistido con slug legacy

Accion ejecutada por la via oficial:

- `./orquesta proyecto descubrir /home/alberto/Trabajo/orquesta`

Resultado:

- el proyecto activo `id=1` queda normalizado como `slug=orquesta`
- las asignaciones activas de `Codex1-6` pasan a referenciar `orquesta`
- las tareas activas/asignadas siguen accesibles bajo `--proyecto orquesta`
- `orquestador` deja de existir como proyecto vivo en la lista

Lectura doctrinal:

- para esta base actual el proyecto canonico vivo es `orquesta`
- `orquestador` queda ya como nombre legacy en documentos antiguos o en historico, pero no como slug operativo
- cuando la via oficial puede normalizar sin fusion destructiva, se prefiere eso antes que borrar o manipular entidades a mano

## 23:55 aprox. — verificación oficial de persistencia y limpieza de mantenimiento

Objetivo:

- cerrar otro resto importante de deriva SQLite-céntrica en el camino oficial de operaciones
- dejar la verificación de persistencia dentro de Orquesta, backend-aware, en vez de empujar al operador hacia scripts o hábitos manuales ambiguos

Hallazgos:

- ya existía `Backup` por backend en `db/backend.go`, pero faltaba la verificación oficial equivalente
- el camino de mantenimiento seguía quedando partido:
  - `persistencia info` explicaba el backend y el daemon
  - `scripts/verificar_bd.sh` seguía siendo la única verificación explícita, y además solo para SQLite
- `docs/manual_tecnico_sistemas.md` todavía arrastraba lenguaje de “SQLite como arquitectura” y una recomendación operativa mala (`pkill -9 orquesta`)
- además `cmd/respaldo.go` conservaba un helper `literalSQLite` ya muerto, que solo añadía ruido conceptual

Cambios aplicados:

- se añade el contrato `Verify` al backend de persistencia
- se incorpora `db.VerificarPersistenciaActual()` con informe estructurado:
  - `driver`
  - `target`
  - `sano`
  - lista de comprobaciones
- SQLite verifica:
  - conexión
  - `PRAGMA quick_check`
  - presencia de tablas core
- MySQL/Postgres verifican:
  - conexión
  - presencia de tablas core
- se expone la ruta oficial `GET /api/persistencia/verificar`
- se añade `orquesta persistencia verificar`
- la biblia deja fijado que `persistencia info` y `persistencia verificar` son la vía oficial de observabilidad/verificación, y que los scripts backend-específicos quedan para rescate
- `manual_tecnico_sistemas.md` deja de vender SQLite como si fuera el diseño entero y sustituye la receta de `pkill -9` por `server doctor` + `persistencia info` + `persistencia verificar`
- se elimina el helper muerto `literalSQLite` de `cmd/respaldo.go`

Lectura doctrinal consolidada:

- el camino oficial ya no es “abre un script y mira SQLite”
- el camino oficial pasa por Orquesta y sus adaptadores
- los scripts que queden deben decir con claridad si son rescate y para qué backend sirven

## 00:10 aprox. — el backup automatizado deja de fingir que es genérico

Objetivo:

- eliminar otra fuente de despiste para agentes futuros: una unit systemd de backup que por nombre parecía general, pero en realidad solo valía para SQLite

Cambios aplicados:

- `scripts/orquesta-backup.service` se sustituye por `scripts/orquesta-backup-sqlite.service`
- `scripts/orquesta-backup.timer` se sustituye por `scripts/orquesta-backup-sqlite.timer`
- `scripts/instalar_servicio.sh` pasa a instalar y anunciar esos nombres explícitos
- `scripts/backup_bd.sh` y `scripts/verificar_bd.sh` dejan más claro que son utilidades de rescate/automatización solo-SQLite
- `manual_tecnico_sistemas.md` añade `orquesta respaldo bd` como snapshot manual por el camino oficial

Lectura doctrinal:

- no es aceptable dejar nombres neutros a piezas que no lo son
- el camino oficial multi-backend pasa por Orquesta
- las piezas SQLite-only deben identificarse como SQLite-only en nombre y documentación

## 00:20 aprox. — limpieza de doctrina residual SQLite-first

Hallazgos:

- `docs/op_088_orquesta_mcp_server.md` seguía mostrando la integración MCP contra `SQLite: orquesta.db`
- `docs/politica_backups_es.md` y `docs/politica_backups_en.md` parecían políticas generales, pero en realidad describían solo restauración/copia de ficheros SQLite
- el índice aún anunciaba esa política como si fuera “de la BD” en general

Cambios aplicados:

- el diagrama de OP-088 pasa a hablar de `Storage Adapter: backend activo`
- las políticas de backup se reetiquetan como políticas del backend SQLite
- ambas políticas remiten ya a `orquesta respaldo bd` y `orquesta persistencia verificar` como camino oficial
- el índice refleja explícitamente que esa política es para SQLite

Lectura doctrinal:

- no basta con tener la biblia correcta si otras piezas siguen sugiriendo una arquitectura vieja
- toda referencia residual que empuje a leer “Orquesta = SQLite file” debe corregirse o desaparecer

## 00:30 aprox. — corrección de OPs que todavía filtraban arquitectura vieja

Hallazgos:

- `docs/op_088_orquesta_servidor_mcp.md` seguía describiendo prompts MCP como si leyesen `reglas/skills/workflows` “de SQLite”
- `docs/op_089_relevo_agentes_por_token.md` seguía formulando el checkpoint en términos de “escribir en SQLite”
- `docs/op_049_matriz_voto.md` mantenía el riesgo como “que SQLite no quede expuesto a demasiada concurrencia”, cuando la regla correcta ya es proteger el single-writer y el backend activo

Cambios aplicados:

- OP-088 larga: briefing y auditoría reformulados contra servicios/adaptador de persistencia, no contra SQLite
- OP-089: checkpoint/handoff reformulados contra `runtime_checkpoints` vía adaptador de persistencia activo
- OP-049: riesgo reformulado en términos de concurrencia lateral y ruptura del single-writer

Lectura doctrinal:

- las OP también son arquitectura viva
- si una OP vieja describe bien la intención pero mal el mecanismo, hay que corregirla para que no vuelva a arrastrar implementación equivocada

## 16:00 aprox. — foco actual

Objetivo en curso:

- verificar desde la propia app de Orquesta, en modo `server-first`, qué control real existe hoy sobre `Codex1-6`
- separar lo que está codificado de verdad de lo que sigue siendo wrapper, compatibilidad o rescate
- definir el flujo operativo correcto sin recurrir a `sqlite`

Hallazgos ya confirmados:

- `Orquesta server` es la vía operativa válida; `CLI`, `web` y `API` son clientes del servicio
- `agente preparar` y `agente tick` ya van por API y sirven para bundle/heartbeat operativo
- el control activo de ciclo de vida existe como `orquesta agente control <arrancar|pausar|continuar|detener> <agente>`
- la observabilidad real del control plane vive en `runtime listar`, `runtime handles`, `runtime ordenes`, `runtime mailbox`, `runtime checkpoints` y `runtime diagnostico`
- los scripts `inicio_agente.sh` y `agente_console.sh` siguen presentes, pero el propio repo los marca como compatibilidad/rescate, no como camino oficial

Trabajo en curso:

- contrastar esos comandos con la instancia viva para identificar señales sanas y señales de degradación por agente
- convertir esta lectura en un runbook operativo concreto para validar control real sobre `Codex1-6` sin tocar BD

Riesgo principal que se está atacando:

- evitar seguir “programando en bucle” sobre síntomas y centrar la validación en contratos observables del control plane

## 18:00 aprox. — evidencia viva contra Orquesta

Trazabilidad enviada a agentes:

- nudges encolados a `Codex1-6` vía `orquesta runtime nudge`
- órdenes creadas: `#79844` a `#79849`

Comandos verificados con salida útil:

- `orquesta server status`
- `orquesta status`
- `orquesta sesion listar`
- `orquesta runtime listar --proyecto orquestador`
- `orquesta runtime handles --agente Codex1`
- `orquesta runtime ordenes --agente Codex1 --proyecto orquestador`
- `orquesta runtime mailbox --to Codex1 --proyecto orquestador`
- `orquesta runtime diagnostico --agente Codex1 --proyecto orquestador --limit 5`
- `orquesta agente tick Codex1..Codex6 --proyecto orquestador`

Señales confirmadas:

- `status` ve `Codex1-6` como agentes activos en la plataforma
- `sesion listar` marca activas las sesiones de `Codex1-5`; `Codex6` queda sin sesión activa
- `tick` devuelve acción operativa útil:
  - `Codex1`, `Codex3`, `Codex4`, `Codex5`: `continuar_trabajo`
  - `Codex2`: `votar_propuestas_pendientes`
  - `Codex6`: `esperar_o_pedir_tarea`
- el nudge nuevo a `Codex1` aparece en `runtime ordenes` como `completada`
- en `runtime mailbox` de `Codex1` siguen quedando mensajes pendientes (`nudge`, `governance_refresh`, `watchdog`, `autonomia`)
- `runtime diagnostico` muestra handle activo actual para `Codex1` y también un volumen excesivo de runtimes/handles históricos cerrados

Desajuste importante observado:

- `status`, `sesion` y `runtime*` responden por la ruta server-first
- `tarea listar` y `propuesta listar` no han quedado consistentes todavía en esta instancia: unas veces exigen servidor aunque hay daemon, y con `ORQUESTA_SERVER_URL=http://127.0.0.1:16543` han devuelto `connection refused`

Conclusión provisional:

- sí existe control plane real observable desde Orquesta sin `sqlite`
- todavía hay incoherencias en la superficie cliente para algunas rutas (`tarea` y `propuesta`) y sigue habiendo backlog pendiente en mailbox/runtimes históricos

## 20:15-20:35 aprox. — arranque real de flota por vía de rescate

Objetivo:

- comprobar personalmente si Orquesta puede arrancar de verdad `Codex1-6` y ponerlos a trabajar sin bajar a `sqlite`

Acciones ejecutadas:

- se levantó el daemon local con `./orquesta server run --addr 127.0.0.1:16543`
- se confirmó `server status` sano en host: daemon activo y `healthz` visible
- se releyó el estado real de proyecto y tareas desde Orquesta
- se preparó un plan manual de lanzamiento de flota
- se ejecutó `scripts/preparar_lanzamiento_agentes.sh --ejecutar /tmp/orquesta-flota-20260330.plan`
- se crearon worktrees dedicados:
  - `orquesta-codex1`
  - `orquesta-codex2`
  - `orquesta-codex3`
  - `orquesta-codex4`
  - `orquesta-codex5`
  - `orquesta-codex6`
- el plan generó además nuevas tareas asignadas en Orquesta:
  - `#421` Codex1
  - `#422` Codex2
  - `#423` Codex3
  - `#424` Codex4
  - `#425` Codex5
  - `#426` Codex6
- se lanzaron seis consolas reales vía `scripts/agente_console.sh` y quedaron procesos `node .../bin/codex` vivos en host, uno por agente
- se inyectaron instrucciones manuales en cada PTY para orientar el trabajo sobre sus frentes reales

Evidencia confirmada:

- el proyecto lógico activo es `orquesta`; `orquestador` ya no existe como proyecto vivo en esta base
- `./orquesta proyecto listar` muestra:
  - `orquesta`
  - `orquesta-archivado-20260329-130458`
- `./orquesta tarea listar --proyecto orquesta --estado asignada --tsv` devuelve las tareas `410`, `411`, `414`, `415`, `417`, `418`, `419`, `420` y las nuevas `421-426`
- los wrappers `agente_console.sh Codex1..6` siguen vivos, con sus procesos `node .../bin/codex` hijos activos

Bloqueo estructural detectado:

- `scripts/agente_console.sh` llama a `orquesta sesion inicio` y `orquesta agente preparar`
- `sesion inicio` sí puede abrir sesión y devuelve el briefing largo del agente
- `agente preparar` falla por API con timeout aun teniendo el daemon activo:
  - `Get "http://127.0.0.1:16543/api/agente/preparar?...": net/http: timeout awaiting response headers`
- como `agente preparar` falla, el launcher cae al `BOOTSTRAP_PROMPT_DEFAULT`
- ese prompt por defecto deja al runtime en modo:
  - "si no hay mas instrucciones del usuario, mantente en espera"

Conclusión operativa:

- sí he arrancado personalmente la flota y he dejado seis agentes vivos en host
- no está cerrado todavía el arranque autónomo correcto porque el launcher de rescate depende de `agente preparar`, y hoy ese endpoint no responde a tiempo
- por eso la flota puede quedar abierta pero mal bootstrappeada si no se le inyecta trabajo manual después

## 21:25-21:35 aprox. — cierre del cuello real en `prepare` y `status`

Objetivo:

- dejar de trabajar a ciegas sobre síntomas y cerrar el cuello del núcleo medido en el daemon real

Diagnóstico confirmado:

- `agente preparar` no estaba roto de forma genérica; el problema era latencia excesiva
- con instrumentación en servidor se midió primero `BuildPrepare` en ~31-34s
- el tramo dominante era `BuildLaunchBootstrapPrompt -> EsSupervisorAutonomiaOperativo`, con ~23s
- además `/api/status` hacía doble trabajo:
  - `statusService.FetchStatus()`
  - `buildEstadoResumen()`
- la CLI `status` usaba además un cliente HTTP distinto con timeout de `3s`, inferior al coste real del endpoint

Cambios aplicados:

- fast-path en selección de supervisor operativo para reutilizar `SupervisorAgente` configurado cuando sigue siendo válido
- `governance_overrides` fuera del camino de lectura lenta, pasando a bootstrap/migración
- `agente preparar` ya no consume bootstrap al inspeccionar
- `/api/status` deja de recomputar el resumen completo por segunda vez
- los votos de propuestas abiertas se cargan ahora en batch con una sola consulta (`ResumenVotosPorPropuestas`)
- `status` reutiliza el cliente HTTP normal de servidor en vez de un cliente separado de `3s`

Verificación real en host, contra daemon actualizado:

- `ORQUESTA_SERVER_URL=http://127.0.0.1:16543 ./orquesta agente preparar Codex1 --proyecto orquestador --conector codex-cli --campo mode`
  - responde `resume`
- `curl /api/agente/preparar?...`
  - `HTTP 200`
  - `time_total=8.683676`
- `./orquesta status`
  - vuelve a responder completo
- `curl /api/status`
  - `HTTP 200`
  - `time_total=3.086607`

Lectura operativa:

- se ha salido del bucle real: el launcher ya no cae por timeout al pedir `prepare`
- `status` vuelve a ser usable como señal de salud del sistema
- el núcleo no está “acabado al 100%”, pero estos dos bloqueos centrales ya no están abiertos
- el siguiente trabajo ya no es adivinar si la app responde, sino validar y endurecer el gobierno vivo de runtimes/agents sobre esta base estable

Validación end-to-end del launcher de rescate:

- se ejecutó `scripts/agente_console.sh Codex1 orquestador /tmp/orquesta-validate-launch codex-cli /tmp/codex-perfil`
- `/tmp/codex-perfil` era un lanzador falso que solo capturaba los argumentos recibidos
- el wrapper completó `sesion inicio`, pidió `agente preparar`, lanzó el runtime y cerró sesión guardando continuidad
- el fichero capturado `/tmp/orquesta-fake-codex-launch.txt` ya no contiene el prompt genérico de “mantente en espera”
- contiene un bootstrap real de Orquesta con:
  - rol del agente
  - proyecto
  - directorio de trabajo
  - perfil/modelo/razonamiento
  - continuidad previa
  - rol operativo de orquestación autónoma
  - tareas activas y reglas efectivas

Conclusión adicional:

- el camino `scripts/agente_console.sh -> orquesta sesion inicio -> orquesta agente preparar -> runtime_launch`
  vuelve a estar operativo como flujo de compatibilidad/rescate

## 22:30-23:30 aprox. — supervisor local residente y cierre de deriva circular

Objetivo:

- alinear el gobierno de runtimes locales con `ARQUITECTURA.md`
- sacar la propiedad del proceso local de `db`/polling oportunista y moverla a `internal/controlruntime`
- dejar de depender de inspecciones laterales para entender por qué una `runtime_order` queda pendiente

Lectura arquitectónica aplicada:

- `ARQUITECTURA.md` insiste en que Orquesta debe depender de un contrato estable de runtime y centralizar el acceso concurrente
- eso invalida seguir tratando el runtime local como un PID sin dueño que luego se "redescubre" desde `db`
- la pieza correcta era un supervisor local residente en el adaptador `controlruntime`, no otro parche en SQL o en el runner

Cambios aplicados:

- se añadió un supervisor local residente en `internal/controlruntime/supervisor_local.go`
- `ArrancarPlan` local registra el proceso arrancado bajo supervisión residente y persiste `supervisor_ref`/`supervision_mode`
- `ProcesoVivo`, `PausarProceso`, `ContinuarProceso`, `DetenerProceso` y `EnviarInstruccionProceso` consultan primero ese supervisor
- `db/controlplane_entities.go` ya no observa el runtime local solo como PID; aplica estado local observado desde `controlruntime`
- el supervisor detecta y propaga `external_session_id` de runtimes Codex cuando puede, para reforzar la vía durable `session_resume`

Riesgo de programación circular detectado y corregido:

- en una primera iteración el supervisor estaba sobrescribiendo `driver=process_pty_cli` con `driver=local_runtime_supervisor`
- eso rompía conceptualmente la ruta `DetectExternalSessionID -> session_resume`, o sea, el cambio "arreglaba" una capa y degradaba otra
- también apareció otra deriva real: al reutilizar una entrada del supervisor por PID, podía quedarse con metadata vieja
- ambos puntos quedaron corregidos:
  - el supervisor ya publica `supervisor_driver` sin pisar el `driver` real del runtime
  - si reengancha por PID, refresca su metadata con el contexto nuevo antes de devolver estado

Validación hecha:

- `go test ./internal/controlruntime ./db ./cmd -run ...` pasa en la batería dirigida del supervisor, control plane y recuperación local
- se añadió test específico para garantizar que `ConsultarEstadoLocal` detecta `external_session_id` sin sobrescribir el `driver`
- el binario `./orquesta` recompila bien

Mejora operativa añadida dentro de Orquesta:

- `orquesta runtime ordenes` ahora muestra `DISPONIBLE` y `DETALLE`
- esto permite ver desde la propia app si una orden está diferida por reintento o simplemente pendiente, sin bajar a `sqlite`

Lectura real contra la base actual:

- `ORQUESTA_FORCE_LOCAL=1 ./orquesta runtime ordenes --agente Codex1 --estado pendiente`
  muestra una sola orden viva:
  - `#79959` `send_instruction`
- esa orden no aparece como diferida ni con `retry_after`; sale pendiente con `available_at = created_at`
- por tanto, el siguiente foco real ya no es el algoritmo de reencolado, sino el consumidor del control plane o el acceso al daemon en este entorno

Limitación del entorno actual:

- desde esta sandbox no se puede abrir socket ni consultar `127.0.0.1:16543` (`socket: operation not permitted`)
- `server doctor` y `server status` no sirven aquí para concluir si el daemon real del host está procesando o no la cola
- el log local visible desde la sandbox mezcla arranques fallidos del propio entorno con `SQLITE_BUSY` y `listen ... operation not permitted`, así que no es fuente fiable para juzgar el daemon del host

Conclusión operativa:

- el núcleo del runtime local ha avanzado de verdad: ya existe supervisor residente y contrato observable para procesos locales
- el siguiente bloqueo serio no parece estar en `start/stop/status`, sino en la ejecución efectiva de `send_instruction` pendiente sobre la cola viva
- gracias a la salida nueva de `runtime ordenes`, ese frente ya se puede seguir desde Orquesta sin recurrir a inspección directa de BD

## 21:45-22:15 aprox. — corrección estructural de `send_instruction` y mailbox durable

Objetivo:

- revisar `ARQUITECTURA.md` antes de seguir tocando el núcleo
- evitar programación cíclica sobre síntomas de runtime
- fijar el contrato correcto de entrega para `send_instruction`

Contraste arquitectónico realizado:

- `ARQUITECTURA.md` confirma tres restricciones relevantes:
  - el runtime debe quedar detrás de un contrato estable de conector
  - `orquesta serve` debe ser el escritor principal
  - el acceso concurrente debe centralizarse, no dispersarse en atajos por SQLite o por handle puntual

Hallazgo importante al mirar la instancia viva desde Orquesta:

- el problema ya no era simplemente "falta un supervisor"
- `send_instruction` había funcionado muchas veces en el día, pero volvió a fallar en caliente con órdenes recientes
- además, el flujo actual trataba cada `send_instruction` como si perteneciera a un `handle` concreto
- cuando ese handle quedaba viejo o el runtime aún no estaba listo, la orden podía:
  - fallar de forma terminal
  - o incluso volver a crear mailbox duplicada
- la causa viva observada en `Codex1` fue concreta:
  - `session_resume fallo: chdir /tmp/orquesta-validate-launch: no such file or directory`
  - es decir, la entrega seguía intentando ejecutar sobre un `working_dir` histórico de validación, no sobre la worktree/runtime activos

Conclusión de diseño:

- eso sí era programación cíclica potencial:
  - seguir añadiendo supervisión sin corregir la semántica de entrega
  - o seguir parcheando reinicios/handsoff sin arreglar la intención duradera del mailbox
- la solución correcta en este punto era redefinir `send_instruction` como intención durable del control plane, no como envío oportunista a un PID o handle viejo

Cambios aplicados:

- `db/controlplane_entities.go`
  - `send_instruction` ahora rebindea al handle activo del agente/proyecto si la orden apunta a uno obsoleto
  - si la orden proviene del mailbox y el runtime todavía no está entregable, la orden se reencola con `backoff` en lugar de quedar `fallida`
  - si falla `session_resume`, la orden nacida del mailbox también se reencola y conserva trazabilidad del error
  - se evita duplicar mailbox cuando el mensaje original ya existe y solo falta una entrega válida
- `db/controlplane_entities_test.go`
  - test nuevo: reencolado sin duplicar mailbox cuando no hay handle entregable
  - test nuevo: reencolado cuando falla `session_resume`
  - test nuevo: rebind al handle activo para evitar `working_dir` obsoleto en `session_resume`

Verificación:

- `go test ./db -run 'TestRuntimeOrderSendInstruction(...)' -count=1` OK
- `go test ./cmd -run 'TestProcesarRuntimeMailbox(...)' -count=1` OK
- `go build -o ./orquesta .` OK

Lectura operativa:

- este cambio no cierra todavía todo el núcleo de orquestación
- pero sí corrige un contrato central:
  - el mailbox vuelve a comportarse como cola durable
  - `send_instruction` deja de ser frágil por apuntar a un runtime puntual
  - la entrega en caliente queda mejor alineada con la arquitectura de Orquesta y con el objetivo de gobierno estable sobre agentes vivos

## 22:45 aprox. — cierre del hueco real del supervisor local

Contexto de diseño:

- se revisó `ARQUITECTURA.md` como referencia de contrato
- la pieza correcta no es otro parche en `db`, sino un propietario residente del runtime en `internal/controlruntime`

Hallazgo importante:

- en el árbol actual ya existía una primera implementación del supervisor local
- no era humo: registra procesos locales, distingue modo `resident`/`attached` y ya alimenta `ConsultarEstadoLocal`
- el hueco serio que seguía abierto era otro:
  - el supervisor no exponía pronto el `external_session_id` de Codex como primera fuente canónica
  - por eso `session_resume` podía tardar demasiado en activarse y el mailbox seguía degradando a backlog

Decisión aplicada:

- completar el supervisor local existente en vez de rehacerlo
- hacer que el supervisor residente sondee y cachee el `external_session_id` de Codex durante la ventana inicial de arranque
- hacer que `DetectExternalSessionID` consulte primero el estado del supervisor local antes de volver a escanear el almacén de sesiones

Objetivo operativo:

- reducir el tiempo en que un runtime Codex vivo queda en `bootstrap_only` efectivo por falta de identificación de sesión
- facilitar que el control plane pase antes a `session_resume`
- evitar volver al patrón cíclico de mailbox pendiente + restart oportunista

## 23:10 aprox. — consolidacion de doctrina canonica unica

Motivo del cambio:

- la arquitectura y las reglas del proyecto estaban repartidas entre demasiados `.md`
- eso hacia demasiado facil que un agente trabajase con una lectura parcial y perdiese el contrato del sistema
- antes de seguir con `SQLITE_BUSY` y server-first habia que fijar una fuente unica y obligatoria

Decisión aplicada:

- se crea `docs/BIBLIA_APP_ORQUESTA.md` como doctrina canonica unica de la app
- esa biblia consolida arquitectura, politicas, runbooks, vision, roadmap y la direccion de producto ya estable
- se deja explicitado que:
  - la doctrina estatica vive en la biblia
  - el estado vivo de tareas, propuestas, sesiones y runtimes se consulta en Orquesta
  - si hay conflicto entre documentos, manda la biblia hasta alinear el resto

Cambios aplicados:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nuevo documento canonico
- `docs/00_INDICE.md`
  - la biblia entra en el indice como referencia primaria
- `ARQUITECTURA.md`
  - pasa a declarar la biblia como consolidacion canonica
- `docs/manual_programador.md`
  - obliga a leer la biblia antes de extender flujos operativos
- `db/runtime_bootstrap_prompt.go`
  - el bootstrap de agentes exige leer la biblia
- `cmd/mcp.go`
  - el briefing MCP incluye la referencia canonica
- `cmd/sesion.go`
  - `sesion inicio` muestra la ruta de la doctrina canonica

Efecto buscado:

- cortar la deriva entre documentos y conversaciones
- evitar que agentes autonomos construyan arquitectura a partir de recuerdos parciales
- fijar una base estable para el siguiente trabajo estructural sobre persistencia, single-writer y eliminacion de `SQLITE_BUSY`

## 23:40 aprox. — raiz estructural de SQLITE_BUSY en el arranque del daemon

Hallazgo confirmado:

- el problema no era solo contencion de SQLite ni "muchas escrituras"
- `serve` y `server run` podian tocar la BD demasiado pronto, incluso antes de saber si el proceso iba a ser realmente el daemon valido
- en los logs se veia el patron exacto:
  - `database is locked (SQLITE_BUSY)`
  - `attempt to write a readonly database`
  - y despues fallo de `listen tcp`
- eso delata arranques fantasma o no validos que primero abren persistencia y solo despues descubren que no pueden poseer el puerto o que van con flags de recuperacion heredadas

Decision aplicada:

- dejar de abrir BD desde el arranque generico de CLI para `serve` y `server run`
- reservar primero el listener del servidor
- solo el proceso que ya posee el puerto abre la BD y arranca el control plane
- el hijo autolanzado del daemon deja de heredar `ORQUESTA_FORCE_LOCAL` y `ORQUESTA_FORCE_LOCAL_DB`

Cambios aplicados:

- `cmd/root.go`
  - `serve` y `server run` dejan de abrir BD via `cobra.OnInitialize`
- `cmd/servidor_unificado.go`
  - el listener se reserva antes de `ensureServerDBOpen()`
  - el servidor sirve con `Serve/ServeTLS` sobre listener ya poseido
- `cmd/server.go`
  - el hijo autolanzado limpia flags de recuperacion local antes de arrancar
- `cmd/root_gating_test.go`
  - se actualizan expectativas para que `serve/server run` no requieran BD previa
- `cmd/servidor_unificado_test.go`
  - test basico del listener previo al arranque

Doctrina consolidada:

- no volver a aceptar un servidor que abra SQLite antes de demostrar que es el dueño legitimo del listener
- no volver a mezclar modo `server` con modo `recovery/local`

## 00:10 aprox. — validacion viva del daemon corregido

Validacion operativa realizada:

- se recompila `./orquesta` con la correccion de arranque
- se comprueba que un segundo `server run` sobre `127.0.0.1:16543` ya falla por `bind: address already in use`
- y ya no aparece el patron previo de tocar SQLite antes del bind

Hallazgo adicional:

- el daemon viejo seguia vivo sin `statefile` del scope actual
- por eso `server doctor` podia ver `healthz` pero no encontraba el estado local en disco
- no era ya un problema del nuevo arranque sino un proceso previo levantado con contrato viejo

Operacion aplicada:

- se reinicia el daemon con el binario nuevo
- queda residente en `127.0.0.1:16543`
- publica `statefile` en `/tmp/orquesta-localrpc-2ea48e3b1141.json`
- `./orquesta server doctor` vuelve a dar `Health RPC: OK`
- `./orquesta status` vuelve a responder por servidor

Evidencia final:

- `server doctor`
  - `State: pid=3196890 addr=127.0.0.1:16543 scope=2ea48e3b1141 db=/home/alberto/Trabajo/orquesta/orquesta.db`
  - `Health RPC: OK`
- log del daemon
  - solo banner de arranque
  - sin `SQLITE_BUSY`
  - sin `attempt to write a readonly database`

Conclusion:

- queda cerrada la causa estructural detectada hoy para `SQLITE_BUSY` en el arranque del daemon
- a partir de aqui el siguiente trabajo ya no es "seguir probando arranques", sino seguir retirando rutas hibridas y cerrar el resto del server-first/single-writer desde una base estable

## 00:25 aprox. — `server status/stop` ya no dependen ciegamente del statefile

Hallazgo:

- puede existir un daemon sano que responda `healthz` aunque el `statefile` falte o se haya quedado fuera de sincronizacion
- si `status` y `stop` dependen solo del `statefile`, se convierten en falsos negativos operativos y empujan a rescate manual innecesario

Decision:

- aceptar `healthz` como fallback de autoridad solo para inspeccion y parada
- ese fallback solo es valido cuando `scope` y el backend activo coinciden con el proceso actual
- la comparacion correcta no es una `DBPath` “especial”, sino `storage_driver + storage_target`
- `statefile + healthz` siguen siendo el estado normal esperado; el fallback no sustituye la coherencia, solo evita operar a ciegas

Cambios:

- `cmd/server.go`
  - `loadServerInfoWithHealthFallback` valida `health.OK`
  - conserva contexto del error de `statefile`
  - reconstruye `ServerInfo` desde `healthz` para `server status` y `server stop`
- `cmd/server_health_fallback_test.go`
  - caso de recuperacion por `healthz`
  - rechazo por `scope` ajeno
  - rechazo por `storage target` ajeno
  - rechazo por `storage driver` ajeno

Validacion:

- `go test ./cmd -run 'Test(LoadServerInfoWithHealthFallbackUsaHealthzSiFaltaStatefile|LoadServerInfoWithHealthFallbackRechazaScopeAjeno|LoadServerInfoWithHealthFallbackRechazaDBAjena|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal)' -count=1` => OK

## 21:25 aprox. — el contrato del daemon deja de razonar en terminos de SQLite

Hallazgo:

- varios checks de `server` seguian comparando `DBPath` como si el backend fuese siempre SQLite
- eso era una fuga conceptual: la persistencia ya no debe validarse por “ruta sqlite”, sino por conector/backend activo

Decision:

- el estado y `healthz` del daemon pasan a publicar y validar `storage_driver` y `storage_target`
- `db_path` queda solo como campo de compatibilidad hacia atras
- los comandos `server status`, `server stop` y `server doctor` deben mostrar y verificar el backend real, no asumir SQLite

Cambios:

- `internal/rpclocal/types.go`
  - añade `storage_driver` y `storage_target` a `State` y `HealthResponse`
- `internal/rpclocal/server.go`
  - publica esos campos en `healthz`
- `cmd/server.go`
  - valida `scope + storage_driver + storage_target`
  - deja de comparar solo `DBPath`
  - `server doctor` y `server status` muestran `Storage`
  - el `statefile` nuevo se guarda con metadatos de backend
- `cmd/persistencia.go`
  - `persistencia info` deja de mostrar `DB objetivo`
  - pasa a exponer `Storage driver` y `Storage target`
- `cmd/server_health_fallback_test.go`
  - cubre recuperacion desde `healthz`
  - rechazo por `scope` ajeno
  - rechazo por `storage target` ajeno
  - rechazo por `storage driver` ajeno
- `cmd/persistencia_test.go`
  - valida la nueva salida backend-agnostica
- `cmd/architecture_test.go`
  - deja de tolerar `db.CurrentDBPath()` en `persistencia.go`
  - el uso de `db_path` queda restringido a compatibilidad puntual del servidor

Validacion:

- `go test ./cmd ./internal/rpclocal -run 'Test(LoadServerInfoWithHealthFallback|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal|BaseURL)' -count=1` => OK
- `go test ./cmd -run 'TestCmdNoUsaSQLDirectoNiAperturasFueraDeExcepcionesControladas' -count=1` => OK
- reinicio controlado del daemon con el binario nuevo
- `./orquesta server doctor` =>
  - `Storage driver: sqlite`
  - `Storage target: /home/alberto/Trabajo/orquesta/orquesta.db`
  - `Health RPC: OK`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta server status` =>
  - `Statefile: ausente; usando healthz del daemon activo`
  - `Storage: sqlite /home/alberto/Trabajo/orquesta/orquesta.db`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta persistencia info` =>
  - `Descubrimiento: fallback a healthz por statefile ausente`
  - `Ruta activa: servidor local`
- `ORQUESTA_SERVER_INFO=/tmp/orquesta-missing-state.json ORQUESTA_SERVER_ADDR=127.0.0.1:16543 ./orquesta status` =>
  - sigue resolviendo el daemon por server-first
  - responde sin caer a modo local ni exigir rescate manual

Limpieza asociada:

- se elimina `scripts/arrancar_codex1_orquestador.sh`
- motivo: hardcodeaba una BD recuperada en `/tmp` y un daemon viejo en `:16546`
- no era el camino canonico actual y solo podia reintroducir operativa paralela y confusion
- se eliminan `scripts/orquesta-vigilante.service` y `scripts/vigilante.sh`
- motivo: mantenian un segundo plano de control separado para `antigravity`, incompatible con el daemon unico server-first
- `scripts/instalar_servicio.sh` deja de instalar o anunciar ese vigilante legacy
- `scripts/backup_bd.sh` y `scripts/verificar_bd.sh` se declaran y validan ya como utilidades solo-SQLite
- `scripts/instalar_servicio.sh` solo instala el timer de backup cuando el backend resuelto es `sqlite`
- `scripts/orquesta-backup.service` deja de hardcodear `orquesta.db` y `ORQUESTA_DB` en la unit
- motivo: no dejar utilidades de mantenimiento vendidas como genericas cuando no cubren MySQL/Postgres

## 00:45 aprox. — el modo local de recuperacion deja de poder escribir por error

Hallazgo:

- aunque la doctrina ya decia que `ORQUESTA_FORCE_LOCAL_DB` era solo para recuperacion, el init generico de CLI todavia podia abrir la BD en escritura para comandos mutantes o de negocio no cubiertos por la whitelist de diagnostico
- eso convertia el modo de recuperacion en una puerta trasera de escritura lateral y reabria riesgo de deriva y contencion

Decision:

- el modo local de recuperacion pasa a ser estrictamente de solo lectura
- solo se permiten comandos de diagnostico explicitamente cubiertos (`status` y familia `runtime` de inspeccion)
- el resto de comandos deben fallar con error claro y exigir servidor

Cambios:

- `cmd/local_recovery.go`
  - nueva validacion explicita de comandos permitidos en recuperacion
  - nuevo error canonico para comandos no cubiertos
- `cmd/root.go`
  - `openDBForCommand` ya no puede abrir la BD en escritura cuando hay `ORQUESTA_FORCE_LOCAL_DB` o `--local`
- `cmd/cliente_servidor.go`
  - `ensureLocalDB` aplica la misma restriccion en el fallback local de la capa API
- `cmd/local_recovery_test.go`
  - tests para rechazo de mutaciones o listados no diagnosticos en recuperacion

Validacion:

- `go test ./cmd -run 'Test(ShouldOpenRecoveryReadOnlyDB|OpenDBForCommandRechazaMutacionesEnRecuperacionLocal|EnsureLocalDBRechazaComandosNoDiagnosticosEnRecuperacionLocal|LoadServerInfoWithHealthFallbackUsaHealthzSiFaltaStatefile|LoadServerInfoWithHealthFallbackRechazaScopeAjeno|LoadServerInfoWithHealthFallbackRechazaDBAjena|ShouldDelegateToLocalServer|ShouldDelegateToLocalServerExcluyeComandosDeRecuperacion|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal|EscucharServidorUnificadoReservaListenerAntesDelArranque|NormalizarAddrServidorLocal)' -count=1` => OK
- `ORQUESTA_FORCE_LOCAL_DB=1 ./orquesta runtime ordenes --agente Codex1` => sigue funcionando
- `ORQUESTA_FORCE_LOCAL_DB=1 ./orquesta tarea listar` => ahora falla con error explicito y no toca la ruta de negocio local

## 22:10 aprox. — alineadas las tareas vivas con la biblia y la arquitectura server-first

Hallazgo:

- varias tareas abiertas seguian demasiado escuetas o no reflejaban todavia la doctrina consolidada en `BIBLIA_APP_ORQUESTA.md`
- eso dejaba margen para que un agente interpretase "auditoria" o "integracion" como permiso para reabrir caminos locales, scripts legacy o semantica SQLite-first

Decision:

- las tareas vivas de este frente deben llevar dentro de Orquesta la misma regla canonica que las OP y la biblia
- si una tarea toca persistencia, web/OpenClaw o git/worktree, la nota debe fijar explicitamente el contrato server-first y prohibir vias paralelas

Cambios:

- se revisan las tareas activas `409-426`
- ya estaban alineadas o suficientemente concretas `409`, `410`, `411`, `412`, `421`, `422`, `423` y `424`
- se añaden notas canonicas nuevas a:
  - `#413` para fijar `git/worktree/merge` solo por `gitoperaciones` y sin limpieza destructiva ni rutas manuales paralelas
  - `#425` para fijar `web/openclaw` solo sobre contratos server-first del daemon, con diagnostico/persistencia/control plane visibles por API real
  - `#426` para fijar la auditoria de `git/worktree/merge` sobre el servicio oficial de Orquesta y exigir eliminacion de caminos legacy superados

Validacion:

- `./orquesta tarea nota 413 ...` => `✓ Nota añadida a tarea #413`
- `./orquesta tarea nota 425 ...` => `✓ Nota añadida a tarea #425`
- `./orquesta tarea nota 426 ...` => `✓ Nota añadida a tarea #426`

## 2026-03-31 00:xx aprox. — fijada en doctrina la semantica correcta de `runtime_mailbox`

Hallazgo:

- `runtime_mailbox` no es una cola accidental ni una mejora optativa; viene de la doctrina de autogestion supervisada y de los repos de referencia como primitive persistente entre agentes, sesiones y handoff
- el riesgo real no estaba en la existencia de la mailbox, sino en interpretar demasiado pronto una entrega como consumida

Decision:

- se fija en la biblia que `runtime_mailbox` se mantiene como primitive persistente
- se fija tambien que un mensaje solo puede darse por consumido cuando Orquesta confirma entrega real por el camino oficial de control
- los mensajes supersedidos si pueden consumirse como tales, pero no el mensaje vigente antes de entrega efectiva

Cambios:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nueva seccion `Mailbox persistente y acuse real`

Validacion:

- revision doctrinal cruzada con `ARQUITECTURA.md`, `docs/op_087_autogestion_supervisada_agentes.md`, `docs/op_095_orquestacion_mixta.md` y `docs/analisis_repos_control_agentes_2026-03-23.md`

## 2026-03-31 00:xx aprox. — regla de proceso endurecida en la biblia

Hallazgo:

- hacia falta dejar por escrito una regla de trabajo mas estricta para evitar cambios de arquitectura o nucleo hechos antes de revisar toda la documentacion del frente

Decision:

- se fija en la biblia que no se toca arquitectura, control plane, persistencia, mailbox, handoff, runtime ni semantica operativa sin estudiar antes biblia, arquitectura, politicas, OP, runbook y estado vivo

Cambios:

- `docs/BIBLIA_APP_ORQUESTA.md`
  - nueva seccion `Regla previa obligatoria antes de tocar codigo`

## 23:10 aprox. — el supervisor local pasa a latir contra el daemon y deja de ser solo observador

Hallazgo:

- el supervisor local de `internal/controlruntime` ya sabia observar PIDs, detectar `external_session_id` y controlar señales, pero no informaba al daemon del estado del runtime
- eso seguia dejando la continuidad demasiado apoyada en que el propio runtime llamase a `agente tick`, justo lo contrario de la doctrina "el server orquesta a los agentes"
- en la práctica eso facilita sesiones zombis, heartbeats perdidos y handoffs/watchdogs disparados por falta de pulso del proceso real

Decision:

- el supervisor local debe emitir `heartbeat` y `finalizacion` al daemon en nombre del runtime supervisado
- la activacion de esa supervision orquestada no debe ocurrir antes de que la sesion/runtime/handle existan en Orquesta, para no abrir una carrera en el arranque

Cambios:

- `internal/controlruntime/supervisor_local.go`
  - nuevo contrato `SupervisorSignal` + `SetSupervisorSignalHandler`
  - el supervisor residente ya guarda `agente` y `proyecto`
  - nueva activacion explicita `ActivarSupervisionOrquestada(...)`
  - nuevo bucle de señales: heartbeat periodico y señal final al salir el proceso
- `internal/controlruntime/pty_local.go`
  - el arranque local inyecta `agente` y `proyecto` en metadata del supervisor/manifest
- `db/controlplane_entities.go`
  - tras crear sesion/handle/runtime y actualizar el handle del arranque, se activa la supervision orquestada del runtime local
- `cmd/controlruntime_hooks.go`
  - el daemon registra el hook del supervisor y traduce cada señal a `agentesService.ProcessTick(...)`
  - se fija `CuotaPct=100` para no reinterpretar una salida del proceso como auto-pausa por cuota

Validacion:

- `go test ./internal/controlruntime -run 'Test(ArrancarPlanYEnviarInstruccionProceso|ArrancarPlanLocalRespetaOverrideCanSendInput|ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|DetectExternalSessionIDUsaSupervisorLocalResidente|EnviarInstruccionSesionResumeUsaCodexPerfil)' -count=1` => OK
- `go test ./db -run 'Test(RuntimeOrderSendInstructionSessionResume|RuntimeOrderSendInstructionSessionResumeRecuperaWorkingDirPreferida)' -count=1` => OK
- `go test ./cmd -run 'TestProcessSupervisorSignalIgnoraSenalesIncompletas' -count=1` => OK
- nota: una batería más amplia `go test ./cmd -run 'Test(ControlPlane.*|Agente.*|Status.*)'` sigue mostrando rojos previos en tests antiguos de recuperación local de `status`; no son consecuencia de este cambio de supervisor

## 2026-03-31 00:24 aprox. — `stop` reconcilia tambien sesiones abiertas stale

Hallazgo:

- un agente podia dejar de verse activo en `/api/agentes` y `orquesta status`, pero conservar una fila `sesiones.activa=1` huérfana
- el caso real salio con `Codex6` y `antigravity`
- la causa era precisa: `ejecutarRuntimeOrderStop()` buscaba la sesion con `GetSesionActiva(...)`, que filtra por heartbeat operativo reciente
- si la sesion seguia abierta pero stale, el stop completaba runtime y handle, pero no llegaba a aparcar/cerrar la sesion

Decision:

- `stop` debe reconciliar cualquier sesion abierta del agente, no solo la sesion que siga operativa
- la regla correcta es: si el runtime se detiene, la sesion abierta correspondiente no puede quedar viva por haber perdido ya el heartbeat

Cambios:

- `db/controlplane_entities.go`
  - `ejecutarRuntimeOrderStop()` pasa de `GetSesionActiva(...)` a `GetSesionAbierta(...)`
- `db/controlplane_entities_test.go`
  - nuevo test para sesion stale con `external_session_id`, verificando que tras `stop` queda `pausada` e `inactiva`

Validacion:

- el caso vivo deja de apoyarse en una "sesion activa fantasma" para agentes fuera de flota
- `orquesta status` vuelve a mostrar solo `Codex1-5` como activos

## 2026-03-31 00:3x aprox. — `runtime purgar-handles` deja de depender de reset roto y clasificacion hibrida

Hallazgo:

- el fallo visible de `runtime purgar-handles` mezclaba dos problemas distintos
- por un lado, `runtime purgar-handles` no estaba marcado como comando server-first en `commandSupportsServerMode`, asi que podia abrir la BD local en el init generico del CLI
- por otro, `resetFlagSet()` reinyectaba `StringSlice` usando `DefValue` textual (`[cerrado,fallido]`), dejando flags corruptas como `"[cerrado"` y `"fallido]"`

Decision:

- los comandos runtime de mutacion soportados por API deben estar clasificados explicitamente como server-first
- el reset generico de flags debe normalizar los tipos slice en vez de reutilizar sin mas el `DefValue` textual
- ademas, el propio comando normaliza defensivamente el slice de `--estado`

Cambios:

- `cmd/cliente_servidor.go`
  - `runtime purgar-handles` pasa a formar parte del conjunto server-first
- `cmd/root.go`
  - nuevo `normalizedFlagResetValue(...)` para resetear correctamente flags slice
- `cmd/runtime.go`
  - `normalizarSliceFlags(...)` aplicado a `--estado` en `runtime purgar-handles`
- tests:
  - `cmd/cliente_servidor_test.go`
- `cmd/root_test.go`
  - y se añade regresion de `shouldBypassLocalDB` / `commandSupportsServerMode` para que `runtime purgar-handles` y mutaciones runtime criticas no vuelvan a abrir BD local

## 2026-03-31 00:5x aprox. — `session_resume` por cuota externa ya no queda como fallo ciego

Hallazgo:

- la reproduccion viva del caso de `Codex1` mostro que el problema actual de `send_instruction -> session_resume` no era `cwd` ni `external_session_id`
- el conector aceptaba `exec resume` correctamente, pero devolvia `You've hit your usage limit ... try again at ...`
- Orquesta estaba tratando ese caso como error bruto del runtime y podia dejar una orden `send_instruction` pendiente o fallida sin semantica operativa suficiente

Decision:

- cuando `session_resume` devuelve error de cuota/usage limit/rate limit del proveedor, Orquesta debe traducirlo a pausa operativa del agente
- si la orden venia de mailbox, se reencola con backoff explicito y largo, no con reintento ciego corto
- el motivo debe quedar reflejado en `estado_cuota`, `reanimar_at`, `motivo_pausa` y en la propia orden

Cambios:

- `db/controlplane_entities.go`
  - deteccion de errores de cuota externa en `send_instruction/session_resume`
  - auto-pausa del agente via `PausarAgente(...)`
  - reencolado explicito con `retry_after` largo cuando la entrega venia de mailbox
  - soporte para inferir backoff desde mensajes tipo `try again at 12:56 AM`
- `db/controlplane_entities_test.go`
  - nuevo test de quota/provider backoff en `session_resume`

Validacion:

- `go test ./db -run 'TestRuntimeOrderSendInstruction(SessionResumeRecuperaWorkingDirPreferida|PausaPorCuotaProveedor|UsaSessionResumeCodexLocal)' -count=1` => OK

## 2026-03-31 00:4x aprox. — `runtime_mailbox` deja de acumular `nudge/watchdog` eternos

Hallazgo:

- el problema visible ya no era una `runtime_order` atascada; la reconciliacion stale y el backoff de proveedor estaban funcionando
- el hueco real quedaba en `runtime_mailbox`
- `nudge` y `watchdog` si llegaban a mailbox persistente, pero la capa `procesarRuntimeMailbox*Batch()` no sabia convertir esos `kind` en `send_instruction`
- resultado: mensajes pendientes viejos durante horas aunque hubiese handle activo, especialmente en `session_resume`
- ademas, `nudge` no estaba marcado como supersedible/coalescible, asi que podia acumular varias copias del mismo empuje mientras habia una `send_instruction` abierta o un backoff largo

Decision:

- `nudge` y `watchdog` pasan a formar parte del mismo contrato de entrega persistente que `instruction` y `autonomia`
- si son coalescibles, el control plane debe consumir los supersedidos aunque todavia no pueda entregar el ultimo por existir una orden abierta
- la semantica correcta es "solo la ultima señal efimera sigue pendiente", no una pila eterna de nudges viejos

Cambios:

- `db/controlplane_entities.go`
  - `nudge` pasa a ser `runtimeMailboxKindSupersedible(...)`
- `cmd/controlplane_support.go`
  - `nudge` y `watchdog` pasan a ser kinds coalescibles y entregables por `interactive`, `session_resume` y `coordinated_restart`
  - nuevo `coalescerRuntimeMailboxPendiente(...)` para consumir supersedidos en el propio batch aunque exista una `send_instruction` abierta
  - `construirInstruccionMailboxInteractivo(...)` ya traduce `nudge/watchdog` a texto entregable
- tests:
  - `db/controlplane_entities_test.go`
  - `cmd/controlplane_support_test.go`

Validacion prevista:

- `runtime_mailbox` no debe seguir creciendo con `nudge/watchdog` antiguos para un mismo agente/proyecto
- con una `send_instruction` abierta, solo debe sobrevivir el mensaje mas nuevo del kind coalescible
- cuando el conector vuelva a poder entregar, `nudge/watchdog` deben convertirse en `send_instruction` por el camino oficial del daemon

## 2026-03-31 01:03 aprox. — respaldo canonico antes de reescribir git

Hallazgo:

- la rama local tenia un commit valido de nucleo (`62f5ad1`) pero el push a remoto fallaba porque `orquesta.db` habia entrado en el commit y GitHub rechazaba el objeto por superar 100 MB
- esa base contiene el estado vivo completo del orquestador: tareas, OP, sesiones, runtime orders, mailbox, checkpoints y auditoria

Decision:

- antes de tocar el historial git, hay que crear respaldos consistentes por el camino oficial de Orquesta
- `orquesta.db` no debe volver a formar parte del historial git; la base viva se conserva fuera del indice y las copias canonicas se guardan fuera del repo

Accion ejecutada:

- respaldo oficial externo:
  - `/home/alberto/Trabajo/backups/orquestador/2026-03-31_01-03-17.186503348_pre_amend_62f5ad1_orquesta.db.bak`
- respaldo oficial local de salvaguarda:
  - `/tmp/orquesta-backups/2026-03-31_01-03-34.704483012_pre_amend_62f5ad1_orquesta.db.bak`
- despues del respaldo, `orquesta.db` se saca del indice git para reamendar el commit sin perder la base local

## 2026-03-31 01:10 aprox. — el supervisor local deja de revivir handles por PID fantasma

Hallazgo:

- el estado vivo mostraba `runtime_handles` de `Codex1` y `Codex2` como `activo` con `last_seen` reciente, pero los PID publicados ya no existian en `/proc`
- la validacion local del supervisor se apoyaba practicamente en `kill(pid, 0)`, lo que es insuficiente para un proceso adjunto o tras reinicios/reutilizacion de PID

Decision:

- un runtime local no se considera el mismo proceso solo porque el PID responda
- el supervisor debe validar tambien identidad minima del proceso:
  - `cwd` esperada
  - firma de comando esperada (`rendered_command` / `wrapped_command`)

Cambio:

- `internal/controlruntime/supervisor_local.go`
  - nuevo endurecimiento de `snapshot()`: antes de revivir un runtime por PID vivo, valida identidad del proceso
  - nuevas ayudas `validarIdentidadProcesoLocal(...)`, `commandIdentityHints(...)` y `samePath(...)`
- `internal/controlruntime/supervisor_local_test.go`
  - cobertura de hints de identidad, comparacion de rutas y mismatch de `cwd`

Validacion:

- `go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|CommandIdentityHintsIncluyeFirmaUtil|SamePathResuelveSymlink|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1` => OK

## 2026-03-31 01:15 aprox. — SQLite vuelve a validar schema real aunque la revision ya exista

Hallazgo:

- tras reiniciar el daemon con el binario nuevo, `orquesta runtime ordenes` fallaba contra la base viva con `SQL logic error: no such column: claimed_by`
- el problema no era el modelo de dominio, sino la puerta de bootstrap SQLite: si la revision `_sqlite_bootstrap_revision` ya estaba marcada, dejaba de comprobar la estructura real y podia convivir con una `runtime_orders` incompleta

Decision:

- en SQLite no basta con “revision marcada”; siempre hay que validar la estructura real del schema si el daemon arranca con bootstrap habilitado
- la revision sigue sirviendo como marca historica, pero no puede anular la verificacion efectiva de columnas y reconstrucciones necesarias

Cambio:

- `db/backend_sqlite.go`
  - `Prepare(...)` vuelve a preguntar `sqliteBootstrapRequired(...)` siempre que `BootstrapSchema=true`
  - `sqliteBootstrapRequired(...)` deja de cortocircuitar por revision marcada y valida tablas, columnas y rebuilds reales

Validacion:

- `go test ./db -run 'TestSQLite(PrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada|BootstrapRequiredEnDBVacia|PrepareOmiteBootstrapConSchemaActualSinRevision)' -count=1` => OK
- `go build -o ./orquesta .` => OK

## 2026-03-31 01:17 aprox. — falso bug de API por slug de proyecto incorrecto

Hallazgo:

- `runtime diagnostico`, `/api/runtimes/tree`, `/api/runtime-orders` y `/api/runtime-checkpoints` devolvian `sql: no rows in result set` cuando se consultaban con `proyecto=orquesta`
- la API no estaba rota: el estado vivo mostraba que el proyecto activo en BD seguia siendo `orquestador`
- el repositorio fisico vive en `/home/alberto/Trabajo/orquesta`, pero ese path no implica que el slug operativo sea `orquesta`

Decision:

- no tocar codigo para esconder un error de filtro
- fijar en doctrina que el slug canónico se consulta en Orquesta y no se infiere del nombre del directorio

Validacion viva:

- `./orquesta proyecto listar` => proyecto activo `orquestador` con ruta `/home/alberto/Trabajo/orquesta`
- `./orquesta runtime diagnostico --agente Codex2 --proyecto orquestador --limit 5` => OK
- `./orquesta runtime ordenes --agente Codex2 --proyecto orquestador --estado pendiente` => OK

## 2026-03-31 01:2x aprox. — un batch colgado ya no congela todo el control plane

Hallazgo:

- el estado vivo mostraba `runtime_orders` y `runtime_mailbox` pendientes durante horas aunque el servidor HTTP seguia sano
- el patron real no era otra vez SQLite: una sola `send_instruction/session_resume` podia quedarse colgada y bloquear el loop entero del control plane
- `planocontrol.Runner.runControlPlane()` ejecutaba todos los batches de forma secuencial en el mismo hilo logico; si uno no devolvia, tampoco corria `runtime_orders_stale`, `runtime_mailbox` ni el resto de reconciliaciones

Decision:

- el `Runner` no puede depender de que cada batch externo devuelva bien
- cada batch del control plane debe tener timeout propio y exclusion mutua por nombre
- si un batch se queda colgado, el resto del plano de control debe seguir avanzando y el batch atascado no debe duplicarse en paralelo

Cambios:

- `planocontrol/runner.go`
  - nuevo timeout por batch (`BatchTimeout`, por defecto `45s`)
  - exclusion por nombre de batch para no lanzar duplicados mientras uno sigue en curso
  - auditoria explicita de timeout (`..._error`) en lugar de congelar el loop completo
- `planocontrol/runner_test.go`
  - nueva regresion `TestRunnerRunControlPlaneTimeoutDeBatchNoCongelaElResto`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./planocontrol -run 'TestRunner(RunControlPlaneTimeoutDeBatchNoCongelaElResto|RunControlPlaneRecuperaPanicDeBatchYSigue|RunControlPlaneAuditaTrabajoProcesado|RunControlPlaneAuditaErrores)' -count=1` => OK

## 2026-03-31 01:3x aprox. — SQLite arrancaba con schema drift si la revision ya estaba marcada

Hallazgo:

- tras reiniciar el daemon con el binario nuevo, `runtime ordenes` fallo con `SQL logic error: no such column: claimed_by (1)`
- la columna y el contrato de lease ya existian en `schema.go` y en `post_migraciones_compat.go`
- el fallo real estaba en la preparacion SQLite: las post-migraciones solo se ejecutaban cuando la revision de bootstrap no estaba marcada
- ademas, la reconstruccion legacy de `runtime_orders` recreaba la tabla sin las nuevas columnas de lease, fiando la correccion a unas post-migraciones que podian no volver a correr

Decision:

- las post-migraciones SQLite deben ejecutarse siempre que no esten desactivadas explicitamente; son idempotentes y forman parte de la compatibilidad viva
- la reconstruccion legacy de `runtime_orders` debe recrear ya la forma completa de la tabla, no una forma intermedia antigua

Cambios:

- `db/backend_sqlite.go`
  - `Prepare(...)` aplica `postMigracionesPorDriver("sqlite")` siempre que `SkipPostMigrations=false`
  - `sqliteRuntimeOrdersNeedsRebuild(...)` ya exige tambien `claimed_by`, `lease_token`, `attempt_count` y `lease_expires_at`
- `db/db.go`
  - `reconstruirRuntimeOrdersLegacy(...)` recrea `runtime_orders` con las columnas de lease y las rellena con defaults seguros durante la copia
- `db/backend_sqlite_test.go`
  - nueva regresion `TestSQLitePrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(SQLitePrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada|OpenMigraRuntimeTablesLegacy|SQLitePrepareOmiteBootstrapConSchemaActualSinRevision|SQLiteBootstrapRequiredEnDBVacia)' -count=1` => OK

## 2026-03-31 01:1x aprox. — `runtime_orders` deja de depender solo de timestamps para stale

Hallazgo:

- el control plane seguia reclamando ordenes solo con `estado + started_at`
- la recuperacion stale se apoyaba sobre todo en `started_at/updated_at`, sin contrato de lease explicita
- eso dejaba la reconciliacion a medio camino: mejor que antes, pero todavia demasiado heuristica para un supervisor server-first duradero

Decision:

- `runtime_orders` pasa a tener lease minima persistente: `claimed_by`, `lease_token`, `attempt_count`, `lease_expires_at`
- toda reclamacion de orden debe materializar esa lease
- toda finalizacion o reencolado debe liberarla
- la reconciliacion stale debe mirar primero `lease_expires_at`

Cambios:

- `db/schema.go`
- `db/post_migraciones_compat.go`
- `db/config_defaults.go`
- `db/controlplane_entities.go`
- tests:
  - `db/runtime_legacy_migration_test.go`
  - `db/controlplane_entities_test.go`

## 2026-03-31 01:2x aprox. — supervisor local y runner endurecidos contra identidades falsas y batches colgados

Hallazgo:

- el supervisor local podia considerar valido cualquier PID vivo aunque ya no correspondiese al runtime esperado
- el runner del control plane podia quedarse colgado entero si un batch no devolvia

Decision:

- el supervisor local debe validar identidad por `cwd` y firma util del comando, no solo por PID
- cada batch del runner necesita aislamiento por nombre y timeout propio para no congelar el resto del plano de control

Validacion:

- `go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|CommandIdentityHintsIncluyeFirmaUtil|SamePathResuelveSymlink|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1`
- `go test ./planocontrol -run 'TestRunner(RunControlPlaneRecuperaPanicDeBatchYSigue|RunControlPlaneTimeoutDeBatchNoCongelaElResto|SafeLoopCallRecuperaPanic)' -count=1`

## 2026-03-31 01:4x aprox. — `send_instruction` recupera entrega caliente por supervisor local sin romper la politica anti-TTY

Hallazgo:

- la cola `runtime_orders_batch` estaba viva y los handles activos de Codex tenian `stdin_path`, `supervisor_ref` y `external_session_id`
- aun asi, `send_instruction` seguia bloqueado en `bootstrap_only/session_resume` porque la decision solo miraba `RuntimeHandlePermiteSendInputInteractivo`
- eso mezclaba dos conceptos distintos: TTY interactivo generico y canal de entrada ya supervisado por el daemon

Decision:

- mantener `RuntimeHandlePermiteSendInputInteractivo=false` para Codex PTY inestable
- introducir una capacidad separada para el caso seguro: entrega caliente por supervisor local de Orquesta
- si el handle `process_pty_cli` tiene `stdin_path` y `supervisor_ref` validos, el daemon debe intentar `controlruntime.EnviarInstruccionProceso(...)` antes de degradar a `session_resume` o mailbox

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper `RuntimeHandlePermiteEntregaCalienteSupervisada`
  - `ejecutarRuntimeOrderSendInstruction(...)` ya usa esa via de control real y registra `delivery_path`
- `db/controlplane_entities_test.go`
  - regresion para exigir `stdin_path + supervisor_ref`
  - regresion para mantener fallback a mailbox cuando falta `supervisor_ref`
  - regresion para verificar entrega caliente por supervisor local aunque `can_send_input=false`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesExplicitas|RuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|RuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|RuntimeOrderSendInstructionEntregaEnCalientePorSupervisorLocalAunqueCanSendInputSeaFalse)' -count=1` => OK

## 2026-03-31 01:5x aprox. — `status` deja de mentir cuando la sesion sigue viva pero el runtime ya murio

Hallazgo:

- `Codex2` seguia apareciendo activo en `status` porque `ListarSesionesActivasOperativas()` daba prioridad al `heartbeat` reciente de la sesion
- el ultimo `runtime_handle` de la misma sesion ya estaba `fallido`, asi que el sistema mostraba como vivo a un agente sin runtime entregable

Decision:

- para presencia visible, el ultimo handle de la misma sesion manda sobre el heartbeat reciente cuando ese handle ya esta `cerrado` o `fallido`
- una sesion reciente no puede sostener por si sola un agente activo si su runtime mas reciente ya cayo

Cambios:

- `db/sesiones.go`
  - mapa del ultimo handle por agente/proyecto
  - invalidacion de sesion operativa por ultimo handle terminal reciente
  - `GetSesionActivaOperativa(...)` y `ListarSesionesActivasOperativas()` ya comparten esa precedencia
- tests:
  - `db/sesiones_test.go`
  - `cmd/api_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ListarAgentesIgnoraSesionesConHeartbeatObsoleto|SesionRecienteNoCuentaComoOperativaSiSuUltimoHandleYaFallo)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIAgentesYStatusOcultanSesionZombi|APIStatusNoCuentaSesionConHandleFallidoReciente)' -count=1` => OK
- validacion viva tras reiniciar daemon:
  - `./orquesta status` pasa de `5` a `4` agentes activos
  - `Codex2` ya no sale activo mientras `runtime handle #352` permanece `fallido`

## 2026-03-31 02:0x aprox. — las `send_instruction` stale de mailbox ya no deben reintentarse contra una sesion vieja

Hallazgo:

- tras corregir la entrega segura para Codex, seguian quedando `send_instruction` antiguas (`#80150`, `#80157`) apuntando al mismo `mailbox_id=64424`
- parte de esa deuda arrastraba `external_session_id` de una sesion anterior, aunque el agente ya hubiese arrancado una sesion nueva
- reintentarlas indefinidamente no aportaba nada: la verdad seguia en el mailbox y la orden quedaba haciendo ruido operativo

Decision:

- si una `send_instruction` procedente de mailbox trae `external_session_id` vieja y el agente ya tiene otra `external_session_id` viva, la orden se completa como `superseded`
- el mailbox permanece como verdad para bootstrap/continuidad; la orden stale deja de competir con la sesion actual

Cambios:

- `db/controlplane_entities.go`
  - nuevo helper `completarRuntimeOrderSendInstructionSesionObsoleta(...)`
  - `ejecutarRuntimeOrderSendInstruction(...)` corta antes los reintentos de mailbox stale por cambio de sesion
- `db/controlplane_entities_test.go`
  - nueva regresion `TestRuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|GuardarSesionActivaPreservaMetadataRicaDelHandle|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK

## 2026-03-31 02:0x aprox. — se corrige una falsa buena idea: `stdin` caliente a Codex PTY rompe el TUI

Hallazgo:

- la entrega caliente por supervisor local parecia funcionar porque `#80147` quedo completada y el transcript registraba `stdin`
- la traza raw del handle `352` demostro la causa real del crash:
  - `The application panicked (crashed).`
  - `byte index ... is out of bounds of 'rueba supervisor local 2026-03-31T01:26Z'`
  - origen en `tui_app_server/src/wrapping.rs:52`
- por tanto, inyectar texto directo por PTY a Codex TUI no es un canal seguro aunque exista `stdin_path` y `supervisor_ref`

Decision:

- la entrega caliente por supervisor local queda permitida solo para runtimes locales seguros
- Codex PTY sigue excluido y debe continuar por `mailbox/session_resume/bootstrap`, no por `stdin` directa
- el hallazgo se fija en la biblia para no volver a caer en programacion ciclica

Cambios:

- `db/controlplane_entities.go`
  - `RuntimeHandlePermiteEntregaCalienteSupervisada(...)` ahora excluye explicitamente `runtimeHandleUsaCodexTTYInestable(...)`
- `db/controlplane_entities_test.go`
  - se mantiene la entrega caliente para un runtime local seguro
  - nueva regresion para forzar que Codex supervisado siga cayendo a mailbox segura

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|RuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|RuntimeOrderSendInstructionEntregaEnCalientePorSupervisorLocalSeguroAunqueCanSendInputSeaFalse|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK
- validacion viva:
  - `./orquesta runtime traza --handle-id 352 --bytes 4096` mostro el `runtime_panic` de Codex tras la inyeccion antigua
  - tras la correccion, una nueva orden `#80161` ya no genero un nuevo `stdin` en transcript ni otro `runtime_panic`

## 2026-03-31 02:0x aprox. — el supervisor local rehidrata metadata rica desde el manifest del runtime

Hallazgo:

- tras reiniciar daemon o reanudar un proceso ya vivo, algunos `runtime_handles` activos quedaban con metadata generica de sesion (`cwd`, `external_session_id`, `herramienta`) aunque el runtime real ya tuviese `stdin_path`, `log_path`, `rendered_command`, `supervisor_ref` y `mailbox_delivery_mode`
- eso degradaba la verdad operativa: el proceso seguia activo, pero el control plane lo veia como handle pobre y perdia capacidad de supervision fina

Decision:

- si el supervisor local observa un proceso valido y la metadata del handle sigue pobre, debe buscar el `runtime manifest` del mismo proceso y rehidratar la metadata rica
- la rehidratacion correcta pertenece al camino oficial del daemon (`sync_status` y supervision), no a rescates manuales ni a acceso directo a BD

Cambios:

- `internal/controlruntime/supervisor_local.go`
  - rehidratacion de `descriptorSupervisorLocal` desde `trace_manifest`, `trace_dir/runtime.json` y candidatos bajo `.orquesta-runtime/.../runtime.json`
  - merge conservador contra manifest validando identidad por `pid`
- `internal/controlruntime/supervisor_local_test.go`
  - nueva regresion `TestConsultarEstadoLocalRehidrataMetadataRicaDesdeRuntimeManifest`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(ConsultarEstadoLocalDetectaSessionIDSinSobrescribirDriver|ConsultarEstadoLocalRehidrataMetadataRicaDesdeRuntimeManifest|SupervisorLocalEmiteHeartbeatYFinalizacionAlActivarse|ValidarIdentidadProcesoLocalDetectaMismatchDeCWD)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(GuardarSesionActivaPreservaMetadataRicaDelHandle)' -count=1` => OK
- validacion viva:
  - `./orquesta runtime orden-nueva Codex2 sync_status --proyecto orquestador --payload '{}'` => encolada `#80221`
  - `curl -sS 'http://127.0.0.1:16543/api/runtime-handles?agente=Codex2'` ya devuelve para el handle activo `#363` metadata rica persistida con `log_path`, `mailbox_delivery_mode`, `rendered_command` y `can_send_input=false`

## 2026-03-31 02:1x aprox. — el daemon deja de rematerializar `send_instruction` nuevas para el mismo mailbox durable

Hallazgo:

- ya habiamos cerrado el bucle dentro de una misma `send_instruction`, pero el control plane seguia fabricando ordenes nuevas para el mismo `mailbox_id`
- el caso vivo estaba en `Codex2` con `mailbox_id=64431`: la orden derivada se completaba como `mailbox_only`, pero en el siguiente ciclo volvia a nacer otra `send_instruction` sobre el mismo `handle_id=363`

Decision:

- para `runtime_mailbox`, un intento de entrega queda identificado por `mailbox_id + handle_id + external_session_id efectiva`
- si ya existe una `send_instruction` abierta para ese mismo intento, o una completada con `mailbox_only=true`, el batch no puede volver a materializar otra orden nueva
- solo se permite un nuevo intento cuando cambia el handle, la sesion efectiva o el contrato de entrega

Cambios:

- `cmd/controlplane_support.go`
  - nueva deduplicacion `existeIntentoSendInstructionMailboxParaHandle(...)`
  - `procesarRuntimeMailboxInteractivoBatch()` y `procesarRuntimeMailboxSessionResumeBatch()` dejan de crear nuevas `send_instruction` si ya hubo intento diferido para el mismo mailbox sobre el mismo handle/sesion
- `cmd/controlplane_support_test.go`
  - nuevas regresiones:
    - `TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion`
    - `TestProcesarRuntimeMailboxInteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestProcesarRuntimeMailbox(SessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion|InteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle|SessionResumeBatchRespetaLeaseBootstrapPendiente|InteractivoBatchCoalesceAutonomiaPendiente)' -count=1` => OK
- validacion viva:
  - se recompila `./orquesta`, se reinicia el daemon y `./orquesta server doctor` vuelve a `Health RPC: OK`
  - antes del reinicio ya existia la orden `#80240` para `mailbox_id=64431`; tras arrancar el daemon nuevo, `#80240` se completa a `mailbox_only`
  - despues del reinicio no aparecen ordenes nuevas `#8024x/#8025x` para ese mismo `mailbox_id=64431`; el mailbox durable sigue pendiente y la cascada de rematerializacion queda cortada

## 2026-03-31 02:2x aprox. — Orquesta gana purga segura de runtime orders terminales para pruebas

Hallazgo:

- ya existia `runtime purgar-handles`, pero no habia una via oficial para limpiar `runtime_orders` de pruebas sin tocar la BD a mano
- eso complicaba reproducir y limpiar escenarios, y empujaba a soluciones ad hoc que contradicen la doctrina server-first

Decision:

- se añade purga oficial solo para `runtime_orders` terminales
- la purga exige filtro por `agente` o `proyecto`
- solo admite estados terminales (`completada`, `fallida`, `expirada`, `cancelada`)
- si se borran ordenes, cualquier `runtime_mailbox.runtime_order_id` asociado queda a `NULL`
- no se ejecuta purga real sobre la BD viva durante esta sesión porque sería destructiva sin necesidad inmediata

Cambios:

- `db/controlplane_entities.go`
  - `PurgarRuntimeOrdersTerminales(...)`
  - validaciones y normalizacion de estados/tipos
- `runtimesapp/service.go`

## 2026-03-31 06:0x aprox. — `session_resume` vuelve a funcionar para Codex desde metadata observada y no desde handles degradados

Hallazgo:

- el runtime real de `Codex1` seguia vivo y supervisado, pero el `runtime_handle` activo podia quedar degradado con metadata pobre de sesion
- en ese estado, la deteccion de runtime Codex y la ruta `session_resume` miraban demasiado la metadata persistida del handle y demasiado poco la observacion viva del supervisor
- el efecto visible era falso: `send_instruction` quedaba completada como `mailbox_only` o `runtime no disponible para entrega inmediata` aunque el proceso real y la `external_session_id` siguiesen siendo validos

Decision:

- para runtimes locales supervisados, `session_resume` debe priorizar la metadata observada por el supervisor local cuando esa observacion sea mas rica que la metadata persistida del handle
- el reconocimiento de Codex no puede depender solo de `driver=process_pty_cli`; debe aceptar senales equivalentes en `herramienta`, `conector`, `rendered_command`, `wrapped_command` y `supervisor_driver`
- `RuntimeHandlePermiteSendInputInteractivo()` debe seguir prohibiendo la ruta insegura de `stdin` directa para Codex, incluso cuando el supervisor tenga metadata rica

Cambios:

- `internal/controlruntime/codex_resume.go`
  - `DetectExternalSessionID(...)` y `EnviarInstruccionSesionResume(...)` consultan primero el estado observado por `ConsultarEstadoLocal(...)` y reutilizan esa metadata si es mas rica/coherente
- `internal/controlruntime/pty_local.go`
  - `esRuntimeCodexLocal(...)` acepta metadatos degradados y señales equivalentes de Codex mas alla del `driver`
- `db/controlplane_entities.go`
  - el gating de entrega interactiva endurece la deteccion de handles sin canal real y evita considerar entregable a un runtime local sin `stdin_path`
- tests:
  - `internal/controlruntime/codex_resume_test.go`
  - `db/controlplane_entities_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(DetectExternalSessionIDFromCodexPerfilSessionStore|EnviarInstruccionSesionResumeUsaCodexPerfil|EnviarInstruccionSesionResumeAceptaMetadataDeSupervisorLocal|DetectExternalSessionIDUsaSupervisorLocalResidente)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(RuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesExplicitas|RuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks|RuntimeOrderSendInstructionCodexSupervisadoSigueCayendoAMailbox)' -count=1` => OK
- validacion viva:
  - `./orquesta runtime orden-nueva Codex1 send_instruction --proyecto orquestador --payload '{\"prompt\":\"SMOKE resume Codex1 2026-03-31T06:03Z\"}'` crea la orden `#80281`
  - `/api/runtime-orders?agente=Codex1` muestra `#80281` como `completada` con `control_real=true`, `delivery_path=session_resume` y `mailbox_delivery=session_resume`
  - `./orquesta runtime transcript --agente Codex1 --limit 12` registra la entrada `stdin` `SMOKE resume Codex1 2026-03-31T06:03Z`

## 2026-03-31 06:1x aprox. — el runner recupera batches expirados sin depender de goroutines zombis

Hallazgo:

- `healthz` podia seguir sano aunque un batch del control plane quedase colgado
- el timeout actual del runner devolvia control al loop, pero no liberaba `runningBatches[name]` hasta que el goroutine terminase realmente
- eso dejaba una fuga estructural: si `runtime_orders` o cualquier otro batch se colgaba una vez, los ciclos siguientes lo veian como `already_running` y dejaban de reclamarlo

Decision:

- `runningBatches` deja de ser un set por nombre y pasa a ser una lease por batch con `token` de generacion y `expires_at`
- si una lease expira, el siguiente ciclo del runner puede volver a reclamar ese batch sin esperar a que el goroutine viejo muera
- el cierre de lease debe validar el token; un goroutine viejo no puede borrar la lease de una ejecucion posterior

Cambios:

- `planocontrol/runner.go`
  - `runningBatches` pasa a `map[string]runningBatchState`
  - `beginControlPlaneBatch(...)` asigna token y caducidad por lease
  - `finishControlPlaneBatch(...)` libera solo si el token coincide
- `planocontrol/runner_test.go`
  - nueva regresion `TestRunnerRunControlPlaneReclamaBatchExpiradoEnSiguienteCiclo`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./planocontrol -run 'TestRunnerRunControlPlane(TimeoutDeBatchNoCongelaElResto|ReclamaBatchExpiradoEnSiguienteCiclo|RecuperaPanicDeBatchYSigue)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- se reinicia `./orquesta server run --addr 127.0.0.1:16543`
- `./orquesta server doctor` => `Health RPC: OK`
- `./orquesta status` vuelve a responder con `5` agentes activos y progreso `353/414`

## 2026-03-31 06:1x aprox. — la deduplicación de mailbox aprende a distinguir intento legacy de intento vigente

Hallazgo:

- tras habilitar `session_resume`, seguian quedando mailbox pendientes de `Codex1` y `Codex2` aunque ya existian `send_instruction` anteriores para el mismo `mailbox_id`
- el problema no era solo “mismo mensaje”: varias de esas ordenes viejas eran `mailbox_only` legacy, creadas antes de que el handle actual expusiera un contrato de entrega mejor
- al deduplicar solo por `mailbox_id + handle_id + external_session_id`, el daemon congelaba mensajes utiles para siempre

Decision:

- cada `send_instruction` derivada desde mailbox persiste una `delivery_attempt_signature`
- una orden `mailbox_only` con la misma firma si bloquea nuevos intentos; una orden legacy sin firma puede permitir un unico reintento cuando el contrato actual ya es mas rico

Cambios:

- `cmd/controlplane_support.go`
  - `encolarSendInstructionDesdeRuntimeMailbox(...)` persiste `delivery_attempt_signature`
  - `existeIntentoSendInstructionMailboxParaHandle(...)` deja pasar un reintento util cuando la orden legacy no llevaba firma y el handle actual si
- `cmd/controlplane_support_test.go`
  - nueva regresion `TestProcesarRuntimeMailboxSessionResumeBatchPermiteReintentoDeMailboxOnlyLegacySinFirma`
  - las regresiones de dedupe estable pasan a fijar firma explicita del intento

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestProcesarRuntimeMailbox(SessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion|SessionResumeBatchPermiteReintentoDeMailboxOnlyLegacySinFirma|InteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle)' -count=1` => OK
- validacion viva:
  - tras reiniciar el daemon, aparecen `#80287` (`Codex1`) y `#80290` (`Codex2`) con firma `session_resume|handle|session`
  - `#80290` completa por `session_resume` y el mailbox de `Codex2` queda por fin consumido (`64431` y `64424`)

## 2026-03-31 06:2x aprox. — `session_resume` deja de tener timeout por defecto de 90s y stale usa su cutoff real

Hallazgo:

- el caso vivo de `Codex1` con mailbox `autonomia` mostraba la fragilidad restante: una `send_instruction` larga podia quedarse demasiado tiempo en `ejecutando` y frenar el batch
- ademas, `ReconciliarRuntimeOrdersStale()` calculaba `cutoff` pero la query seguia seleccionando con `now`, lo que hacia menos coherente la reconciliacion

Decision:

- el timeout por defecto de `session_resume` baja a `20s`; sigue siendo configurable por metadata, pero el valor por defecto ya no puede bloquear tanto tiempo
- la reconciliacion stale debe usar el `cutoff` calculado de verdad

Cambios:

- `internal/controlruntime/codex_resume.go`
  - `codexResumeTimeout(...)` pasa a `20s` por defecto
- `db/controlplane_entities.go`
  - `ReconciliarRuntimeOrdersStale()` selecciona con `cutoff`
- tests:
  - `internal/controlruntime/codex_resume_test.go`
  - `db/controlplane_entities_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(EnviarInstruccionSesionResumeUsaCodexPerfil|EnviarInstruccionSesionResumeAceptaMetadataDeSupervisorLocal|CodexResumeTimeoutDefaultYOverride)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ReconciliarRuntimeOrdersStaleRecuperaBasicasYExpiraNoSoportadas|ReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado|RuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesExplicitas|RuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks)' -count=1` => OK
- estado vivo:
  - `Codex2` ya drena mailbox durable por `session_resume`
  - `Codex1` sigue exponiendo el siguiente cuello real: la entrega de `autonomia` larga sobre `session_resume` sigue siendo mas delicada que una instruccion corta y queda abierta como siguiente frente del nucleo

## 2026-03-31 07:3x aprox. — el `start` reconoce y consume bootstrap mailbox sin lease formal

Hallazgo:

- en arranques normales del servidor (`server_autobootstrap_supervisor`) puede haber bootstrap con mailbox pendiente aunque no exista una orden `handoff/resume` que mantenga lease formal
- en ese caso el runtime arranca bien y el `start` queda `completada`, pero el resultado seguia mostrando `mailbox_count>0` y `consumidos=0`
- eso dejaba continuidad falsa: el propio `start` ya habia levantado runtime real, pero el mailbox bootstrap seguia pendiente por no pasar nunca por `AckBootstrapRuntimeLeaseByEvidence()`

Decision:

- cuando el `start` completa con runtime real y el bootstrap solo trae mailbox/checkpoint, sin orden de relevo asociada, el propio `start` debe hacer `ack` de esos mailbox bootstrap
- ese `ack` se registra como `runtime_bootstrap_start_ack`

Cambios:

- `db/controlplane_entities.go`
  - nueva ruta `ackBootstrapRuntimeSinLeaseEnStart(...)`
  - `ejecutarRuntimeOrderStart(...)` consume mailbox bootstrap directa cuando no hay `bootstrap.Order`
- `db/controlplane_entities_test.go`
  - nueva regresion `TestEjecutarRuntimeOrderStartConsumeMailboxBootstrapSinLease`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(EjecutarRuntimeOrderStartConsumeMailboxBootstrapSinLease|ReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado|RuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva parcial:
  - el caso historico `#80302` sigue mostrando el resultado viejo porque es anterior al parche
  - no he forzado aun un `start` fresco equivalente solo para reescribir historia operativa; el cambio queda cubierto por test y listo para el siguiente arranque real
  - servicio `PurgeTerminalRuntimeOrders(...)`
- `cmd/api.go`
  - endpoint `POST /api/runtime-orders/purgar`
- `cmd/runtime.go`
  - comando `runtime purgar-ordenes`
- tests:
  - `db/controlplane_entities_test.go`
  - `runtimesapp/service_test.go`
  - `cmd/api_runtimes_test.go`
  - `cmd/runtime_test.go`
  - `cmd/cliente_servidor_test.go`
  - `cmd/status_test.go`

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestPurgarRuntimeOrdersTerminales(BorraSoloTerminalesYDesenlazaMailbox|BloqueaEstadosVivos)' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./runtimesapp -run 'TestServiceDelegatesRuntimeQueries' -count=1` => OK
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIRuntimeControlPlaneEndpoints|RuntimeControlPlaneUsaAPICuandoHayServidor|RuntimeMutacionesCriticasSoportanServerMode|ShouldBypassLocalDBConServidorDescubierto)' -count=1` => OK

## 2026-03-31T08:19:24Z - Autonomia deja de duplicar `pause` para agentes ya pausados

Problema:

- `Codex2` seguia generando una cascada de `runtime_orders pause` (`#80350`-`#80373`) pese a estar ya en `estado_cuota=enfriamiento` y con `runtime_handle` `pausado`
- el control plane solo evitaba duplicados si ya existia otra `pause` `pendiente`; en cuanto una `pause` completaba, el siguiente tick podia rematerializar otra aunque la pausa siguiese vigente

Decision:

- la pausa debe ser idempotente a nivel operativo, no solo a nivel de cola
- la autonomia debe considerar la pausa ya satisfecha si:
  - existe `pause` pendiente
  - la `sesion` ya esta `pausada`
  - el `runtime_handle` ya esta `pausado`
  - o el agente esta en `estado_cuota=enfriamiento` sin runtime entregable activo

Cambios:

- `cmd/controlplane_support.go`
  - nueva guarda `pausaAutonomiaYaSatisfecha(...)`
  - `procesarAutonomiaSesionActiva`, `procesarCierreProyectoSesion`, `procesarAparcadoAutonomoSesion` y la rama de conector no disponible ya usan la misma semantica antes de encolar `pause`
- `cmd/controlplane_support_test.go`
  - nueva regresion `TestProcesarAutonomiaAgentesBatchNoDuplicaPauseSiYaEstaPausadoPorCuota`

Validacion:

- `go test ./cmd -run 'TestProcesarAutonomiaAgentesBatch(NoDuplicaPauseSiYaEstaPausadoPorCuota|EncolaPausePorBloqueoHumano|AparcaSesionBloqueadaYPausaAsignacion|RespetaEstadoOperativoProyectoEsperandoHumano)' -count=1` => OK
- `go test ./cmd -run 'TestResetReanimacionEncola(StartCuandoNoHayRuntimePeroSiTrabajo|ResumeCuandoHandlePausado)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - daemon reiniciado con el binario nuevo (`pid=3645754`, `Health RPC: OK`)
  - `./orquesta agente tick Codex2 --proyecto orquestador` sigue devolviendo `pausar_por_cuota`
  - tras el reinicio no aparecieron nuevas `pause` por encima de `#80373`; la cascada quedo cortada

## 2026-03-31T08:25:57Z - La mailbox guidance supersede entre `nudge` y `autonomia`

Problema:

- habia mailbox pendientes viejas que no se limpiaban aunque ya hubiera guia operativa mas nueva para el mismo agente
- caso vivo claro: `Codex5` conservaba `runtime_mailbox #64332` (`nudge`) en `pendiente` aunque el runtime ya habia seguido recibiendo `autonomia` y `governance_refresh`
- la causa era que `ConsumirRuntimeMailboxPendienteSupersedido(...)` solo coalescia por `kind` exacto

Decision:

- la coalescencia de guia operativa debe ir por familia, no por kind literal
- familia `guidance`: `autonomia`, `nudge`, `watchdog`, `governance_refresh`, `skills_refresh`
- `instruction` explicita queda fuera para no borrar mensajes dirigidos por una guia generica

Cambios:

- `db/controlplane_entities.go`
  - `ConsumirRuntimeMailboxPendienteSupersedido(...)` ahora supersede por familia `guidance`
  - nuevas helpers `runtimeMailboxSupersedeFamily(...)`, `runtimeMailboxFamilyKinds(...)` y `runtimeMailboxFamilyPlaceholders(...)`
- `db/controlplane_entities_test.go`
  - nueva regresion `TestEnviarRuntimeMailboxCoalesceFamiliaGuidancePendiente`

Validacion:

- `go test ./db -run 'TestEnviarRuntimeMailboxCoalesce(WatchdogPendiente|NudgePendiente|FamiliaGuidancePendiente)' -count=1` => OK
- `go test ./db -run 'Test(RuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|RuntimeOrderSendInstructionCubiertaPorBootstrapLease|EjecutarRuntimeOrderStartConsumeMailboxBootstrapSinLease)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - daemon reiniciado con el binario nuevo (`pid=3648618`, `Health RPC: OK`)
  - `./orquesta runtime nudge Codex5 ... --kind autonomia --proyecto orquestador` => `#80378`
  - tras un ciclo del runner, `#80378` quedo `completada`
  - `runtime_mailbox #64332` paso de `pendiente` a `consumido`
  - la pendiente vigente para `Codex5` pasa a ser `#64491` (`autonomia`), no el `nudge` historico

## 2026-03-31T08:33:00Z - `send_instruction` reclama lease corta

Problema:

- una `send_instruction` colgada heredaba la `lease` general de `runtime_orders` (`120s`)
- eso dejaba el control plane demasiado tiempo con una orden en `ejecutando` aunque el camino real (`session_resume`) es corto y fragil

Decision:

- mantener `lease` larga para ciclo de vida (`start`, `handoff`, etc.)
- introducir `lease` especifica y corta para `send_instruction`

Cambios:

- `db/controlplane_entities.go`
  - nuevas helpers `runtimeOrderLeaseDurationForType(...)` y `runtimeOrderLeaseDeadlineForType(...)`
  - nueva ruta `MarcarRuntimeOrderEjecutando(...)`
  - `claimRuntimeOrderByID(...)` y `ProcesarRuntimeOrdersBatch()` usan `lease` por tipo de orden
- `db/config_defaults.go`
  - nueva config `runtime_send_instruction_lease_seconds=35`
- `db/schema.go`
  - misma clave añadida al bootstrap de config
- `db/controlplane_entities_test.go`
  - nueva regresion `TestClaimRuntimeOrderSendInstructionUsaLeaseCorta`

Validacion:

- `go test ./db -run 'Test(ClaimRuntimeOrderSendInstructionUsaLeaseCorta|ReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado|ReconciliarRuntimeOrdersStaleRecuperaLeaseExpiradaSinEsperarOtroCutoff|EnviarRuntimeMailboxCoalesceFamiliaGuidancePendiente)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - daemon reiniciado con el binario nuevo (`pid=3652622`, `Health RPC: OK`)
  - `./orquesta runtime nudge Codex3 ... --kind autonomia --proyecto orquestador` => `#80382`
  - el batch materializo `#80387 send_instruction`
  - `#80387` entro en `ejecutando` con `lease_expires_at=2026-03-31T08:33:30Z`, es decir ~35s, no ~120s

## 2026-03-31T08:4xZ - Codex supervisado separa `interactive=false` de `supervisor_local=true`

Problema:

- el nucleo seguia tratando `codex-perfil` como si toda entrega caliente fuese equivalente a `send_input` interactivo
- eso congelaba el contrato en falso: aunque el handle activo tuviese `stdin_path`, `supervisor_ref` y supervision local real, `RuntimeHandlePermiteEntregaCalienteSupervisada(...)` devolvia `false`
- el efecto visible era que `Codex3` y `Codex5` seguian cayendo a `session_resume` o dejando mailbox pendiente aunque el runtime supervisado ya estaba vivo y trazable

Decision:

- separar de forma definitiva dos conceptos distintos:
  - `can_send_input=false` sigue prohibiendo el input interactivo generico para Codex
  - `supervisor_local=true` queda permitido cuando el handle real es `process_pty_cli` y trae `stdin_path` + `supervisor_ref`
- `session_resume` deja de ser el camino preferente para un Codex ya supervisado; pasa a fallback cuando no existe entrega caliente supervisada real

Cambios:

- `db/controlplane_entities.go`
  - `RuntimeHandlePermiteEntregaCalienteSupervisada(...)` ya no excluye a Codex por ser Codex si el handle tiene supervision local real
  - `RuntimeHandleMailboxDeliveryMode(...)` prioriza el contrato observado del supervisor local antes de degradar a `session_resume`
- `db/controlplane_entities_test.go`
  - `TestRuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin`
  - `TestRuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks`
  - `TestRuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal`
- `docs/BIBLIA_APP_ORQUESTA.md`
  - queda fijado que para Codex sigue prohibido el `interactive` generico, pero no la entrega caliente supervisada por Orquesta

Validacion:

- `go test ./db -run 'TestRuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|TestRuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks|TestRuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal|TestRuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|TestRuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta|TestRuntimeOrderSendInstructionMailboxSeCompletaDejandoLaVerdadEnMailboxSiNoHayHandleEntregable' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - daemon reiniciado con el binario nuevo (`pid=3658025`, `Health RPC: OK`)
  - `./orquesta runtime orden-nueva Codex3 send_instruction --proyecto orquestador --payload '{...}'` => `#80393`
  - `./orquesta runtime ordenes --agente Codex3 | head -n 8` muestra `#80393` `completada`
  - `./orquesta runtime transcript --agente Codex3 --limit 5` registra `stdin` con `prueba directa supervisor local codex 2026-03-31T08:41Z`
  - conclusion: el camino `supervisor_local` ya funciona para Codex supervisado sin reabrir el canal `interactive`

## 2026-03-31T08:4xZ - el runner materializa mailbox durable por `supervisor_local`

Problema:

- tras habilitar la entrega caliente supervisada para Codex, seguian quedando mailbox durables pendientes en `Codex4` y `Codex5`
- la causa ya no era el handle ni el agente: el runner solo tenia batches para `interactive`, `session_resume` y `coordinated_restart`
- ademas, el nuevo camino `supervisor_local` decidia demasiado pronto con metadata persistida pobre; en vivo los handles activos tenian `trace dir` y `manifest`, pero no siempre `stdin_path/supervisor_ref` ya rehidratados en la fila antes del batch

Decision:

- introducir un batch especifico de `runtime_mailbox -> send_instruction` para `supervisor_local`
- hacer que ese batch sincronice primero el estado observado del supervisor local y solo despues evalúe el contrato de entrega
- persistir esos intentos con firma propia `supervisor_local|handle|session` para no mezclarlos con `bootstrap_only` ni con `session_resume`

Cambios:

- `cmd/controlplane_support.go`
  - nuevo `procesarRuntimeMailboxSupervisorLocalBatch()`
  - `procesarRuntimeMailboxBatch()` suma ya `interactive + supervisor_local + session_resume + coordinated_restart`
  - `runtimeMailboxDeliveryAttemptSignature(...)` usa `supervisor_local` cuando el handle admite entrega caliente supervisada pero no `interactive`
- `db/controlplane_entities.go`
  - nueva helper exportada `SincronizarRuntimeHandleSupervisado(...)`
  - primero observa y rehidrata metadata del supervisor local, luego sincroniza `external_session_id`
- `cmd/controlplane_support_test.go`
  - nueva regresion `TestProcesarRuntimeMailboxSupervisorLocalBatchEncolaSendInstructionParaCodexSupervisado`
  - ajuste de `TestProcesarRuntimeMailboxBatchMaterializaAutonomiaMailbox` para el contrato durable actual del batch general
- `docs/BIBLIA_APP_ORQUESTA.md`
  - queda fijado que `supervisor_local` es un batch propio y que no puede decidir con metadata pobre

Validacion:

- `go test ./cmd -run 'TestProcesarRuntimeMailbox(InteractivoBatchOmiteHandlesSinInputInteractivo|SupervisorLocalBatchEncolaSendInstructionParaCodexSupervisado|SessionResumeBatchEncolaSendInstructionSinConsumirMailbox|BatchMaterializaAutonomiaMailbox)' -count=1` => OK
- `go test ./db -run 'TestRuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin|TestRuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks|TestRuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - antes del cierre, `Codex4` y `Codex5` tenian mailbox pendientes `#64498` y `#64499`
  - tras el ciclo del runner, `./orquesta runtime mailbox --to Codex4 --estado pendiente` y `--to Codex5 --estado pendiente` ya no devuelven filas
  - `./orquesta runtime ordenes --agente Codex4 | head -n 10` muestra `#80397 send_instruction completada`
  - `./orquesta runtime ordenes --agente Codex5 | head -n 10` muestra `#80396 send_instruction completada`
  - `./orquesta runtime transcript --agente Codex4 --limit 4` ya registra `stdin`

## 2026-03-31T08:5xZ - el `watchdog` no revive agentes ya pausados por cuota

Problema:

- `Codex2` seguia acumulando `runtime_mailbox watchdog` pendiente aunque Orquesta ya lo tenia en `estado_cuota=enfriamiento`
- el handle vivo de `Codex2` estaba `pausado`, así que ese watchdog ya no representaba trabajo real sino deuda de cola

Decision:

- si un watchdog apunta a un agente en `enfriamiento` y el handle vigente ya está `pausado`, el runner debe consumirlo
- ese caso no debe intentar reanimar al agente ni dejar basura pendiente hasta que acabe el cooldown

Cambios:

- `cmd/controlplane_support.go`
  - `reconciliarRuntimeMailboxWatchdogSinHandleBatch()` ahora consume también watchdogs satisfechos por enfriamiento
  - nueva helper `watchdogPuedeConsumirsePorEnfriamiento(...)`
- `cmd/controlplane_support_test.go`
  - nueva regresion `TestProcesarRuntimeMailboxBatchConsumeWatchdogEnEnfriamiento`
- `docs/BIBLIA_APP_ORQUESTA.md`
  - queda fijado que el watchdog no despierta agentes ya pausados conscientemente por cuota

Validacion:

- `go test ./cmd -run 'TestProcesarRuntimeMailboxBatch(ConsumeWatchdogSinHandleActivo|ConsumeWatchdogEnEnfriamiento)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion viva:
  - antes del cambio, `Codex2` mantenia `runtime_mailbox #64496` (`watchdog`) en `pendiente`
  - tras reiniciar el daemon y dejar correr el runner, `./orquesta runtime mailbox --to Codex2 --estado pendiente` ya no devuelve filas
  - `./orquesta runtime handles --agente Codex2 | head -n 8` sigue mostrando el handle `369` como `pausado`
  - conclusion: la cola deja de mentir y el agente no se reanima por error durante el cooldown

## 2026-03-31T09:0xZ - `runtime_panic` entra en cuarentena corta antes de reanimar

Problema:

- `Codex1` llegó a registrar `runtime_panic` de TUI y, aun así, el sistema podía volver a empujar guidance al mismo worker demasiado pronto
- el batch de transcript ya evitaba `send_instruction` inmediata al worker tras `runtime_panic`, pero no dejaba una cuarentena explícita de control plane
- eso dejaba margen para reanimación/continuidad demasiado agresiva tras un crash del TUI

Decision:

- un `runtime_panic` o `runtime_crash` entra en enfriamiento corto usando el mecanismo oficial de `reanimar_at`
- no se crea otro estado nuevo: se reutiliza `estado_cuota=enfriamiento` con `motivo_pausa` explícito de `runtime_panic`
- al vencer ese tiempo, la reanimación automática existente sigue funcionando sin rutas paralelas

Cambios:

- `db/sesiones.go`
  - nueva helper `PausarAgenteHasta(...)`
- `cmd/controlplane_support.go`
  - nueva helper `enfriarAgentePorRuntimePanic(...)`
  - `procesarSignalTranscript(...)` aplica esa cuarentena antes de notificar al supervisor
- `cmd/controlplane_support_test.go`
  - `TestProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic` ahora verifica también `estado_cuota=enfriamiento`, `reanimar_at` y `motivo_pausa` con trazabilidad de `runtime_panic`
- `docs/BIBLIA_APP_ORQUESTA.md`
  - queda fijado que `runtime_panic` no puede ir seguido de guidance inmediata al mismo worker

Validacion:

- `go test ./cmd -run 'Test(ProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic|ResetReanimacionEncolaResumeCuandoHayHandlePausado|ResetReanimacionEncolaStartCuandoNoHayRuntimePeroSiTrabajo)' -count=1` => OK
- `go build -o ./orquesta .` => OK
- validacion operativa:
  - el daemon sigue sano (`./orquesta status` responde y mantiene `Codex1`, `Codex3`, `Codex4`, `Codex5` visibles)
  - no he forzado un nuevo panic en vivo para evitar meter ruido artificial en la flota; la validacion del contrato queda cerrada por test dirigido y por la observacion del incidente real previo de `Codex1`
## 2026-03-31 — server_autobootstrap no debe resembrar workers vivos

Hallazgo:

- la cascada `nudge -> send_instruction -> start` vista en `Codex3-5` no venia del mailbox general ni del tick de sesion
- el origen real eran nudges repetidos con `bootstrap_kind=server_autobootstrap`
- al reiniciar el daemon, `bootstrapServerAutonomy()` reinyectaba `esperar_o_pedir_tarea` aunque el worker ya estuviera operativo

Decision:

- el autobootstrap pasa a ser sembrado inicial, no recordatorio periodico por reinicio de servidor
- si un agente ya tiene sesion activa, handle activo o mailbox bootstrap durable pendiente para el proyecto, no se vuelve a encolar bootstrap

Codigo:

- [cmd/server_autobootstrap.go](/home/alberto/Trabajo/orquesta/cmd/server_autobootstrap.go)
- [cmd/server_autobootstrap_test.go](/home/alberto/Trabajo/orquesta/cmd/server_autobootstrap_test.go)

Validacion:

- `go test ./cmd -run 'TestBootstrapServerAutonomy(ConfiguraProyectoYArranque|NoDuplicaBootstrapEnAgenteYaOperativo)|TestAgenteYaBootstrappeadoServidorDetectaMailboxDurablePendiente' -count=1`
- tras este cambio, el siguiente reinicio del daemon debe conservar los agentes ya vivos sin reinyectarles bootstrap `server_autobootstrap`

## 2026-03-31 — el supervisor no debe recibir nudge pasivo por tick

Hallazgo:

- `Codex1` seguia recibiendo `nudge/send_instruction` cada ~1 minuto aunque el runtime estaba sano, el mailbox limpio y el bootstrap ya corregido
- el payload mostraba `accion=supervisar_proyecto` sin `supervision_cycle_kind`
- la fuente era `procesarAutonomiaSesionActiva()`, que trataba al supervisor activo como si necesitara un recordatorio generico por cada tick

Decision:

- `supervisar_proyecto` deja de emitirse desde el tick pasivo de una sesion ya activa
- la supervision del supervisor queda solo en:
  - `procesarSupervisionAutonomaBatch()`
  - señales ricas de transcript/review

Codigo:

- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)

Validacion:

- `go test ./cmd -run 'TestProcesarAutonomiaAgentesBatch(NoEncolaNudgePorContinuarTrabajoEnSesionActiva|NoEncolaNudgePorEsperarOPedirTareaEnSesionActiva|NoEncolaNudgePorSupervisarProyectoEnSesionActiva|EncolaNudgePorPropuestasPendientes)' -count=1`

## 2026-03-31 — el upsert de autonomia debe preservar timestamps operativos

Hallazgo:

- tras corregir el nudge pasivo del supervisor, `Codex1` seguia recibiendo supervision rica inmediatamente despues de reiniciar el daemon
- la causa no era el intervalo de supervision, sino que `bootstrapServerAutonomy()` hace `UpsertProjectPolicy()` al arrancar
- `UpsertProyectoAutonomia()` estaba sobrescribiendo `last_supervision_at` y `last_review_at` con `NULL` cuando el caller no los informaba

Decision:

- el `upsert` de autonomia debe preservar `last_supervision_at` y `last_review_at` existentes si el caller no los trae
- reiniciar el daemon no puede rearmar supervision rica como si nunca hubiera corrido

Codigo:

- [db/autonomia_proyecto.go](/home/alberto/Trabajo/orquesta/db/autonomia_proyecto.go)
- [db/autonomia_proyecto_test.go](/home/alberto/Trabajo/orquesta/db/autonomia_proyecto_test.go)

Validacion:

- `go test ./db -run 'Test(ProyectoAutonomiaUpsertYListarActivos|ProyectoAutonomiaUpsertPreservaTimestampsOperativosSiNoSeInforman|AutonomiaCyclesRegistrarYFiltrar)' -count=1`
- `go test ./cmd -run 'TestProcesarSupervisionAutonomaBatch(EncolaSupervisionYRegistraCiclo|ArrancaSupervisorPreferidoSinSesion|GeneraBacklogInicialSiAutoCreateTasks)' -count=1`

## 2026-03-31 — watchdog no debe sondar agentes ya enfriados

Hallazgo:

- `Codex2` estaba correctamente pausado por `usage limit de proveedor` hasta una hora concreta
- aun asi, el handoff manager seguia detectando `heartbeat_stale` y generaba `sync_status + watchdog`
- eso no desbloqueaba nada; solo metia ruido operativo sobre un agente que el propio control plane ya habia aparcado

Decision:

- un agente en `estado_cuota=enfriamiento` con handle `pausado` deja de ser candidato watchdog
- el watchdog solo sirve para estados ambiguos; no para reinterrogar un cooldown ya reconocido por Orquesta

Codigo:

- [db/handoff_manager.go](/home/alberto/Trabajo/orquesta/db/handoff_manager.go)
- [db/handoff_manager_test.go](/home/alberto/Trabajo/orquesta/db/handoff_manager_test.go)

Validacion:

- `go test ./db -run 'TestProcesarHandoffsBatch(SinHandleHaceHandoffDirecto|PorPresupuestoNoRequiereSondeo|SinHandleEscalaDirectoAHandoff|NoSondeaAgenteEnEnfriamientoPausado)' -count=1`

## 2026-03-31 — la reanimacion no puede perderse por limpiar el cooldown demasiado pronto

Hallazgo:

- el runner llamaba a `ResetReanimacion()` y ese servicio limpiaba `reanimar_at/estado_cuota/motivo_pausa` antes de intentar encolar el `resume/start`
- si la reactivacion fallaba despues, el agente quedaba fuera de cooldown pero sin haber sido reactivado, y el siguiente ciclo ya no podia reintentarlo porque la deuda habia desaparecido
- ademas, el runner auditaba `reanimar_agente` antes de conocer el resultado real

Decision:

- la reactivacion debe ejecutarse primero y solo despues limpiar el cooldown
- si la reactivacion falla, la deuda de reanimacion debe seguir visible para el siguiente ciclo
- el runner solo audita `reanimar_agente` en exito real; en fallo audita `reanimar_agente_error`

Codigo:

- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)
- [planocontrol/runner.go](/home/alberto/Trabajo/orquesta/planocontrol/runner.go)
- [planocontrol/runner_test.go](/home/alberto/Trabajo/orquesta/planocontrol/runner_test.go)

Validacion:

- `go test ./cmd -run 'TestResetReanimacion(EncolaResumeCuandoHayHandlePausado|EncolaStartCuandoNoHayRuntimePeroSiTrabajo|ConservaCooldownSiFallaReactivacion)' -count=1`
- `go test ./planocontrol -run 'TestRunnerRunReanimaciones(AuditaSoloExitoReal|AuditaErrorSinMentirExito)' -count=1`

## 2026-03-31 — status no debe mezclar tareas globales sin proyecto en el bloque operativo

Hallazgo:

- `./orquesta status` mostraba la tarea `#252` dentro de `En progreso ahora mismo`
- pero esa tarea no pertenece al proyecto operativo (`Proyecto: —`), asi que el bloque visible mentia: mezclaba backlog global sin proyecto con el estado del proyecto actual
- eso hacia parecer que `Codex2` tenia trabajo activo en `orquestador` cuando `agente tick` ya respondia que no habia tarea activa en ese proyecto

Decision:

- `tareasActivas` del resumen de estado no debe incluir tareas sin `proyecto_id`
- este ciclo corrige solo el bloque visible de tareas activas; no cambia aun los contadores globales de estado

Codigo:

- [cmd/status_service.go](/home/alberto/Trabajo/orquesta/cmd/status_service.go)
- [cmd/api_test.go](/home/alberto/Trabajo/orquesta/cmd/api_test.go)

Validacion:

- `go test ./cmd -run 'TestAPIStatus(ExponeResumenOperativoCompat|OmiteTareasActivasSinProyecto)' -count=1`

## 2026-03-31 — la mailbox zombi no puede quedar pendiente para agentes fuera de vida operativa

Hallazgo:

- la cola pendiente seguia arrastrando mensajes para `Codex6`, `antigravity` y `claude`
- esos destinatarios ya no tenian runtime activo, ni sesion activa, ni trabajo vivo, pero la deuda seguia visible como si todavia fuera entregable
- eso ensuciaba diagnostico y aparentaba trabajo pendiente donde ya no habia flota ni ciclo de vida que sostener

Decision:

- reconciliar de forma preventiva la mailbox pendiente de agentes sin vida operativa
- criterio conservador: solo se consume si el destinatario no tiene handle activo, sesion activa, asignacion activa, tareas activas ni runtime orders abiertas
- ajuste posterior: `Codex6` y `antigravity` seguian protegidos por una asignacion `activa` con nota `reactivacion_automatica`, pero ambos estaban ya fuera de la flota `Codex1-5`; esa asignacion ya no puede retener mailbox zombie
- `watchdog` queda fuera de esta regla y conserva su auditoria especifica

Codigo:

- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)

Validacion:

- `go test ./cmd -run 'TestProcesarRuntimeMailboxBatch(ConsumeAgenteSinVida|NoConsumeAgenteSinHandlePeroConTrabajo|ConsumeAgenteFueraDeFlotaConAsignacionAutomatica|ConsumeWatchdogSinHandleActivo)' -count=1`

## 2026-03-31 — el banner de script no es un runtime panic

Hallazgo:

- `Codex1` seguia entrando en `runtime_panic` tras `start + send_instruction`, pero el transcript reciente mostraba como señal `runtime_panic` una linea de sistema: `Script started on ...`
- el ingestor ya marcaba esas lineas como `stream=system`, pero aun las pasaba por el clasificador generico de `panic/crash`
- el resultado era un falso positivo que disparaba cooldown y escondia al agente sin existir un fallo semantico real del runtime

Decision:

- las lineas `Script started on ...` y `Script done on ...` se siguen persistiendo como transcript de sistema
- pero no se clasifican como `runtime_panic` ni `runtime_crash`
- la deteccion de fallos debe salir del contenido semantico real del runtime, no del wrapper `script(1)`

Codigo:

- [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [db/runtime_transcript_test.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript_test.go)

Validacion:

- `go test ./db -run 'Test(IngestarRuntimeTranscriptHandleClasificaYGeneraEventos|IngestarRuntimeTranscriptHandleNoClasificaLineaSistemaScript)' -count=1`

## 2026-03-31 — el bootstrap del supervisor Codex debe compactarse antes de entrar al PTY

Hallazgo:

- tras quitar el falso positivo de `Script started on ...`, `Codex1` seguia cayendo, pero ahora con un `runtime_panic` real

## 2026-03-31 — verificación SQLite ya no acepta `journal_mode` fuera de WAL

- el contrato documental ya decía que `journal_mode=WAL` forma parte del backend SQLite sano, pero `persistencia verificar` todavía no lo exigía de forma estricta
- ese hueco permitia que un backend writable pudiera seguir apareciendo como `ok` aun si habia caido a `DELETE` o a un modo equivalente, escondiendo una regresion que despues puede volver a producir `SQLITE_BUSY`
- se endurece `sqliteBackend.Verify` para marcar error cuando el `journal_mode` no es `wal` en modo writable
- se añade test dirigido para cubrir la degradacion intencional a `DELETE` y fijar que la comprobacion `config_sqlite` pasa a `error`
- el transcript ya muestra la causa: `tui_app_server/src/wrapping.rs:52` y `byte index ... is out of bounds of 'Orquesta: has sido arrancado como orquestador ...'`
- `Codex2` no cae porque su bootstrap ya era mas corto y acababa compactado de facto; el hueco estaba en el texto especifico del supervisor

Decision:

- el bootstrap del supervisor para runtimes Codex locales debe compactarse a la misma forma corta segura que otras guias (`supervisa proyecto actual y sigue`)
- no se inyectan parrafos largos crudos al PTY de Codex cuando el runtime local ya exige una instruccion corta y ASCII estable

Codigo:

- [internal/controlruntime/pty_local.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/pty_local.go)
- [internal/controlruntime/proceso_test.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/proceso_test.go)

Validacion:

- `go test ./internal/controlruntime -run 'TestNormalizarInstruccionProceso(CompactaParaCodexLocal|CompactaBootstrapSupervisorCodexLocal|HaceASCIIYCorta)' -count=1`

## 2026-03-31 — compactacion legacy de runtime handles sin bloquear la API

Hallazgo:

- el cambio previo ya compactaba `bootstrap_prompt` y `continuity_prompt`, pero los handles vivos antiguos seguian devolviendo metadata enorme por `rendered_command` y `wrapped_command`
- al intentar persistir esa compactacion durante `ListarRuntimeHandles`, la ruta `/api/runtime-handles` pasaba a mezclar listado con escritura y se volvia lenta o bloqueante
- la necesidad real era doble: sanear getters puntuales y mantener la vista operativa rapida

Decision:

- `GetRuntimeHandle` y `GetRuntimeHandleBySesionID` siguen reconciliando y persistiendo compactacion legacy del handle concreto
- `ListarRuntimeHandles` deja de persistir durante el listado; compacta solo en memoria para responder rapido
- `runtime_handles.metadata_json` ya no debe conservar comandos gigantes con el bootstrap entero embebido; se compactan a una forma observacional corta que preserva deteccion y `session_resume`

Codigo:

- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [db/controlplane_entities_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(GetRuntimeHandleBySesionIDCompactaMetadataLegacy|ListarRuntimeHandlesCompactaMetadataLegacy)' -count=1`
- `go build -o ./orquesta .`
- daemon reiniciado y sano: `./orquesta server doctor` => `Health RPC: OK`
- `time curl -s -m 5 'http://127.0.0.1:16543/api/runtime-handles?agente=Codex1'` => `200` en menos de `1s`
- validacion viva del handle `390` de `Codex1`: `rendered_command` ya sale como `'/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil' 'Codex1'`, `wrapped_command` compacto y `contains_bootstrap=False`

## 2026-03-31 — purga segura de runtime orders terminales antiguas

Hallazgo:

- `runtime purgar-ordenes` ya existia, pero sin corte temporal; servia para limpiar ruido, pero era demasiado agresivo para un control plane serio
- el diagnostico vivo de `Codex2` seguia mostrando miles de `send_instruction` terminales historicas aunque ya no hubiera órdenes vivas ni mailbox pendiente

Decision:

- la purga de órdenes terminales debe aceptar y propagar un cutoff temporal
- el camino oficial por CLI/API usa `older-than-minutes` y el default conservador queda en `60`
- `Get/Purge` de pruebas deben seguir yendo por daemon, no por acceso lateral a BD

Codigo:

- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [db/controlplane_entities_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities_test.go)
- [runtimesapp/service.go](/home/alberto/Trabajo/orquesta/runtimesapp/service.go)
- [runtimesapp/service_test.go](/home/alberto/Trabajo/orquesta/runtimesapp/service_test.go)
- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/runtime.go](/home/alberto/Trabajo/orquesta/cmd/runtime.go)
- [cmd/runtime_test.go](/home/alberto/Trabajo/orquesta/cmd/runtime_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(PurgarRuntimeOrdersTerminales(BorraSoloTerminalesYDesenlazaMailbox|BloqueaEstadosVivos|RespetaCreatedBefore)|GetRuntimeHandleBySesionIDCompactaMetadataLegacy|ListarRuntimeHandlesCompactaMetadataLegacy)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go test ./runtimesapp -run 'Test(PurgeInactiveRuntimeHandlesDelegatesAndAudits|PurgeTerminalRuntimeOrdersAplicaCutoffYAudita)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestRuntimeControlPlaneUsaAPICuandoHayServidor' -count=1`
- validacion viva: `./orquesta runtime purgar-ordenes --agente Codex2 --tipo send_instruction` => `1601` órdenes purgadas con el default seguro; despues `./orquesta runtime ordenes --agente Codex2 --estado fallo` ya no devuelve filas

## 2026-03-31 — startup grace para el runner del servidor

Hallazgo:

- tras varios reinicios del daemon, `healthz` respondia enseguida pero `/api/status` y `/api/runtime-handles` podian quedarse esperando justo en la primera ventana de arranque
- el `Runner` del control plane ejecutaba `salud`, `planificacion` y `control_plane` inmediatamente al hacer `Start`, compitiendo con la API server-first desde el segundo cero

Decision:

- el runner usado por el daemon server-first arranca con una `startup grace` corta antes del primer batch pesado
- el objetivo no es retrasar el control plane indefinidamente, sino permitir que listener, statefile y API entren primero
- la gracia se configura solo en `newControlPlaneRunner`; el `Runner` base mantiene comportamiento inmediato por defecto para no romper tests ni ejecuciones embebidas

Codigo:

- [planocontrol/runner.go](/home/alberto/Trabajo/orquesta/planocontrol/runner.go)
- [planocontrol/runner_test.go](/home/alberto/Trabajo/orquesta/planocontrol/runner_test.go)
- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./planocontrol -run 'Test(RunnerRunControlPlaneAuditaTrabajoProcesado|RunnerRunControlPlaneSinTrabajoNoAudita|RunnerStartRespetaStartupGrace)' -count=1`
- `go build -o ./orquesta .`
- daemon reiniciado con el binario nuevo
- validacion viva inmediata tras reinicio:
  - `time curl -s -m 5 'http://127.0.0.1:16543/api/status'` => `200` en `~1.5s`
  - `time curl -s -m 5 'http://127.0.0.1:16543/api/runtime-handles?agente=Codex1'` => `200` en `~45ms`

## 2026-03-31 — bootstrap runtime proyecta guidance pendiente y stale por edad real

Hallazgo:

- el refactor anterior habia dejado la `startup grace` en `newControlPlaneRunner`, contaminando tests y runners embebidos que necesitan ejecución inmediata
- `prepararBootstrapRuntimeAgente` habia dejado de reflejar `nudge`/`discordia` pendientes en el bundle de bootstrap si todavía no existia `runtime_mailbox` real
- la reconciliación de `runtime_orders stale` ignoraba intentos claramente viejos si `lease_expires_at` seguia futura, lo que dejaba órdenes atascadas en `ejecutando`

Decision:

- la `startup grace` pertenece al daemon server-first y se aplica en `servidor_unificado`, no en el constructor genérico del runner
- el bootstrap debe proyectar guidance pendiente modelada como `runtime_order` simple sin mutar BD ni fingir consumo
- `runtime_orders stale` debe recuperarse por lease vencida o por edad efectiva del intento; la lease no puede enmascarar un intento muerto

Codigo:

- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [cmd/servidor_unificado.go](/home/alberto/Trabajo/orquesta/cmd/servidor_unificado.go)
- [internal/bootstrapruntime/bootstrap.go](/home/alberto/Trabajo/orquesta/internal/bootstrapruntime/bootstrap.go)
- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'Test(PrepararBootstrapRuntimeAgenteInyectaHandoffYMailbox|APIControlPlaneArranqueRealConBootstrapMultilinea|ControlPlaneRunnerExponeEventoAutoGuidancePorAPI|ControlPlaneRunnerRecuperaRuntimeOrderStaleYLaProcesaEndToEnd)' -count=1`
- el bloque queda en verde
- `prepararBootstrapRuntimeAgente` vuelve a exponer la `nudge` pendiente como mailbox sintética dentro del bundle
- el runner embebido vuelve a ejecutar batches sin esperar `5s`
- una `runtime_order` vieja en `ejecutando` vuelve a `pendiente` y se procesa de verdad aunque su `lease_expires_at` siga futura

## 2026-03-31 — server-first coherente en gating y status

Hallazgo:

- seguian conviviendo dos expectativas incompatibles sobre `serve/server run`: una antigua queria abrir BD desde el gating genérico y la doctrina nueva exige reservar listener primero y abrir persistencia dentro del servidor
- `status` seguia descubriendo servidor aunque `ORQUESTA_DISABLE_SERVER_CLIENT=1`, y los tests que invocan `RunE` directo perdian el contexto de comando para recuperación local explícita

Decision:

- `serve` y `server run` no abren BD desde `commandNeedsDB`; el gating genérico debe seguir dejando esa responsabilidad al arranque del servidor
- `activeServerURL()` debe respetar `ORQUESTA_DISABLE_SERVER_CLIENT`
- `status` debe fijar su propio contexto de comando cuando se ejecuta por `RunE` directo en tests o llamadas embebidas

Codigo:

- [cmd/root.go](/home/alberto/Trabajo/orquesta/cmd/root.go)
- [cmd/root_test.go](/home/alberto/Trabajo/orquesta/cmd/root_test.go)
- [cmd/status.go](/home/alberto/Trabajo/orquesta/cmd/status.go)
- [cmd/status_remoto_compat.go](/home/alberto/Trabajo/orquesta/cmd/status_remoto_compat.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'Test(CommandNeedsDB|CommandNeedsDBWithDelegationCoverage|StatusExigeServidorSalvoRecuperacionLocal|StatusPermiteRecuperacionConForceLocal|StatusFuncionaEnRecuperacionLocalDB)' -count=1`
- el bloque queda en verde

## 2026-03-31 — cierre final de `./cmd`: review gate y política MCP

Hallazgo:

- `procesarReviewGatesBatch` estaba buscando worktrees con la vista coherente, lo que descartaba worktrees activos registrados pero aún no materializados en disco y dejaba `WorktreeID=nil` en el gate
- `ResolverPoliticaModelo` resolvía por `id` ascendente cuando dos políticas tenían misma prioridad, así que una seed vieja podía tapar un override explícito reciente en MCP

Decision:

- separar listado raw de worktrees para decisiones de gobernanza donde importa la relación registrada, no la verificación física del path
- a igualdad de prioridad, la política de modelo más reciente debe ganar

Codigo:

- [db/worktrees.go](/home/alberto/Trabajo/orquesta/db/worktrees.go)
- [cmd/controlplane_autonomia_nivel2.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_autonomia_nivel2.go)
- [db/model_policies.go](/home/alberto/Trabajo/orquesta/db/model_policies.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'TestProcesarReviewGatesBatchCreaGateYEncolaRevision' -count=1`
- `go test ./cmd -run 'TestMCPToolResuelveModeloPorPolitica' -count=1`
- `go test ./db -run 'TestResolverPoliticaModelo(EconomicaPorPerfil|ConOverridesPorProyectoYTarea)' -count=1`
- `go test ./cmd -count=1`
- `./cmd` queda completo en verde

## 2026-03-31 — cierre de suite completa: supervisor local, seeds y e2e estables

Hallazgo:

- `supervisor local` se habia estrechado bien para no tratar cualquier `PID` como runtime supervisado, pero seguia teniendo dos fugas: aceptaba señales demasiado débiles en algunos casos y, a la vez, podia mezclar estado viejo cuando un supervisor adjunto promovia una `supervisor_ref` canónica desde `runtime.json`
- los tests de `pools` seguian asumiendo una BD temporal vacía, pero `prepararDBTemporal(...)` ya siembra pools/modelos/políticas base por contrato
- los E2E de `cmd` estaban correctos funcionalmente, pero uno sufria una carrera de cleanup por árbol de procesos del shell y otro quedaba demasiado justo de tiempo bajo la carga de la suite completa
- además, varios tests de `db` estaban mutando metadata de handles vivos hasta dejarla incoherente con el proceso real, y eso hacia que la validación local los descartara como fantasmas

Decision:

- `supervisor local` solo debe activarse con identidad real de runtime local: `supervisor_ref`, `supervisor_driver`, `driver=process_pty_cli`, `trace_manifest`, `trace_dir` o manifest detectable en `.orquesta-runtime`
- `stdin_path` solo no convierte un handle en supervisor local
- cuando una sesión adjunta pasa de `pid:<pid>` a `supervisor_ref` canónica, esa identidad canónica debe refrescar los campos ricos del supervisor
- los tests de `pools` deben validar el pool objetivo (`codex`) en lugar de asumir que no existen seeds base
- los E2E deben usar `exec sleep 30` en el launcher de prueba y un margen temporal suficiente para eventos API bajo carga de suite
- los fixtures de `send_instruction` deben preservar la identidad real del runtime arrancado; si el test solo quiere simular `process_pty_cli` o `can_send_input=false`, no debe falsear `working_dir` o `rendered_command`

Codigo:

- [internal/controlruntime/supervisor_local.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/supervisor_local.go)
- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [db/model_policies.go](/home/alberto/Trabajo/orquesta/db/model_policies.go)
- [db/pools_test.go](/home/alberto/Trabajo/orquesta/db/pools_test.go)
- [db/controlplane_entities_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities_test.go)
- [db/controlplane_handoff_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_handoff_test.go)
- [cmd/controlplane_e2e_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_e2e_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./internal/controlruntime -count=1`
- `go test ./db -run 'TestGuardarPoolYListarResumen|TestProcesarRuntimeOrdersBatchSyncStatusObservaProcesoLocal|TestCrearHandoffAgenteStaleConDestinoActivoProyectoEncolaReinicioBootstrap|Test(RuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo|RuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal)' -count=1`
- `go test ./cmd -run 'Test(APIControlPlaneArranqueRealConBootstrapMultilinea|ControlPlaneRunnerExponeEventoAutoGuidancePorAPI)' -count=10`
- `go test ./... -count=1`
- la suite completa queda en verde

## 2026-03-31 — persistencia activa: mailbox con emisor lógico y SQLite en WAL

Hallazgo:

- el plano de control usa `server` y `orquesta` como emisores lógicos de `runtime_mailbox`, pero el schema seguía anclando `from_agente` a `agentes(nombre)`
- al endurecer la apertura SQLite salieron dos verdades distintas: el bug estructural no era “usar SQLite”, sino que `runtime_mailbox.from_agente` estaba mal modelado; además la verificación viva seguía reportando `journal_mode=delete` hasta reiniciar el daemon con el binario nuevo

Decision:

- `runtime_mailbox.from_agente` deja de tener FK a `agentes(nombre)`; solo `to_agente` sigue referenciando un agente real
- se añade migración/rebuild SQLite para `runtime_mailbox` legacy con ese contrato nuevo
- el adaptador SQLite fija `journal_mode=WAL` al abrir en modo lectura-escritura, pero no fuerza pragmas mutables en `mode=ro`
- no se fuerza `foreign_keys=ON` globalmente desde `Open`, porque ese cambio estaba arrastrando deuda legacy mucho más amplia y no era el bug a cerrar en este ciclo

Codigo:

- [db/schema.go](/home/alberto/Trabajo/orquesta/db/schema.go)
- [db/db.go](/home/alberto/Trabajo/orquesta/db/db.go)
- [db/backend_sqlite.go](/home/alberto/Trabajo/orquesta/db/backend_sqlite.go)
- [db/backend_sqlite_test.go](/home/alberto/Trabajo/orquesta/db/backend_sqlite_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./db -run 'TestSQLitePrepareOmiteBootstrapConSchemaActualSinRevision|TestSQLitePrepareRebuildRuntimeMailboxPermiteEmisorLogico|TestSQLite(BackendOpenFuerzaJournalModeWALAunqueElDSNFalte|BackendOpenReadOnlyNoFuerzaPragmasMutables)' -count=1`
- `go test ./db ./storage ./cmd -count=1`
- `go build -o ./orquesta .`
- reinicio del servidor por la vía oficial y validación viva:
  - `./orquesta server doctor` => `Health RPC: OK`
  - `./orquesta persistencia verificar` => `journal_mode=wal`
  - `./orquesta status` vuelve a responder por server-first

## 2026-03-31 — daemonización oficial del servidor

Hallazgo:

- el servidor ya arrancaba sano en foreground con `server run`, pero faltaba una vía oficial y robusta para dejarlo residente sin depender de `nohup`, `&` o arranques manuales externos
- además, cuando el hijo falla antes de publicar `healthz`, la ruta interna de arranque local necesitaba diagnosticar mejor el fallo y limpiar estado falso

Decision:

- se añade `server start` como comando canónico para levantar el daemon local y esperar a `healthz`
- `ensureLocalServer()` pasa a detectar muerte temprana del hijo, resumir el log y limpiar el `statefile` en fallos/timeout
- `server run` se mantiene como modo foreground; la operación persistente del orquestador deja de apoyarse en background manual

Codigo:

- [cmd/server.go](/home/alberto/Trabajo/orquesta/cmd/server.go)
- [cmd/root_test.go](/home/alberto/Trabajo/orquesta/cmd/root_test.go)
- [cmd/root_gating_test.go](/home/alberto/Trabajo/orquesta/cmd/root_gating_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'Test(CommandNeedsDB|ShouldDelegateToLocalServer|CommandNeedsDBWithDelegationCoverage|BuildLocalServerProcessEnvLimpiaRecuperacionLocal)' -count=1`
- `go build -o ./orquesta .`
- validación viva:
  - `./orquesta server start` => servidor activo
  - `./orquesta server doctor` => `Health RPC: OK`
  - `./orquesta status` => vuelve a responder por server-first

## 2026-03-31 — la supervision periodica deja de reinyectarse sobre un supervisor ya operativo

Hallazgo:

- el repique visible de `Codex1` no venia de mailbox ni de watchdog, sino de la supervision autonoma periodica
- la evidencia viva quedo aislada por API server-first:
  - `GET /api/runtime-orders?agente=Codex1` mostraba la pareja `#80638/#80639`
  - `#80638` era `nudge` con `accion=supervisar_proyecto`
  - `#80639` era su `send_instruction` derivada con el texto de supervision rica
- el batch de supervision ya respetaba `last_supervision_at`, pero una vez vencido el intervalo seguia reinyectando la misma guidance aunque el supervisor tuviera sesion operativa y handle activo en ese mismo proyecto

Decision:

- el primer bootstrap de supervision sigue permitido
- a partir de ahi, si `last_supervision_at` ya existe y el supervisor sigue operativo de verdad en ese proyecto, `procesarSupervisionAutonomaBatch()` no debe volver a encolar `supervisar_proyecto`
- la supervision periodica solo debe volver a inyectarse cuando el supervisor ya no este operativo o cuando haya señales reales que lo justifiquen

Codigo:

- [cmd/controlplane_autonomia_nivel2.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_autonomia_nivel2.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'TestProcesarSupervisionAutonomaBatch(EncolaSupervisionYRegistraCiclo|ArrancaSupervisorPreferidoSinSesion|NoRepiteSupervisionPeriodicaConSupervisorOperativo)' -count=1`
- `go build -o ./orquesta .`
- verificacion viva del emisor:
  - `curl -sf http://127.0.0.1:16543/api/runtime-orders?agente=Codex1`
  - la ultima pareja previa al cambio seguia siendo `nudge(supervisar_proyecto) -> send_instruction`
  - tras reiniciar con el binario nuevo, el criterio de corte ya queda cubierto por test y por el emisor identificado; la siguiente comprobacion viva util es dejar pasar el siguiente intervalo de supervision sin que aparezca una pareja nueva equivalente

## 2026-03-31 — el watchdog deja de declarar stale a un runtime que sigue vivo

Hallazgo:

- `Codex2` seguia recibiendo `sync_status + watchdog` con motivo `heartbeat hace 30 min` aun teniendo runtime y handle activos
- la evidencia viva salia por API:
  - `GET /api/runtime-orders?agente=Codex2` mostraba `#80635/#80636/#80637`
  - `#80635` era `sync_status` por `watchdog_heartbeat_stale`
  - `#80636` era el `nudge` watchdog derivado
  - el mismo payload mostraba `runtime_id=392`, `handle_id=389`, `logical_state=esperando_io`, `process_alive=true`
- la causa estructural estaba en `DetectarAgentesAgotados()`: solo miraba `sesiones.heartbeat_at` y podia considerar “agotado” a un agente cuyo runtime seguia vivo y con actividad reciente

Decision:

- el watchdog de handoff sigue usando la sesion como señal principal, pero antes de declarar stale debe consultar la actividad real del runtime activo
- si el runtime activo del handle vigente tiene `last_event_at` o `last_heartbeat_at` recientes, no se encola `sync_status/watchdog`
- timestamps administrativos como `updated_at` no cuentan como actividad viva; solo valen señales reales del runtime

Codigo:

- [db/handoff_manager.go](/home/alberto/Trabajo/orquesta/db/handoff_manager.go)
- [db/handoff_manager_test.go](/home/alberto/Trabajo/orquesta/db/handoff_manager_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./db -run 'TestDetectarAgentesAgotados(DevuelveAgentesConHeartbeatAntiguo|SinHandleActivoNoRequiereSondeo|IgnoraHeartbeatReciente|IgnoraHeartbeatSesionStaleSiRuntimeSigueActivoReciente|IncluyePresupuestoCriticoConHeartbeatReciente|IgnoraPresupuestoObsoletoConHeartbeatReciente)' -count=1`
- `go build -o ./orquesta .`
- verificacion viva del caso que motivó el cambio:
  - `curl -sf http://127.0.0.1:16543/api/runtime-orders?agente=Codex2`
  - el problema real quedo aislado como falso stale de sesion con runtime vivo

## 2026-03-31 — el runner no debe handoffear a un agente cuyo proceso sigue vivo

Hallazgo:

- la suite completa detectó una contradiccion entre doctrina y e2e: `planocontrol.TestRunnerWatchdogYHandoffConProcesoVivo` seguia esperando `sync_status + watchdog + handoff` aunque el proceso local permaneciera vivo
- ese test era compatible con el comportamiento viejo, pero ya no con la regla correcta fijada en el ciclo anterior: un runtime activo y reciente invalida el stale de sesion

Decision:

- el escenario e2e correcto pasa a ser el inverso: con proceso vivo y runtime activo reciente, el runner no debe encolar watchdog ni handoff
- la tarea debe permanecer `en_progreso` con el agente original y el handle debe seguir `activo`
- el relevo automatico sigue cubierto por los tests de presupuesto y por los escenarios donde el runtime ya no tiene actividad real

Codigo:

- [planocontrol/runner_e2e_test.go](/home/alberto/Trabajo/orquesta/planocontrol/runner_e2e_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./planocontrol -run 'TestRunner(NoEscalaWatchdogNiHandoffConProcesoVivo|HandoffPreventivoPorPresupuesto)' -count=1`
- `go test ./... -count=1`
- suite completa en verde

## 2026-03-31 — MCP deja de ser solo stdio y pasa a superficie server-first del daemon

Hallazgo:

- MCP ya existia de verdad en `cmd/mcp.go`, con resources, prompts y tools cubiertos por tests, pero solo salia por `orquesta mcp serve` en `stdio`
- eso dejaba OP-088 a medio cerrar como producto: habia nucleo MCP, pero no una puerta server-first del daemon para gestores externos HTTP

Decision:

- reutilizar el mismo nucleo MCP y exponerlo por `/api/mcp` en el servidor oficial
- `GET /api/mcp` describe el endpoint y las versiones soportadas
- `POST /api/mcp` acepta JSON-RPC 2.0 y reutiliza `resources`, `prompts` y `tools` sin duplicar logica
- la via HTTP se deja explicita como `stateless`, para no fingir una segunda semantica de sesion MCP separada del daemon

Codigo:

- [cmd/mcp.go](/home/alberto/Trabajo/orquesta/cmd/mcp.go)
- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/mcp_test.go](/home/alberto/Trabajo/orquesta/cmd/mcp_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'Test(APIMCPDescribeEndpoint|APIMCPCallToolStateless|MCP)' -count=1`
- `go build -o ./orquesta .`
- validacion viva:
  - `./orquesta server stop && ./orquesta server start`
  - `curl -sf http://127.0.0.1:16543/api/mcp`
  - `curl -sf -X POST http://127.0.0.1:16543/api/mcp -H 'Content-Type: application/json' -d '{... \"method\":\"initialize\" ...}'`
- `curl -sf -X POST http://127.0.0.1:16543/api/mcp -H 'Content-Type: application/json' -d '{... \"method\":\"tools/list\" ...}'`

## 2026-03-31 — A2UI gana superficie server-first en la API de runtimes

Hallazgo:

- A2UI ya estaba contractual y renderizaba en `serve`, pero seguia siendo una proyeccion solo HTML
- faltaba una puerta canonica por API para que el mismo detalle llegase a clientes server-first sin duplicar parseo ni renderizado

Decision:

- exponer el detalle A2UI por `/api/runtimes/{id}/a2ui`
- reutilizar la proyeccion existente del panel y solo transformar la respuesta a JSON estable
- mantener `serve` como vista de lectura y no abrir aun edicion o respuesta humana por esta via

Codigo:

- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/api_test.go](/home/alberto/Trabajo/orquesta/cmd/api_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)
- [docs/op_094_ui_declarativa_agentes.md](/home/alberto/Trabajo/orquesta/docs/op_094_ui_declarativa_agentes.md)

Validacion:

- `go test ./cmd -run 'TestAPIRuntimeA2UIExponeMensajesDeFormaServerFirst' -count=1`
- `go test ./cmd -run 'TestWebRuntimeDetalleMuestraA2UIReadOnly' -count=1`
- smoke HTTP local pendiente de repetir con el binario recompilado si hace falta

## 2026-03-31 — las señales asíncronas del supervisor local dejan de tumbar el teardown

Hallazgo:

- la ultima bateria global de `go test ./cmd ./db ./planocontrol ./internal/controlruntime -count=1` saco un `panic` real en `db/sqlwrap.go`
- el origen no era SQL directo en el camino normal, sino un `SupervisorSignal` que llegaba tarde desde `internal/controlruntime/supervisor_local.go`
- cuando el handler de `cmd/controlruntime_hooks.go` intentaba hacer `ProcessTick`, la base temporal de pruebas ya podia estar cerrada y el handler terminaba reventando el goroutine

Decision:

- la emision de señales del supervisor local debe ser panic-safe
- si el plano de control ya se ha desmontado, la señal se degrada a error recuperable y no rompe ni el runtime local ni los tests

Codigo:

- [internal/controlruntime/supervisor_local.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/supervisor_local.go)
- [internal/controlruntime/supervisor_local_test.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/supervisor_local_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd ./db ./planocontrol ./internal/controlruntime -count=1` pendiente de repetir tras el fix

## 2026-03-31 — la supervision periodica ya no repica el mismo nudge sobre un supervisor vivo

Hallazgo:

- el nucleo ya no se quedaba atascado, pero seguia sembrando `nudge` + `send_instruction` de autonomia cada ~5 minutos sobre `Codex1`
- la evidencia salio por `/api/runtime-orders?agente=Codex1`: payload `accion=supervisar_proyecto`, `kind=autonomia`, repetido en serie aunque la mailbox ya drenaba a vacio
- la guardia de `supervisorAutonomiaYaOperativo()` no bastaba en estados transitorios del runtime y dejaba escapar repiques periodicos del mismo ciclo de supervision

Decision:

- la supervision autonoma periodica debe deduplicar por semantica reciente, no solo por “order pendiente” ni por “handle operativo”
- si ya existe un `nudge` reciente con `accion=supervisar_proyecto` para el mismo agente/proyecto dentro del intervalo de supervision, no se vuelve a sembrar

Codigo:

- [cmd/controlplane_autonomia_nivel2.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_autonomia_nivel2.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'TestProcesarSupervisionAutonomaBatch(NoRepiteSupervisionPeriodicaConSupervisorOperativo|NoRepiteSupervisionPeriodicaSiYaEmitioNudgeReciente)' -count=1`

## 2026-03-31 — `/api/runtime-handles` ya devuelve metadata viva del supervisor local

Hallazgo:

- la orquestacion ya estaba mas estable, pero la observabilidad de handles seguia siendo demasiado pasiva
- tras reiniciar el daemon, un `runtime_handle` activo observado por el supervisor local podia seguir devolviendo `supervisor_owner_pid` viejo hasta que otra ruta incidental lo resincronizara
- eso degradaba la verdad visible del sistema y hacia menos profesional la autoridad del daemon sobre runtimes vivos

Decision:

- cuando el listado de handles se pide filtrado por agente, el servicio debe resincronizar primero cada handle supervisado antes de devolverlo
- la observabilidad server-first debe ser verdad viva, no solo un dump de metadata persistida

Codigo:

- [runtimesapp/service.go](/home/alberto/Trabajo/orquesta/runtimesapp/service.go)
- [runtimesapp/service_test.go](/home/alberto/Trabajo/orquesta/runtimesapp/service_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./runtimesapp -count=1`
- `go test ./cmd -run 'TestAPIObservabilidadReadOnly|TestAPIRuntimeA2UIExponeMensajesDeFormaServerFirst' -count=1`
- `go build -o ./orquesta .`
- reinicio del daemon y comprobacion viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta server doctor`
  - `curl -sf 'http://127.0.0.1:16543/api/runtime-handles?agente=Codex1'`
  - el handle activo `#395` ya devuelve `supervisor_owner_pid=3840830`, que coincide con el daemon vivo actual

## 2026-03-31 — OpenClaw Gateway vuelve a ser visible para operador por API y dashboard

Hallazgo:

- el gateway OpenClaw ya existia como adaptador saliente real en `notificaciones/openclaw.go`
- pero el operador no tenia una lectura canonica de su estado operativo, y la unica pista quedaba dispersa en claves de config
- para cerrar el fleco sin crear otra fuente de verdad, el estado se expone ahora desde el daemon como snapshot de lectura

Decision:

- `OpenClaw Gateway` y Telegram se describen desde el servidor en `/api/notificaciones`
- el dashboard web del daemon muestra el mismo snapshot para operador
- no se ha tocado A2UI ni se ha creado una ruta paralela fuera del daemon

Codigo:

- [notificaciones/openclaw.go](/home/alberto/Trabajo/orquesta/notificaciones/openclaw.go)
- [notificaciones/openclaw_test.go](/home/alberto/Trabajo/orquesta/notificaciones/openclaw_test.go)
- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/serve.go](/home/alberto/Trabajo/orquesta/cmd/serve.go)
- [cmd/api_observabilidad_test.go](/home/alberto/Trabajo/orquesta/cmd/api_observabilidad_test.go)
- [cmd/serve_notificaciones_test.go](/home/alberto/Trabajo/orquesta/cmd/serve_notificaciones_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./notificaciones ./cmd -run 'Test(DescribirConfiguracionReflejaOpenClawYTelegram|APIObservabilidadReadOnly|WebDashMuestraEstadoOpenClawNotificaciones)' -count=1`
- smoke web/API pendiente de ejecutar tras el formateo y la recompilacion final

## 2026-03-31 — la supervisión periódica ya no repica sobre un supervisor que ya está trabajando

Hallazgo:

- `Codex1` seguía recibiendo ciclos `supervisar_proyecto` cada ~5 minutos aunque el mailbox drenara y la sesión/runtimes siguieran vivos
- `last_supervision_at` sí se actualizaba; el problema no era el reloj sino el criterio de re-siembra
- el batch periódico estaba tratando igual a un supervisor ocioso y a un supervisor que ya tenía tareas activas del proyecto

Decision:

- la supervisión periódica deja de sembrar `supervisar_proyecto` si el supervisor ya tiene trabajo activo real en ese proyecto
- el criterio estable no depende solo de heartbeat/handle; incorpora también la verdad funcional del backlog activo del supervisor
- esto endurece la orquestación contra repiques por estados transitorios del runtime

Codigo:

- [cmd/controlplane_autonomia_nivel2.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_autonomia_nivel2.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./cmd -run 'TestProcesarSupervisionAutonomaBatch(EncolaSupervisionYRegistraCiclo|ArrancaSupervisorPreferidoSinSesion|NoRepiteSupervisionPeriodicaConSupervisorOperativo|NoRepiteSupervisionPeriodicaSiYaEmitioNudgeReciente|NoRepiteSupervisionSiSupervisorYaTieneTrabajoActivo)' -count=1`
- `go build -o ./orquesta .`

## 2026-03-31 — el spinner del TUI ya no dispara falsos `runtime_panic`

Hallazgo:

- seguían apareciendo señales de `runtime_panic` falsas originadas por transcript con secuencias `OSC` de título de terminal del TUI de Codex
- ese ruido visual llegaba al clasificador como texto semántico y podía terminar enfriando agentes o generando nudges al supervisor sin fallo real

Decision:

- la limpieza de transcript ahora elimina secuencias `OSC` (`ESC ] ... BEL/ST`) antes de clasificar
- si tras esa limpieza una línea queda vacía, se descarta por completo y no genera transcript ni `runtime_events`
- el contrato correcto es: solo se escala por evidencia semántica real de fallo, no por ruido visual del cliente

Codigo:

- [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [db/runtime_transcript_test.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go test ./db -run 'Test(IngestarRuntimeTranscriptHandleNoClasificaLineaSistemaScript|IngestarRuntimeTranscriptHandleNoClasificaRuidoOSCSpinner|IngestarRuntimeTranscriptHandleClasificaYGeneraEventos)' -count=1`
- `go test ./cmd -run 'Test(ProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic|ProcesarRuntimeTranscriptBatchEnviaNudgeSupervisorPorSignal)' -count=1`
- `go build -o ./orquesta .`

## 2026-03-31 — `server start` espera publicación real del `statefile`

Hallazgo:

- había una carrera operativa en el arranque del daemon: `server start` podía considerar listo al servidor con `healthz` antes de que el `statefile` estuviera visible
- eso dejaba un estado incómodo donde el servidor ya estaba sano pero `doctor` podía responder `State: no disponible` en la comprobación inmediata

Decision:

- el handshake de `server start` ahora exige no solo `healthz`, sino también que `LoadServerInfo()` ya vea un `statefile` válido publicado por el daemon
- el contrato correcto de “daemon arrancado” incluye salud HTTP y descubrimiento local coherente

Codigo:

- [cmd/server.go](/home/alberto/Trabajo/orquesta/cmd/server.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `go build -o ./orquesta .`
- smoke vivo:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `ls -l /tmp/orquesta-localrpc-2ea48e3b1141.json`
  - `./orquesta server doctor`

## 2026-03-31 — cuota por ventana e identidad observada de cuenta

Hallazgo:

- `status` ya mostraba una cuota efectiva, pero seguía faltando el desglose visible por ventana con su `reset` propio; eso seguía ocultando si el cuello real era de `5h`, diario o semanal.
- además faltaba una identidad operativa útil por agente (`correo`/`usuario`) para saber qué cuenta real estaba usando cada runtime cuando hay varias credenciales o ventanas activas.

Decision:

- `db.Agente` pasa a exponer porcentaje y `reset_at` por `sesión`, `diario` y `semanal`, además de la `ventana efectiva`.
- la CLI de `status` renderiza ese desglose completo en una sola línea por agente.
- la identidad de cuenta se extrae solo de snapshots ya persistidos:
  - `presupuestos_sesion.raw_snapshot_json`
  - `runtime_handles.metadata_json`
- no se abre ninguna fuente paralela ni se “deduce” correo fuera de esos artefactos; si no hay dato observado, el campo queda vacío.

Codigo:

- [db/sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [db/presupuestos_sesion_test.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion_test.go)
- [cmd/status_remoto_compat.go](/home/alberto/Trabajo/orquesta/cmd/status_remoto_compat.go)
- [cmd/status_test.go](/home/alberto/Trabajo/orquesta/cmd/status_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'Test(ListarAgentesEnriquecePresupuestoVisible|ListarAgentesUsaPresupuestoSemanalSiEsMasRestrictivo|GetAgenteExtraeCuentaDesdeRuntimeHandle|EvaluarPresupuestoSesionHandoffPreventivo)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(RenderStatusSummaryMuestraCuotaAgente|RenderStatusSummaryMuestraBackendActivo)' -count=1`
- `go build -o ./orquesta .`
- smoke vivo:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `./orquesta server doctor`
  - `./orquesta status`
- resultado vivo:
  - `status` ya muestra `diario` y `semanal` con `reset` propio por agente
  - los agentes activos aún no exponen `correo` porque no hay identidad observada en sus snapshots/metadata actuales; el soporte queda listo para mostrarla cuando el runtime la persista

## 2026-03-31 — el dashboard ya enseña cuenta observada y ventanas de cuota

Hallazgo:

- la CLI `status` ya mostraba cuenta y desglose de cuota, pero el panel web seguía ciego a esos datos
- eso obligaba a usar terminal para ver qué cuenta estaba detrás de cada agente y qué ventana (`5h`, diaria o semanal) estaba mandando

Decision:

- el dashboard de `/` pasa a renderizar en la tarjeta de cada agente:
  - `cuenta`
  - `usuario`
  - `efectivo`
  - desglose `sesión / diario / semanal` con su `reset` propio
- el panel sigue sin inventar correo: solo pinta lo que Orquesta haya observado y persistido

Codigo:

- [cmd/serve.go](/home/alberto/Trabajo/orquesta/cmd/serve.go)
- [cmd/serve_notificaciones_test.go](/home/alberto/Trabajo/orquesta/cmd/serve_notificaciones_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(WebDashMuestraEstadoOpenClawNotificaciones|WebDashMuestraCuentaYVentanasDeCuotaAgente|RenderStatusSummaryMuestraCuotaAgente|RenderStatusSummaryMuestraBackendActivo)' -count=1`
- `go build -o ./orquesta .`
- resultado:
  - el dashboard ya pinta `cuenta` y `usuario` cuando existen en snapshots persistidos
  - el dashboard ya muestra `efectivo`, `diario` y `semanal` con su `reset` propio
  - la ventana `sesión` también aparece cuando el snapshot trae datos suficientes para calcular ratio (`window_started_at`, `reset_at`, `remaining_seconds`)

## 2026-03-31 — la vista de agentes y la API ya exponen cuenta y cuota

Hallazgo:

- el dashboard ya estaba alineado, pero la vista operativa `/agentes` y el detalle `/agentes/:nombre` seguían sin mostrar qué cuenta estaba detrás del agente ni el desglose completo de presupuesto
- además el contrato JSON de `/api/agentes` y `/api/status` lo exponía por serialización implícita, pero no estaba congelado con un test explícito

Decision:

- el panel de agentes pasa a mostrar `cuenta`, `usuario`, cuota `efectiva`, `diaria` y `semanal`
- el detalle de agente añade esos mismos datos en el resumen lateral
- se congela con test que `/api/agentes` y `/api/status` expongan cuenta y desglose de cuota cuando existan snapshots persistidos

Codigo:

- [cmd/agentes_web.go](/home/alberto/Trabajo/orquesta/cmd/agentes_web.go)
- [cmd/agentes_web_test.go](/home/alberto/Trabajo/orquesta/cmd/agentes_web_test.go)
- [cmd/api_test.go](/home/alberto/Trabajo/orquesta/cmd/api_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(WebAgentesPanelMuestraEstadoVivo|WebAgenteDetalleMuestraControlPlaneYDetalleOperativo|APIAgentesYStatusExponenCuentaYCuotaVisible|WebDashMuestraCuentaYVentanasDeCuotaAgente)' -count=1`
- `go build -o ./orquesta .`

## 2026-03-31 — endpoints dedicados de presupuesto y cuentas de agentes

Hallazgo:

- `/api/status` y `/api/agentes` ya exponían cuenta y cuota, pero eran payloads demasiado amplios para un consumidor que solo quiera telemetría operativa
- hacía falta una entrada API pequeña para:
  - presupuestos por agente
  - nombre de agente + cuenta observada

Decision:

- se añade `GET /api/agentes/presupuesto`
  - devuelve solo telemetría de cuota por agente
  - acepta `?activos=true` para filtrar la flota viva
- se añade `GET /api/agentes/cuentas`
  - devuelve `nombre`, `rol`, `activo`, `habilitado`, `cuenta_email`, `cuenta_usuario`, `cuenta_fuente`
  - acepta también `?activos=true`
- ambos endpoints salen del mismo modelo enriquecido de `db.Agente`; no abren ninguna fuente de verdad nueva

Codigo:

- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/cliente_servidor.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor.go)
- [cmd/api_test.go](/home/alberto/Trabajo/orquesta/cmd/api_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIAgentesPresupuestoYCuentas|APIAgentesYStatusExponenCuentaYCuotaVisible)' -count=1`
- `go build -o ./orquesta .`

## 2026-03-31 — CLI server-first para cuentas y presupuesto de agentes

Hallazgo:

- con los endpoints nuevos ya se podía consultar la telemetría por API, pero seguía faltando una vía operativa en CLI para el uso diario del orquestador sin tirar de `curl`

Decision:

- se añaden dos comandos server-first:
  - `orquesta agente cuentas [--activos] [--json]`
  - `orquesta agente presupuesto [--activos] [--json]`
- ambos reutilizan los endpoints dedicados y no abren ninguna ruta local paralela

Codigo:

- [cmd/agente_telemetria.go](/home/alberto/Trabajo/orquesta/cmd/agente_telemetria.go)
- [cmd/agente_telemetria_test.go](/home/alberto/Trabajo/orquesta/cmd/agente_telemetria_test.go)
- [cmd/cliente_servidor_recursos.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor_recursos.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(AgenteCuentasCmdRenderizaListado|AgentePresupuestoCmdRenderizaListado|APIAgentesPresupuestoYCuentas)' -count=1`
- `go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server start`
  - `./orquesta agente cuentas --activos`
  - `./orquesta agente presupuesto --activos`
- resultado vivo:
  - `agente presupuesto` ya devuelve la cuota efectiva y el desglose diario/semanal de `Codex1-5`
  - `agente cuentas --activos` hoy devuelve `—` en todos los agentes activos porque sus snapshots/metadata vivas todavía no persisten correo observado

## 2026-03-31 — Ranking de cuentas y separacion entre telemetria viva y observada

Hallazgo:

- `codex --help` no expone un comando externo claro tipo `usage` o `whoami`, pero los perfiles de Codex CLI si dejan artefactos utiles fuera de la TUI:
  - `auth.json` con identidad/cuenta
  - eventos `token_count` en `sessions/*.jsonl` con ventanas primaria `300m` y secundaria `10080m`, `used_percent` y `resets_at`
- esos artefactos son valiosos, pero no equivalen siempre a dato vivo al segundo; si se mezclan sin marcar frescura, el orquestador puede tomar malas decisiones

Decision:

- se anade `GET /api/agentes/ranking-cuentas` y `orquesta agente ranking-cuentas [--activos] [--json]`
- el ranking agrupa por cuenta observada y ordena por la mejor telemetria disponible: `remaining_tokens`, `remaining_credits`, `remaining_messages`, `remaining_seconds` y, si no existe nada mejor, `cuota_pct`
- se fija en la doctrina que Orquesta debe distinguir entre telemetria `viva` y `observada`; la segunda sirve como respaldo/enriquecimiento, no como mentira de tiempo real

Codigo:

- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/cliente_servidor.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor.go)
- [cmd/cliente_servidor_recursos.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor_recursos.go)
- [cmd/agente_telemetria.go](/home/alberto/Trabajo/orquesta/cmd/agente_telemetria.go)
- [cmd/api_test.go](/home/alberto/Trabajo/orquesta/cmd/api_test.go)
- [cmd/agente_telemetria_test.go](/home/alberto/Trabajo/orquesta/cmd/agente_telemetria_test.go)
- [cmd/cliente_servidor_test.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(APIAgentesPresupuestoYCuentas|AgenteRankingCuentasCmdRenderizaListado|CommandSupportsServerMode|AgenteCuentasCmdRenderizaListado|AgentePresupuestoCmdRenderizaListado)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `curl -sS http://127.0.0.1:16543/api/agentes/ranking-cuentas?activos=true`
  - `./orquesta agente ranking-cuentas --activos`
- resultado vivo:
  - `Codex2` queda primero con `89%`
  - `Codex1` segundo con `77%`
  - `Codex5` tercero con `74%`

## 2026-03-31 — Ingesta canonica de `auth.json` y `token_count` de Codex CLI

Hallazgo:

- la telemetria de cuota de Codex no sale por un comando externo fiable, pero si queda persistida en artefactos locales del propio perfil:
  - `auth.json` con identidad/correo
  - `sessions/*.jsonl` con eventos `token_count`
- hasta ahora Orquesta solo reflejaba bien el bloqueo cuando el proveedor respondia `usage limit`; faltaba la via observada regular para ventanas `5h` y semanal

Decision:

- el daemon ingesta esa telemetria observada dentro del ciclo del control plane
- la ventana primaria `300m` pasa a proyectarse como `5h`
- la secundaria `10080m` pasa a alimentar el presupuesto semanal observado
- la cuenta observada sale de `auth.json` / claims del token y queda persistida con `cuenta_fuente=codex_token_count_observed`

Codigo:

- [internal/controlruntime/codex_observe.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe.go)
- [internal/controlruntime/codex_observe_test.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe_test.go)
- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [db/presupuestos_sesion.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion.go)
- [db/sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [db/presupuestos_sesion_test.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime ./db ./cmd -run 'Test(ObserveCodexArtifactsLeeTokenCountYAuth|ListarAgentesUsaVentanasObservadasDesdeTokenCount)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `./orquesta sesion presupuesto ver --agente Codex1`
  - `./orquesta agente presupuesto --activos --json`
  - `./orquesta agente cuentas --activos`
  - `./orquesta agente ranking-cuentas --activos`
- resultado vivo:
  - `Codex1` ya muestra `Window kind: 5h`, `Ratio restante: 0.89`
  - `agente cuentas --activos` ya devuelve correos reales observados
  - el ranking por cuenta ya agrupa por email real y deja a `maritere@avidad.com` con `Codex1,Codex5`

## 2026-03-31 — La cuota observada stale deja de mandar sobre la efectiva

Hallazgo:

- tras meter la ingesta de `token_count`, los snapshots observados de primera hora seguian pudiendo dominar la cuota efectiva muchas horas despues
- eso era peligroso: para inspeccion sirve, pero para orquestacion profesional una foto vieja no puede gobernar decisiones vivas

Decision:

- si `presupuestos_sesion` es observada y supera `pool_budget_snapshot_max_age_seconds`, Orquesta:
  - mantiene visibles `sesión` y `semanal` observadas
  - marca `telemetría observada stale`
  - deja que la `cuota efectiva` vuelva al derivado seguro

Codigo:

- [db/presupuestos_sesion.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion.go)
- [db/sesiones.go](/home/alberto/Trabajo/orquesta/db/sesiones.go)
- [db/handoff_manager.go](/home/alberto/Trabajo/orquesta/db/handoff_manager.go)
- [cmd/status_remoto_compat.go](/home/alberto/Trabajo/orquesta/cmd/status_remoto_compat.go)
- [db/presupuestos_sesion_test.go](/home/alberto/Trabajo/orquesta/db/presupuestos_sesion_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db ./cmd -run 'Test(GetAgenteNoDejaQueSnapshotObservadoStaleMandeSobreLaCuotaEfectiva|ListarAgentesUsaVentanasObservadasDesdeTokenCount)' -count=1`
- smoke viva:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `./orquesta status`

Resultado vivo:

- `Codex1` ya enseña `telemetría observada stale`
- su `sesión 89%` y `semanal 3%` siguen visibles para inspeccion
- la `cuota efectiva` vuelve a `77% daily`, que es la decision segura mientras la observacion siga vieja

## 2026-03-31 — Fallback al snapshot mas fresco del perfil Codex

Hallazgo:

- la ingesta observada seguia atada en exceso a la sesion exacta del handle
- eso dejaba `stale` innecesario cuando el mismo perfil de Codex ya habia emitido un `token_count` mas fresco en otra sesion del mismo home

Decision:

- `ObserveCodexArtifacts(...)` primero intenta la sesion exacta
- si existe otro `token_count` mas fresco dentro del mismo perfil/home de Codex, Orquesta usa ese snapshot y lo marca internamente como `profile`
- no inventa datos ni consulta nada fuera del propio home de Codex; solo elige la observacion canonica mas fresca disponible

Codigo:

- [internal/controlruntime/codex_observe.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe.go)
- [internal/controlruntime/codex_observe_test.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(ObserveCodexArtifactsLeeTokenCountYAuth|ObserveCodexArtifactsUsaSnapshotMasFrescoDelPerfil)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `./orquesta status`

Resultado vivo:

- la vista ya muestra edades observadas mas razonables donde el perfil tenia un snapshot mas fresco
- el ranking sigue siendo coherente y seguro, sin volver a hacer pasar por vivo lo que no lo es

## 2026-03-31 — Fallback al snapshot mas fresco de la cuenta OAuth compartida

Hallazgo:

- varios perfiles (`Codex1`, `Codex5`) comparten la misma cuenta OAuth
- limitarse al perfil deja `stale` evitable cuando otro perfil de la misma cuenta ya emitio un `token_count` mas reciente

Decision:

- el observador de Codex ya no cae solo a `session` y `profile`
- si el mismo correo aparece en otro home de `codex-perfiles`, Orquesta puede usar el `token_count` mas fresco de esa misma cuenta y marcarlo internamente como `account`
- sigue siendo la misma fuente canonica observada (`auth.json` + `sessions/*.jsonl`), no una heuristica inventada

Codigo:

- [internal/controlruntime/codex_observe.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe.go)
- [internal/controlruntime/codex_observe_test.go](/home/alberto/Trabajo/orquesta/internal/controlruntime/codex_observe_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./internal/controlruntime -run 'Test(ObserveCodexArtifactsLeeTokenCountYAuth|ObserveCodexArtifactsUsaSnapshotMasFrescoDelPerfil|ObserveCodexArtifactsUsaSnapshotMasFrescoDeLaCuenta)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop`
  - `./orquesta server start`
  - `./orquesta status`

Resultado vivo:

- `Codex1/Codex5` ya se pueden beneficiar del snapshot mas fresco de la misma cuenta observada
- la cuota efectiva sigue siendo segura; el cambio solo reduce ceguera operativa dentro de la misma identidad OAuth

## 2026-03-31 — Retry de descubrimiento server-first tras restart

Hallazgo:

- tras `server stop && server start`, algunos subcomandos server-first podian caer en una ventana corta donde no redescubrian el daemon aunque este ya estuviese sano
- `status` entraba porque su camino ya tenia una segunda oportunidad, pero `agente ranking-cuentas` podia fallar justo despues del restart

Decision:

- `apiGet(...)` y `apiPost(...)` ya hacen un segundo intento de descubrimiento del servidor si el primer `serverBaseURL()` sale vacio o si la primera conexion cae durante la ventana de reinicio
- no se abre fallback local nuevo; sigue siendo server-first, solo con redescubrimiento mas robusto

Codigo:

- [cmd/cliente_servidor.go](/home/alberto/Trabajo/orquesta/cmd/cliente_servidor.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(CommandSupportsServerMode|APIAgentesPresupuestoYCuentas|AgenteRankingCuentasCmdRenderizaListado)' -count=1`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta agente ranking-cuentas --activos`

Resultado vivo:

- `agente ranking-cuentas` ya responde correctamente inmediatamente tras el restart del daemon

## 2026-03-31 — Higiene automatica de deuda historica de runtime

Hallazgo:

- el nucleo ya estaba estable, pero seguia acumulando deuda historica terminal en `runtime_handles` y `runtime_orders`
- esa basura no bloqueaba el flujo vivo, pero contaminaba diagnostico, recuperacion y lectura operativa

Decision:

- el runner del control plane incorpora un batch `runtime_hygiene`
- la purga automatica solo toca deuda terminal vieja:
  - `runtime_handles` en `cerrado/fallido`
  - `runtime_orders` en `completada/fallida/expirada/cancelada`
- las ventanas de retencion quedan configurables con:
  - `runtime_handles_retention_hours`
  - `runtime_orders_retention_hours`
- la purga valida antes de borrar y desvincula referencias (`runtime_orders.handle_id`, `runtime_transcript.handle_id`, `runtime_mailbox.runtime_order_id`) para no romper integridad

Codigo:

- [db/config_defaults.go](/home/alberto/Trabajo/orquesta/db/config_defaults.go)
- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [planocontrol/runner.go](/home/alberto/Trabajo/orquesta/planocontrol/runner.go)
- [db/controlplane_entities_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities_test.go)
- [planocontrol/runner_test.go](/home/alberto/Trabajo/orquesta/planocontrol/runner_test.go)
- [planocontrol/runner_e2e_test.go](/home/alberto/Trabajo/orquesta/planocontrol/runner_e2e_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db ./planocontrol ./cmd -run 'TestPurgarRuntimeHistoricoSoloBorraDeudaViejaTerminal|TestRunner|TestRunnerE2E' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go test ./... -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
- `./orquesta server doctor`
- `./orquesta runtime diagnostico --agente Codex1 --limit 5`

Resultado vivo:

- el daemon sigue sano y el handle activo de `Codex1` permanece intacto
- la higiene queda integrada en el runner oficial, sin scripts ni purgas manuales para la deuda terminal vieja

## 2026-03-31 — La higiene automatica tambien purga runtimes terminales viejos

Hallazgo:

- tras integrar la higiene automatica de `runtime_handles` y `runtime_orders`, seguian acumulandose muchas `runtime_instances` cerradas/degradadas/fallidas
- ese historico no bloqueaba el flujo vivo, pero hacia mas ruidoso `runtime diagnostico` y retrasaba el cierre del nucleo

Decision:

- extender `PurgarRuntimeHistorico()` para incluir tambien `runtime_instances` terminales viejas
- mantener la misma disciplina: solo estados terminales, solo por cutoff configurable y con validacion previa para no tocar nada vivo
- nueva retencion configurable: `runtime_instances_retention_hours`

Codigo:

- [db/config_defaults.go](/home/alberto/Trabajo/orquesta/db/config_defaults.go)
- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [db/controlplane_entities_test.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db ./planocontrol ./cmd -run 'TestPurgarRuntimeHistoricoSoloBorraDeudaViejaTerminal|TestRunner|TestRunnerE2E' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go test ./... -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta server doctor`
  - `./orquesta runtime diagnostico --agente Codex1`

Resultado vivo:

- la purga automatica ya cubre deuda terminal vieja de `runtime_instances`, `runtime_handles` y `runtime_orders`
- el daemon sigue sano y los handles vivos no se tocan

## 2026-03-31 — Mailbox de refresh no debe quedar viva en enfriamiento

Hallazgo:

- `Codex1` seguia con `runtime_mailbox` pendiente `governance_refresh` aunque estaba correctamente en `estado_cuota=enfriamiento` por `usage limit`
- eso no era trabajo entregable real; era ruido operativo que hacia parecer que quedaba deuda viva cuando el agente no debia despertarse todavia

Decision:

- extender la reconciliacion de mailbox para consumir tambien `governance_refresh` y `skills_refresh` cuando el agente esta en enfriamiento
- mantener la misma doctrina que con `watchdog`: durante el cooldown esos mensajes ya no deben quedar pendientes; el contexto canonico se regenerara en el siguiente `start/resume`

Codigo:

- [cmd/controlplane_support.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support.go)
- [cmd/controlplane_support_test.go](/home/alberto/Trabajo/orquesta/cmd/controlplane_support_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestProcesarRuntimeMailboxBatch(ConsumeWatchdogEnEnfriamiento|ConsumeGovernanceRefreshEnEnfriamiento)' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta server doctor`
  - `./orquesta runtime mailbox --to Codex1 --estado pendiente`

Resultado vivo:

- `Codex1` ya no deja `governance_refresh` pendiente durante el cooldown
- `./orquesta runtime mailbox --to Codex1 --estado pendiente` vuelve a vacio
- el daemon sigue sano con `Health RPC: OK`

## 2026-03-31 — El transcript PTY no debe tragar fragmentos de control

Hallazgo:

- en la flota viva seguian apareciendo en transcript entradas recientes de `pty_out` con restos tipo `BEL/BS` y fragmentos de un solo caracter (`r`, `"`, etc.)
- eso no rompia la entrega, pero ensuciaba diagnostico y abria la puerta a tomar ruido de repintado como actividad real del runtime

Decision:

- endurecer `limpiarLineaTranscript()` para eliminar caracteres de control residuales del PTY antes de clasificar o registrar la linea
- mantener la doctrina de que solo el texto semantico cuenta como output util; fragmentos de repintado no deben entrar en transcript ni derivar eventos

Codigo:

- [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [db/runtime_transcript_test.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestIngestarRuntimeTranscriptHandle(NoClasificaRuidoOSCSpinner|DescartaFragmentosPTYConControlChars|FiltraRuidoYClasificaPanic|RecortaPreambuloScriptEnRuntimePanic)' -count=1`

Resultado vivo:

- el transcript futuro ya no debe registrar restos `BEL/BS` ni repintados de un solo caracter como output util
- la observabilidad del runtime queda mas limpia y el control plane depende menos de ruido del TUI

## 2026-03-31 — `runtime ordenes` ya soporta limite real

Hallazgo:

- `./orquesta runtime ordenes --agente Codex3` seguia volcando toda la historia y dificultaba inspeccionar el estado reciente del control plane
- ya existia `--limit` en transcript y checkpoints, pero no en runtime orders, asi que el diagnostico reciente era innecesariamente ruidoso

Decision:

- añadir `limit` de punta a punta en runtime orders: filtro DB, API server-first, CLI y recuperacion local
- mantener el orden `ORDER BY id DESC`, pero permitir recortar en origen para que la salida operativa sirva de verdad

Codigo:

- [db/controlplane_entities.go](/home/alberto/Trabajo/orquesta/db/controlplane_entities.go)
- [cmd/api.go](/home/alberto/Trabajo/orquesta/cmd/api.go)
- [cmd/runtime.go](/home/alberto/Trabajo/orquesta/cmd/runtime.go)
- [cmd/runtime_test.go](/home/alberto/Trabajo/orquesta/cmd/runtime_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd ./db -run 'Test(RuntimeControlPlaneUsaAPICuandoHayServidor|RuntimeOrdenesPermiteRecuperacionConForceLocal)$' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta server doctor`
  - `./orquesta runtime ordenes --agente Codex3 --limit 5`

Resultado vivo:

- `runtime ordenes` ya devuelve solo el tramo reciente solicitado
- la inspeccion viva del control plane vuelve a ser util sin tener que purgar historico para leer lo ultimo

## 2026-03-31 — El statefile no vale si no pasa healthz

Hallazgo:

- `server start` podia devolver exito y, acto seguido, `server doctor` todavia ver un `statefile` viejo con `Health RPC: KO`
- el problema no era el daemon nuevo, sino que `loadServerInfoWithHealthFallback()` confiaba demasiado pronto en el `statefile` si existia, sin verificar que ese estado seguia vivo

Decision:

- endurecer `loadServerInfoWithHealthFallback()` para validar por `healthz` el `statefile` cargado
- si el `statefile` existe pero ya no responde o no cuadra con el scope/storage esperados, Orquesta cae al `healthz` real y recupera el estado vivo

Codigo:

- [cmd/server.go](/home/alberto/Trabajo/orquesta/cmd/server.go)
- [cmd/server_health_fallback_test.go](/home/alberto/Trabajo/orquesta/cmd/server_health_fallback_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'TestLoadServerInfoWithHealthFallback(UsaHealthzSiFaltaStatefile|RecuperaStatefileStale|RechazaScopeAjeno|RechazaDBAjena|RechazaDriverAjeno)$' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start && ./orquesta server doctor`

Resultado vivo:

- `server start` seguido inmediatamente de `server doctor` ya devuelve el PID/addr reales del daemon nuevo
- el arranque oficial deja de mentir por un `statefile` stale

## 2026-03-31 — `transcript_log_pending` ya no arrastra basura del PTY

Hallazgo:

- los handles activos seguian guardando en `metadata_json` un `transcript_log_pending` enorme con secuencias ANSI/control del PTY
- eso no aportaba continuidad real y convertia la metadata viva en un sumidero de ruido

Decision:

- compactar `transcript_log_pending` antes de persistirlo
- limpiar OSC/ANSI/control chars, normalizar espacios, truncar a un tamaño razonable y eliminarlo por completo si no queda contenido semantico

Codigo:

- [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [db/runtime_transcript_test.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestIngestarRuntimeTranscriptHandle(CompactaPendingTranscript|DescartaFragmentosPTYConControlChars|NoClasificaRuidoOSCSpinner)$' -count=1`

Resultado:

- la metadata futura del handle ya no debe arrastrar pendientes crudos del TUI
- baja el ruido persistido y se hace mas fiable la inspeccion de handles vivos

## 2026-03-31 — El transcript ya descarta el banner de arranque de Codex

Hallazgo:

- incluso con el saneado de control chars, el transcript reciente de agentes activos seguia mostrando lineas de banner como `Perfil activo`, `CODEX_HOME`, `Credenciales`, consejo de login o la URL de releases
- eso seguia ensuciando la lectura operativa: parecia salida del agente cuando en realidad era solo banner del launcher/TUI

Decision:

- descartar explicitamente ese banner de arranque de Codex durante la ingestión de transcript
- mantener en transcript solo sistema relevante y salida semantica del agente, no ruido del bootstrap del cliente

Codigo:

- [db/runtime_transcript.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript.go)
- [db/runtime_transcript_test.go](/home/alberto/Trabajo/orquesta/db/runtime_transcript_test.go)
- [docs/BIBLIA_APP_ORQUESTA.md](/home/alberto/Trabajo/orquesta/docs/BIBLIA_APP_ORQUESTA.md)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestIngestarRuntimeTranscriptHandle(DescartaBannerCodex|CompactaPendingTranscript|DescartaFragmentosPTYConControlChars|NoClasificaRuidoOSCSpinner)$' -count=1`

Resultado:

- el transcript futuro ya no debe llenarse con banner de Codex al arrancar o reanudar
- sube la calidad de la observabilidad y reduce falsas lecturas de “actividad” en agentes vivos

Nota 2026-03-31 22: se amplía el filtro del banner para incluir también las URLs de uso/licencia de ChatGPT/Codex (`/codex/settings/usage`, `/explore/pro`), que seguían entrando como ruido operativo.

## 2026-03-31 — `runtime diagnostico` ya prioriza lo vivo y reciente

Hallazgo:

- `runtime diagnostico --limit 5` seguia mostrando bloques enormes de runtimes cerrados viejos, porque el limite solo afectaba a órdenes/checkpoints
- eso hacia poco usable el diagnostico operativo justo cuando mas se necesitaba

Decision:

- mantener el resumen global completo (`Runtimes: 69`, `Órdenes: 4554`, etc.)
- pero recortar las secciones detalladas de `Runtimes` y `Handles` a lo reciente/relevante:
  - primero estados no terminales (`activo`, `degradado`, `esperando_*`, etc.)
  - luego solo el tramo terminal mas reciente si sobra hueco

Codigo:

- [cmd/runtime_diagnostico.go](/home/alberto/Trabajo/orquesta/cmd/runtime_diagnostico.go)
- [cmd/runtime_diagnostico_test.go](/home/alberto/Trabajo/orquesta/cmd/runtime_diagnostico_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(RuntimeDiagnosticoUsaAPI|RuntimeDiagnosticoRequiereServidor|RuntimeDiagnosticoRecortaRuntimesYHandlesSegunLimit)$' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta server stop && ./orquesta server start`
  - `./orquesta server doctor`
  - `./orquesta runtime diagnostico --agente Codex2 --limit 5`

Resultado:

- el diagnostico ya enseña primero los runtimes/handles vivos o degradados y deja fuera la mayor parte de la arqueologia
- sigue conservando el conteo total para no perder contexto

## 2026-03-31 — `runtime transcript` oculta ruido operativo por defecto

Hallazgo:

- aunque el motor ya no reingesta banner nuevo, el transcript historico seguia mostrando banners viejos del launcher/TUI y hacia poco util la CLI diaria
- la inspeccion normal necesitaba ver señal de trabajo; el ruido antiguo solo hacia falta en modo forense

Decision:

- `runtime transcript` pasa a filtrar por defecto el ruido operativo/banners ya conocidos
- se añade `--raw` para recuperar la vista completa sin filtros cuando haga falta analisis forense

Codigo:

- [cmd/runtime.go](/home/alberto/Trabajo/orquesta/cmd/runtime.go)
- [cmd/runtime_test.go](/home/alberto/Trabajo/orquesta/cmd/runtime_test.go)

Validacion:

- `env GOCACHE=/tmp/orquesta-gocache go test ./cmd -run 'Test(ImprimirRuntimeTranscriptOcultaRuidoOperativoPorDefecto|ImprimirRuntimeTranscriptRawMantieneRuidoOperativo|RuntimeControlPlaneUsaAPICuandoHayServidor)$' -count=1`
- `env GOCACHE=/tmp/orquesta-gocache go build -o ./orquesta .`
- smoke viva:
  - `./orquesta runtime transcript --agente Codex5 --limit 8`
  - `./orquesta runtime transcript --agente Codex5 --limit 8 --raw`

Resultado:

- la vista normal ya enseña solo señal util
- `--raw` mantiene el acceso a todo el transcript historico cuando se necesita
