# Decisiones: orquesta-runtime-codex

## RTCODEX-DEC-010

```text
Fecha: 2026-05-14
Decision: Con sandbox `workspace-write`, `runtime_work_dir` debe estar fuera de
`project_work_dir`; el wrapper lo autoriza con `--add-dir`.
Motivo: una prueba real OPES con varios agentes mostro contaminacion de
contexto: al vivir `.orquesta-runtime` dentro del proyecto, un agente podia leer
`agent_packet.json` y `codex_stderr.log` de otros agentes. Eso rompe el
aislamiento, aumenta contexto y reproduce el patron de v1/v2 de bucles por
estado oculto compartido.
Evidencia: smoke real local del 2026-05-14 con `codex exec`, `workspace-write`
y `--add-dir runtime_work_dir` externo escribio correctamente artifact del
proyecto y `agent_ack.json` en runtime externo.
Alternativas:
  - Mantener runtime dentro del proyecto: descartado por contaminacion entre
    agentes paralelos.
  - Ejecutar OPES en secuencial: valido como fallback operativo, descartado como
    solucion principal porque Orquesta debe paralelizar sin mezclar contexto.
  - Usar `danger-full-access`: descartado como default porque amplia permisos.
Impacto: el conector rechaza runtime igual, interno o ancestro del proyecto. El
server usa por defecto `.orquesta-control/<proyecto>/runtime` como sibling
externo al proyecto. Cada agente sigue teniendo runtime propio por run/agente.
El prompt explicita que el ACK externo se escribe desde runtime por shell, no
por `apply_patch` de proyecto.
Estado: aceptada; sustituye RTCODEX-DEC-008.
```

## RTCODEX-DEC-009

```text
Fecha: 2026-05-13
Decision: El cierre seguro de Codex usa un protocolo cooperativo por ficheros
de control en `runtime_work_dir`.
Motivo: `codex exec` arranca con prompt de entrada y no ofrece un canal stdin
interactivo fiable para decirle a cualquier agente vivo que pare y guarde
checkpoint. Fingir esa capacidad repetiria el problema de v1/v2: esperar o
parchear sin contrato real.
Alternativas:
  - Mandar senales al proceso y asumir continuidad: descartado porque no genera
    evidencia durable de checkpoint.
  - Usar rutas o detalles de proceso dentro del core: descartado por romper
    hexagonal y filtrar runtime.
  - Esperar a que no haya agentes vivos: conservador, pero insuficiente para
    cierre no forzado de trabajos largos.
Impacto: cada prompt incluye `orquesta_shutdown_request.json`; si aparece, el
agente debe escribir `agent_shutdown_checkpoint_ack.json` con schema
`codex_shutdown_checkpoint_ack.v0`. El stack solo podra registrar checkpoint
cuando ese ACK exista, correle y no filtre detalles prohibidos.
Estado: aceptada.
```

## RTCODEX-DEC-008

```text
Fecha: 2026-05-09
Decision: Con sandbox `workspace-write`, `runtime_work_dir` debe vivir dentro de `project_work_dir`, normalmente bajo `.orquesta-codex-runtime/<run>/<agent>`.
Motivo: En prueba real de equipo de programacion, los agentes pudieron crear codigo dentro de `project_work_dir`, pero no pudieron escribir `agent_ack.json` cuando `runtime_work_dir` era un sibling externo. Anadir `--add-dir` al wrapper se mantuvo como defensa extra, pero no fue suficiente con el protocolo real de escritura de Codex.
Alternativas:
  - Usar siempre `danger-full-access`: descartado para smokes/producto porque aumenta permisos sin necesidad.
  - Mantener runtime externo con `--add-dir`: descartado como solucion principal porque la prueba real siguio sin poder escribir el ACK.
  - Mezclar ACK con artifacts del usuario: descartado; el runtime queda en directorio oculto de control y fuera del write-set de producto.
Impacto: El runtime sigue fuera del core y del request publico, pero queda dentro del arbol escribible real del agente para evitar esperas infinitas por entregas ya hechas sin receipt.
Estado: sustituida por RTCODEX-DEC-010.
```

## RTCODEX-DEC-007

```text
Fecha: 2026-05-11
Decision: El ACK acepta artifacts concretos que satisfacen un write-set con
glob cerrado.
Motivo: en smoke real multiagente el director creo una microtarea con
`internal/bootstrap/*.go`. El agente genero archivos concretos validos dentro
del patron, pero el validador exigia el literal `*.go` como artifact y fallo.
Alternativas: prohibir globs en write-set del director; pedir al agente que
liste patrones en vez de archivos; relajar el write-set cerrado.
Impacto: `codexAckPathMatchesWriteSetEntryV0` permite patrones cerrados como
`*.go` y `/**` para validar tanto artifact requerido como pertenencia al
write-set. Los paths siguen siendo relativos y no se permite salida fuera del
patron.
Estado: aceptada.
```

## RTCODEX-DEC-007

```text
Fecha: 2026-05-09
Decision: Validar ProcessRuntimeLaunchRequestV0 antes de materializar packet, prompt y wrapper Codex.
Motivo: La prueba real mostro que un directorio de agente con wrapper preparado puede confundirse con un agente arrancado. El conector no debe escribir artefactos si el request privado de proceso no pasara la validacion del runtime.
Alternativas:
  - Seguir materializando y dejar que LaunchV0 bloquee despues: descartado; empeora el diagnostico y crea restos ambiguos.
  - Validar con reglas locales Codex: descartado; duplicaria las reglas de proceso y podria divergir.
Impacto: `CodexExecResolverV0` construye primero el request privado, llama a `ValidateProcessRuntimeLaunchRequestV0` y solo despues escribe ficheros de control. Si falla, devuelve issue sanitizado sin paths ni secretos.
Estado: aceptada.
```

## RTCODEX-DEC-006

```text
Fecha: 2026-05-09
Decision: La plantilla de ACK debe incluir evidencia esperada, no arrays vacios.
Motivo: en prueba real de programacion, un agente creo codigo y test pero omitio `go test ./...` en el ACK porque la plantilla mostraba `tests: []`. La validacion hizo bien en rechazarlo.
Alternativas:
  - Relajar validacion de tests obligatorios: descartado porque aceptaria entregas sin evidencia.
  - Mantener plantilla vacia y confiar en instrucciones generales: descartado porque ya produjo un fallo real.
Impacto: `BuildCodexAgentPromptV0` serializa `files` con el write-set y `tests` con los tests obligatorios. El ACK sigue validandose estrictamente y el core no recibe detalles operacionales.
Estado: aceptada.
```

## RTCODEX-DEC-005

```text
Fecha: 2026-05-09
Decision: Convertir ACK Codex validado a observacion neutral antes de entrar al nucleo.
Motivo: `RegisterDelivery` pertenece al workflow, pero el ACK real puede contener nombres, paths o detalles de proveedor que el core no debe conocer.
Alternativas:
  - Pasar el ACK completo al nucleo: descartado por filtrar runtime/proveedor/artifacts.
  - Registrar DeliveryRegistered directamente desde el conector: descartado por saltarse el workflow.
Impacto: `CodexDeliveryObservationV0` solo expone delivery_ref, phase_id, task_id, agent_ref, summary y evidence_refs compactas; si alguna ref contiene detalles prohibidos para el core, se rechaza como ACK no aceptable.
Estado: aceptada.
```

## RTCODEX-DEC-001

```text
Fecha: 2026-05-07
Decision: Implementar Codex como conector externo opt-in, no como dependencia del nucleo ni de orquesta-runtime.
Motivo: la agenda real no debe depender de procesos hijos controlados, pero tampoco puede filtrar Codex, HOME, OAuth, modelo ni paths reales al core.
Alternativas:
  - Llamar codex directamente desde core: descartado por acoplamiento.
  - Ampliar ExternalAgentLaunchSpec con command_path/env reales: descartado por filtracion operacional.
  - Mantener solo proceso controlado: descartado para la prueba no controlada solicitada.
Impacto: el resolver concreto vive en mini-proyecto separado y puede sustituirse por Ollama, vLLM, API remota u otro agente.
Estado: aceptada.
```

## RTCODEX-DEC-004

```text
Fecha: 2026-05-07
Decision: El prompt Codex opt-in exige protocolo compacto por defecto.
Motivo: v1/v2 ya habian identificado que workers narrativos queman cuota y contexto; Codex real debe arrancar con salida minima salvo necesidad explicita.
Impacto: cada launch Codex pide activar $caveman o compact si existe, salida minima y evidencia corta; el ACK estructurado sigue siendo la evidencia valida.
Estado: aceptada.
```

## RTCODEX-DEC-002

```text
Fecha: 2026-05-07
Decision: Usar wrapper operacional generado en runtime workdir.
Motivo: ProcessRuntimeConnectorV0 debe seguir cerrado y no recibir HOME/PATH/OAuth/modelo; el wrapper permite ejecutar el comando real sin exponer esos detalles al contrato publico.
Impacto: el request publico de proceso solo contiene wrapper y project workdir. Los ficheros de control, logs y ACK siguen en runtime workdir para permitir varios agentes sobre el mismo proyecto.
Estado: aceptada.
```

## RTCODEX-DEC-003

```text
Fecha: 2026-05-07
Decision: Validar el ACK de Codex como receipt operacional del conector, no como contrato de core.
Motivo: un agente externo no controlado puede emitir un ACK exitoso incompleto, no correlado o con evidencia que filtre HOME/secreto/transcript; el conector debe convertirlo en error publico antes de aceptarlo.
Alternativas:
  - Aceptar cualquier ACK JSON con request_id: descartado por evidencia insuficiente.
  - Mover campos de proveedor/HOME/OAuth al core para validar: descartado por romper frontera hexagonal.
Impacto: el ACK exige correlacion opaca con request_id, correlation_id y ack_ref, cierra artifacts al write-set y rechaza detalles operacionales prohibidos sin devolverlos como evidencia.
Estado: aceptada.
```

## RTCODEX-DEC-006

```text
Fecha: 2026-05-10
Decision: `decision_path` es un archivo de control obligatorio solo cuando el paquete del agente lo exige.
Motivo: el prompt base lo exponia como opcional y podia contradecir tareas de director que necesitan emitir decisiones ejecutables.
Impacto: el conector mantiene `director_decisions.json` fuera del write-set, pero aclara que debe escribirse si objetivo o criterios de cierre lo piden.
Estado: aceptada.
```

## RTCODEX-DEC-007

```text
Fecha: 2026-05-13
Decision: Un ACK `completed` con contexto requerido truncado debe justificar
materializacion externa.
Motivo: OPES puede necesitar temas amplios, pero Orquesta mantiene contextos
pequenos. Si un campo obligatorio se corta y el agente completa sin declararlo,
aceptariamos contenido inventado o parcial.
Impacto: el prompt avisa cuando `agent_packet.context.entries` tiene
`required=true` y `truncated=true`; el validador rechaza `completed` salvo que
`notes` incluya `contexto_truncado_resuelto: ...`. Si no puede resolverlo, el
agente debe devolver `failed` con `CONSULTA AL DIRECTOR`.
Estado: aceptada.
```
