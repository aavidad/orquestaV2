# Tarea Orquesta/OPES: validadores con 0 ficheros y ruido de stream en external-work

Fecha: 2026-06-26.

Contexto: ola OPES de Integración Social B en Orquesta aislado `19022`, sin tocar
el núcleo de Orquesta activo en otra sesión.

## Incidencia 1: validador de texto público puede devolver `pass` con 0 ficheros

En temas 011 y 023 se observó que
`validate_public_text_no_author_notes.py` puede producir informe `pass` con
`files_scanned=0` cuando se invoca sobre subdirectorios de Markdown o rutas de
tema no previstas por el descubridor. Ejemplo observado:

- curso: `auxiliar-tecnico-superior-integracion-social-b-dipgra-2026`;
- tema: `023`;
- salida: `temas/tema_023/09_validacion/informe_texto_publico_sin_notas_autor.*`;
- resultado: `status=pass`, `files_scanned=0`, `finding_count=0`;
- mitigación manual: búsqueda focal con `rg` sobre los Markdown públicos.

Problema: un `pass` con 0 ficheros no debe servir como validación real. Orquesta
debe tratarlo como `invalid_validation_empty_scan` y pedir reintento con raíz de
curso o lista explícita de ficheros publicables.

Aceptación esperada:

- Si `files_scanned=0`, la validación queda `fail` o `invalid`, nunca `pass`.
- El ACK del agente debe distinguir "validador oficial pasó" de "comprobación
  focal manual pasó".
- El supervisor debe pedir rework o reintento cuando una validación obligatoria
  devuelve 0 ficheros revisados.

## Incidencia 2: ruido repetido `Failed to create stream fd`

En varios agentes OPES external-work aparece repetidamente:

```text
Failed to create stream fd: Operation not permitted
```

Se observó en subroles y padres de la ola de Integración Social B. No siempre
es fatal: algunos agentes escribieron artefactos, ACK y `codex_last_message.txt`
correctamente. Aun así, ensucia logs, puede confundirse con fallo real y debería
clasificarse.

Aceptación esperada:

- Orquesta debe clasificar este mensaje como aviso conocido si el proceso sale
  bien y hay ACK/evidencia.
- Si coincide con ausencia de ACK o salida incompleta, debe elevarse como causa
  de bloqueo técnico.
- La vista de estado/API debería mostrar `warning_stream_fd` sin ocultar el
  estado real del trabajo.

## Incidencia 3: búsquedas demasiado amplias en OPES external-work

En el tema 023 se observó búsqueda inicial demasiado amplia que alcanzó backups
y otros cursos antes de enfocar el tema. No bloqueó la entrega, pero consume
tiempo y contexto.

Aceptación esperada:

- Los prompts/plantillas OPES para external-work deberían incluir exclusiones
  por defecto para `opes-salidas/backups/**`, paquetes históricos y carpetas de
  runtime, salvo que el objetivo sea auditoría global.
- La búsqueda normal debe empezar por curso, tema, programa, canones y material
  reutilizable relevante.

## Incidencia 4: subroles reales materializados después del ACK del padre

En la misma ola, el padre del tema 023 escribió `agent_ack.json` a las
`2026-06-26 01:49:45`, pero los seis subroles reales aparecieron después, con
`agent_packet.json` y prompts a partir de `2026-06-26 01:50:11`.

El patrón se repitió en la ola `19022b`: el padre de T025 escribió ACK a las
`2026-06-26 02:39`, pero a las `02:41` seguían vivos seis procesos de subrol
`subrole-fuentes`, `subrole-reutilizacion`, `subrole-redaccion`,
`subrole-visuales`, `subrole-tests-tutor` y `subrole-html-rag-audio-qa` para el
mismo `task_ref`, nacidos después del cierre del padre.

Esto es mejor que no materializar subagentes, pero rompe el cierre causal si
Orquesta interpreta el ACK del padre como fin de trabajo antes de que existan
ACK, bloqueo o informe de cada hijo.

Aceptación esperada:

- El padre no debe quedar terminal si hay `child_task_refs` pendientes de
  materializar o cerrar.
- La vista/API debe distinguir `parent_ack_received` de `cohort_closed`.
- Si Orquesta decide lanzar subroles después del ACK del padre, debe convertirlo
  en fase explícita de revisión/post-assessment, no en cierre normal del padre.
- Los informes finales de OPES deben poder saber si los seis subroles son
  subagentes reales, subroles internos del padre o revisiones tardías.

## Incidencia 5: cadena recursiva de assessments post-ACK en T023

En el run aislado `run-opes-integracion-social-b-t023-creacion-textual-20260626-19022`
se observó una cadena de assessments después de que el padre y los seis subroles
ya hubieran escrito ACK. La primera oleada de assessments fue útil porque generó
revalidaciones y reportes focales, pero tras cerrar S5 tests/tutor con ACK a las
`2026-06-26 02:13:34` Orquesta lanzó otro assessment S5
`agent-ref-assessment-task-ref-app-change-appchange-f8398562d44a0ae56a829ad564bc478f-s-0875928c2cedb8775ff735eac8b49bb2`
para el mismo subrol, mismo tema y mismo write-set.

Evidencia observada:

- padre T023 con ACK a `2026-06-26 01:49:45`;
- seis subroles reales cerrados entre `01:54:00` y `02:00:59`;
- múltiples assessments útiles cerrados entre `01:57:48` y `02:13:34`;
- nuevo assessment S5 nacido después del ACK S5 de `02:13:34`;
- el tema ya tenía texto ampliado válido para nivel B y estado editorial
  correcto `pendiente_assets_html_tests_rag_audio_qa`, no `ready`.

Problema: si cada assessment puede disparar otro assessment equivalente sin
criterio de cierre por `subrole + topic + artifact_version`, Orquesta consume
cuota y mantiene el run vivo aunque el producto ya esté en el estado pendiente
correcto.

Aceptación esperada:

- Definir llave de idempotencia de assessment por `run_ref + topic_id +
  subrole + artifact_version + assessment_kind`.
- No lanzar un nuevo assessment del mismo subrol si ya existe ACK posterior a la
  última modificación del write-set evaluado.
- La vista/API debe mostrar `cohort_closed_with_pending_product_work` cuando la
  revisión ya concluyó que faltan assets/HTML/tests/RAG/audio/QA externos, en
  vez de seguir abriendo assessments del mismo tipo.
- Si se alcanza una segunda revisión equivalente sin cambios de producto,
  convertirlo en tarea técnica `assessment_recursion_guard_needed`, no en nuevo
  agente de contenido.

## Incidencia 6: supervisión global elige runs antiguas frente a ola activa

En la ola OPES aislada `19022b` se lanzaron correctamente los temas `025`,
`027`, `028`, `029`, `030` y `031` de Integración Social B. A las
`2026-06-26 02:31` el API `/api/v0/autoprogramming/status` mostraba la cola con
la ola nueva y varias runs antiguas `011`, `012`, `013`, `022`, `023` y `024`
como `running`, aunque esas runs ya habían sido revisadas o drenadas por el
director OPES.

Al ejecutar una supervisión global recomendada por el propio estado:

```text
POST http://127.0.0.1:19022/api/v0/autoprogramming/supervise
```

Orquesta devolvió:

```text
run_ref=run-opes-integracion-social-b-t023-creacion-textual-20260626-19022
stop_reason=max_ticks
evidence_refs=[..., quiescent, error, wait_external]
```

Problema: el supervisor global atendió una run antigua de T023 en vez de una run
de la ola activa nueva. Esto impide usar la recomendación `supervise:queue` como
señal fiable de avance y puede reabrir trabajo ya controlado.

Aceptación esperada:

- La selección global debe priorizar runs `ready/running` con proceso vivo,
  ACK pendiente o actividad reciente real.
- Las runs sin proceso vivo y ya drenadas deben pasar a estado terminal,
  `stale_terminal`, `needs_manual_reconcile` o equivalente, pero no competir
  con la ola nueva.
- El API debe exponer una lista separada de `stale_running` con causa y acción
  segura, sin que `supervise` global las elija por defecto.
- Cuando el director pida supervisar una ola concreta, debe poder pasar
  `run_ref` o prefijo de run y evitar que Orquesta salte a runs antiguas.

## Incidencia 7: ACK de subrol escrito en el `agent_ack.json` del padre

En la ola `19022b`, run
`run-opes-integracion-social-b-t029-creacion-textual-20260626-19022b`, se
observó que el fichero de control del padre:

```text
.../agent-ref-task-ref-app-change-appchange-cfa8690009c02f8429b649c9a54b3ba8/agent_ack.json
```

contenía un ACK con `task_ref` terminado en `-subrole-tests-tutor` y solo dos
ficheros de tests/tutor:

```text
temas/tema_029/plan_tests_tutor_tema.md
temas/tema_029/subroles/tests_tutor/ack_tests_tutor.md
```

Sin embargo, el proceso del padre seguía vivo y el log mostraba que aún estaba
esperando el subrol S3 redacción:

```text
S5 tests/tutor completado ... Solo falta S3 redacción
```

Problema: si un subagente o subrol escribe el `agent_ack.json` reservado al
padre, Orquesta puede interpretar un cierre causal falso aunque el padre no haya
integrado el tema ni emitido el estado final. En este caso T029 no tenía
`borrador_ampliado.md` ni mínimo textual, por lo que no podía cerrarse como
creación textual.

Aceptación esperada:

- Cada subrol debe escribir su ACK en ruta propia, no en el ACK de control del
  padre.
- El ACK del padre debe poder escribirse solo por el proceso padre al final de
  la integración.
- Si aparece `agent_ack.json` con `task_ref` de subrol en directorio de padre,
  Orquesta debe marcar `invalid_parent_ack_subrole_collision` y no cerrar la
  run.
- La vista/API debe mostrar `parent_running_with_child_ack_collision` hasta que
  el padre emita ACK final válido o se haga rework.

## Incidencia 8: subagente escribe duplicados fuera del write-set del curso

En la ola `19022b`, run
`run-opes-integracion-social-b-t031-creacion-textual-20260626-19022b`, el propio
padre del tema 031 informó de duplicados creados fuera del write-set del curso
por error de ruta. Se observaron ficheros bajo la raíz global de OPES, no bajo
el curso de Integración Social:

```text
/home/alberto/Trabajo/OPES/temas/tema_031/00_fuentes_y_bases/fuentes_tema_031.md
/home/alberto/Trabajo/OPES/temas/tema_031/05_html_rag_audio_qa/handoff_html_rag_audio_qa_tema_031.md
/home/alberto/Trabajo/OPES/00_control/orquesta_runs/run-opes-integracion-social-b-t031-creacion-textual-20260626-19022b/subroles/fuentes/agent_ack.md
/home/alberto/Trabajo/OPES/00_control/orquesta_runs/run-opes-integracion-social-b-t031-creacion-textual-20260626-19022b/subroles/html_rag_audio_qa/agent_ack.md
```

Problema: un subagente puede escribir en rutas relativas a `/home/alberto/Trabajo/OPES`
en vez de la raíz del curso asignado. Eso contamina el repositorio, dificulta la
limpieza y puede hacer que validadores o inventarios encuentren artefactos fuera
del producto real.

Aceptación esperada:

- Cada task/subtask debe recibir una raíz de curso normalizada y validada.
- Orquesta debe rechazar escrituras fuera del write-set declarado, salvo rutas
  explícitas de control autorizadas.
- La vista/API debe marcar `write_set_escape_detected` con lista de rutas.
- El agente debe poder entregar un informe de incidencia, pero no debe crear
  duplicados fuera de la raíz de curso.

## Incidencia 9: registro OPES queda en `en_progreso` tras ACK y sin proceso vivo

Tras cerrar la ola `19022b`, el registro OPES seguía mostrando como
`en_progreso` temas antiguos de Integración Social B (`015`, `016`, `017`,
`018`, `019`, `020` y `026`). En `ps` no había procesos vivos asociados y varios
temas tenían ACK de Orquesta o informes completos. Los textos ampliados ya
superaban el mínimo B:

```text
T015 10888 palabras
T016 10836 palabras
T017 10942 palabras
T018 10881 palabras
T019 12067 palabras
T020 12188 palabras
T026 11104 palabras
```

Problema: la vista humana y el `status` del registro no distinguían trabajo vivo
de locks antiguos no liberados. Esto llevó a considerar que había agentes
activos cuando realmente solo quedaban entregas pendientes de reconciliar.

Aceptación esperada:

- Orquesta/OPES bridge debe liberar o reconciliar el lock cuando el padre emite
  ACK terminal y no hay procesos vivos.
- El registro debe exponer `stale_lock_no_process` o `needs_reconcile`, no
  `en_progreso`, si no hay proceso vivo y existe ACK/informe terminal.
- La vista/API debe mostrar fecha de último proceso vivo, ACK terminal y acción
  recomendada (`release`, `rework` o `relaunch`).
- Un tema con texto mínimo cumplido pero sin derivados finales debe quedar en
  estado `texto_minimo_B_ok_pendiente_assets_html_tests_rag_audio_qa`, no
  bloqueado como si siguiera en ejecución.

## Incidencia 10: `usage limit` deja runs como `running` sin proceso, ACK ni producto

En la ola aislada `19022c` se lanzaron los temas `032`, `033`, `034`, `035`,
`036` y `037` de Integración Social B mediante `external-work/run` y
supervisión dirigida por `run_ref`. Orquesta aceptó las solicitudes, materializó
`agent_packet.json` y `/api/v0/runs/supervise` devolvió `running` con evidencia
`projection-tasks-7` y `requested-agents-1`.

Sin embargo, los procesos Codex no llegaron a producir artefactos ni ACK. En
los seis `codex_stderr.log` apareció:

```text
ERROR: You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 4:58 AM.
```

Comprobación posterior:

- no había procesos vivos para `run-opes-integracion-social-b-t032...t037`;
- no existía `agent_ack.json` en las seis runs;
- los directorios `temas/tema_032` a `temas/tema_037` solo contenían
  `README.md` y `topic_manifest.json`;
- el registro OPES seguía mostrando los seis temas como `en_progreso`.

Problema: una denegación estructurada del proveedor por cuota no debe mantener
la run como trabajo vivo ni dejar bloqueado el tema. Debe convertirse en estado
reintentable con hora/causa, sin perder el paquete ni crear la falsa impresión
de agentes activos.

Aceptación esperada:

- Detectar `usage limit`/cuota en `stderr` como causa terminal reintentable del
  intento actual.
- Marcar la run como `provider_usage_limit_retry_after` o equivalente, no
  `running`.
- Exponer en API/vista que no hay ACK, no hay producto y no hay proceso vivo.
- Reconciliar el registro OPES como `pendiente_reintento_orquesta_usage_limit`
  con hora de reintento si está disponible.
- Al recuperar cuota, permitir relanzar de forma idempotente la misma ola sin
  duplicar directorios ni pisar entregas previas.

## Incidencia 11: `runs/supervise` devuelve `runtime_error` aunque hay procesos vivos

Tras arrancar una Orquesta nueva aislada `19023` con el código actualizado, se
relanzó la ola T032-T037 de Integración Social B. La cola inicial salió limpia:
seis runs `ready`, sin arrastre antiguo y `queue_health.queued=6`.

Al supervisar de forma dirigida T032:

```text
POST /api/v0/runs/supervise
run_ref=run-opes-integracion-social-b-t032-creacion-textual-20260626-19023c
```

la respuesta fue `estado=error`, `stop_reason=runtime_error` y
`last.status=failed`, pero la propia evidencia incluía:

```text
evidence-ref-codex-supervisor-drain-projection-requested-agents-7
evidence-ref-codex-supervisor-live-process-check
evidence-ref-codex-supervisor-process-live
```

La comprobación externa con `ps` confirmó que existían siete procesos reales
para T032: padre y seis subroles. Por tanto, el estado público del endpoint no
describía el trabajo real. Además, una supervisión posterior de T033 agotó
timeout sin cuerpo, pero sí dejó arrancado al menos el proceso padre.

Diagnóstico mostrado por la respuesta de T032:

```text
status=wait_unhandled_outbox ... error=transicion_invalida: idempotency_key
```

Problema: un conflicto de idempotencia/transición del supervisor no debe
convertir en `failed` una run que ya tiene procesos vivos y trabajando. El
estado correcto debería distinguir `supervisor_transition_error_but_agents_live`
de fallo real de contenido/proveedor.

Aceptación esperada:

- Si hay procesos vivos asociados al `run_ref`, `/api/v0/runs/supervise` no
  debe devolver `last.status=failed` sin marcar explícitamente que los agentes
  siguen vivos.
- El error `transicion_invalida: idempotency_key` debe reconciliarse como
  incidencia del supervisor/outbox, no como fallo terminal del run de OPES.
- La respuesta debe incluir acción segura: `wait_agents`, `retry_supervise`,
  `reconcile_outbox` o `manual_reconcile`, según corresponda.
- Una llamada que agota timeout después de arrancar agentes debe devolver o
  persistir un diagnóstico recuperable; no debe dejar al director sin saber si
  se inició trabajo real.

## Incidencia 12: `autoprogramming/status` marca `running_stale` con procesos vivos

En la misma ola `19023`, después de arrancar T032 y T033, el endpoint
`/api/v0/autoprogramming/status` informó:

```text
queue_health.running_stale=2
```

Sin embargo, `ps` mostraba procesos vivos para T032 y T033 en el runtime
`19023_runtime`. En T032 había siete procesos asociados al run y en T033 al
menos el padre.

Problema: `running_stale` es útil para detectar restos muertos, pero si se
activa mientras hay procesos vivos introduce una señal falsa y puede llevar a
reconciliar o relanzar trabajo que está ejecutándose.

Aceptación esperada:

- Calcular `running_stale` comprobando liveness real de procesos/rutas de
  runtime antes de clasificar.
- Separar `running_without_recent_stats` de `running_stale_no_process`.
- Mostrar conteo `agents_live` o evidencia equivalente para que el director sepa
  si debe esperar o intervenir.

## Incidencia 13: el `agent_packet` estrecha el write-set y bloquea consolidación de producto

En la ola `19023`, T032 sí produjo trabajo útil: seis subroles reales y un
borrador ampliado de redacción con 10.851 palabras, por encima del mínimo B de
10.800. Sin embargo, el informe del padre dejó constancia de una contradicción
entre el contrato externo y el paquete activo:

```text
La solicitud externa original permitía `temas/tema_032`, pero el paquete activo
gana. Por eso no se crea texto publicable, HTML, tests, visuales, RAG ni audio
en esta unidad.
```

Evidencia:

- solicitud `external-work/run`: `allowed_write_set` incluía `temas/tema_032`;
- `agent_packet` del padre limitó la escritura efectiva a coordinación;
- el subrol S3 escribió `temas/tema_032/subroles/redaccion/borrador_ampliado.md`
  con 10.851 palabras;
- el padre no pudo consolidar `temas/tema_032/borrador_ampliado.md` ni cerrar el
  tema como texto canónico revisable.

Problema: para temarios OPES, el padre debe poder consolidar los artefactos de
producto dentro del directorio del tema. Si el paquete activo gana con un
write-set más estrecho que la solicitud externa, Orquesta genera trabajo útil
pero lo deja sin integración canónica.

Aceptación esperada:

- El `agent_packet` del padre debe heredar el write-set de producto autorizado
  por `external-work/run`, al menos `temas/tema_NNN` y su coordinación.
- Los subroles pueden tener write-set estrecho, pero el padre integrador debe
  poder consolidar Markdown canónico, informes de extensión y checkpoint del
  tema.
- Si Orquesta estrecha el write-set por seguridad, debe crear una fase
  explícita `integration_required` y lanzar un agente integrador con write-set
  suficiente, no cerrar con producto disperso.
- La API/vista debe marcar `product_not_consolidated_due_write_set_narrowing`,
  no `completed`, cuando hay borrador útil en subrol pero falta artefacto
  canónico del tema.

## No hacer

No arreglar esto con parches ad hoc en la sesión OPES. Debe corregirse en la
app/flujo de Orquesta u OPES bridge para que no vuelva a ocurrir en nuevas olas.

## Mitigación Orquesta 2026-06-26 - cola `running` stale y límite de uso

Se mitigó en `modulos/orquesta-app-codex-stack` la parte de Orquesta que hacía
que una run OPES antigua en `running` siguiera compitiendo con olas nuevas:

- `reconcileQueuedRunningStaleCandidateV0` ya no exige `ProcessRegistry` con al
  menos un registro para reconciliar trabajo OPES `running` sin proceso vivo
  verificable.
- Si los logs/runtime del agente contienen límite de cuota de Codex, la run se
  terminaliza en `RunControl`/`RunQueue` como `stopped`, pero con causa
  accionable `provider_usage_limit_retry_after`.
- La cola conserva evidencias específicas:
  `evidence-ref-provider-usage-limit-retry-after` y
  `evidence-ref-codex-usage-quota-exhausted`/`limited`.
- Se añadió cobertura:
  `TestCodexStackV0RunGlobalTickReconciliaOPESUsageLimitSinProcesoRegistradoV0`.

Validación ejecutada:

- `git diff --check`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 -p=1 ./...`

Queda pendiente: exponer una lista pública separada `stale_running`/acciones en
`autoprogramming/status` y web, reconciliar el registro OPES `en_progreso` desde
el bridge de dominio, y cerrar las incidencias independientes de ACK de subrol
en ruta de padre, assessments recursivos y validaciones con `files_scanned=0`.

## Incidencia 14: `/api/v0/autoprogramming/supervise` despacha agentes pero deja el cliente HTTP colgado

En la ola OPES aislada `19023d` para Integración Social B, temas T038-T043,
la cola quedó correctamente en `ready` y `autoprogramming/status` recomendó
`supervise:queue`. Al llamar:

```text
POST http://127.0.0.1:19023/api/v0/autoprogramming/supervise
body {}
```

la llamada no devolvió respuesta en más de 90 segundos. Se cortó solo el cliente
HTTP local (`curl`), sin matar procesos de Orquesta. Aun así, el despacho sí se
había producido: `ps` mostró procesos Codex vivos de la ola `19023d`, incluyendo
T038 con padre y seis subroles reales, y T039 iniciando padre/subroles.

Problema: una API de supervisión que ejecuta la acción pero no responde deja al
director sin señal fiable. Además puede parecer un fallo total aunque el trabajo
siga vivo.

Aceptación esperada:

- `autoprogramming/supervise` debe responder con `202 accepted`, `200 ok` o
  estado parcial aun cuando los agentes sigan vivos.
- La respuesta debe incluir runs despachados, runs pendientes, procesos vivos
  detectados y siguiente acción segura.
- Si la supervisión sigue en segundo plano, debe devolver un `operation_ref`
  consultable en vez de mantener el HTTP bloqueado.
- Si hay timeout interno, no debe duplicar despachos ya materializados al
  reintentar.

## Mitigación Orquesta 2026-06-28 - timeout HTTP en observación goal-first

Seguimiento relacionado, sin marcar cerrada toda la incidencia 14: las rutas
`/api/v0/autoprogramming/supervise`, `/api/v0/runs/supervise`,
`/api/v0/autoprogramming/status` y `/api/v0/external-work/run` ya tenían ventana
HTTP acotada o aceptación en segundo plano en el código actual. El hueco análogo
que seguía directo en el flujo goal-first era
`/api/v0/apps/director/goal/observe`.

Se añadió timeout público en `modulos/orquesta-mcp` para esa ruta:

- si el executor no responde dentro de la ventana HTTP acotada, devuelve
  `504 Gateway Timeout` con JSON público;
- el error usa `observe_app_director_goal_timeout`, `field=executor` y conserva
  `run_ref`/`X-Correlation-ID`;
- el contexto del executor se cancela para no dejar la llamada HTTP retenida;
- la respuesta se flushea igual que en otros endpoints MCP acotados.

Cobertura añadida:

- `TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutDevuelveJSONPublico`
- `TestServerObserveAppDirectorGoalHTTPClienteRealRecibeTimeoutJSONV0`

Validación ejecutada:

- `git diff --check`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPObserveAppDirectorGoalHTTPHandlerV0(Delega|Timeout|RunRef|Error)|TestMCPObserveAppDirectorGoalErrorResultFromErrorV0|TestMCPObserveAppDirectorGoalToolExecutorV0RunRef'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerObserveAppDirectorGoalHTTPClienteRealRecibeTimeoutJSONV0|TestServerExternalWorkRunHTTPClienteRealRecibeTimeoutJSONV0'`
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`
- `go test -count=1 ./...`

Queda pendiente de esta familia: comprobar con una ola OPES temporal nueva que
el operador ya no recibe llamadas sin cuerpo en las rutas de supervisión y
estado bajo carga real, y cerrar las incidencias independientes de
`running_stale` con procesos vivos y write-set estrechado del `agent_packet`.
