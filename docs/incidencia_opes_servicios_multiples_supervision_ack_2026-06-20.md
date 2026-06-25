# Incidencia OPES Servicios Multiples: supervision y ACK de artefactos

Fecha: 2026-06-20.

Contexto: prueba real OPES para el curso C2 `oficial-de-servicios-multiples`
en `/home/alberto/Trabajo/OPES`, con OPES API local y Orquesta residente.

## Sintomas observados

- El drenaje inicial de 17 `draft_content_block` vio todos los trabajos, pero
  solo envio 11 antes de devolver fallo por `effect_timeout` y
  `external_bridge_claim_failed`.
- Varios `opes-drain-once` devolvieron `supervision_status=retry_pending` o
  `resident_director_pending` con `supervision_unavailable`; hubo que observar
  o relanzar reworks concretos desde el operador.
- Algunos padres escribieron contenido util y prueba local, pero el ACK
  persistido no incluia `artifact_type`, `complete_job` ni `idempotency_key`.
  En los temas 10 y 16 OPES respondio HTTP 400 al submit y creo trabajos de
  rework causales.
- Un timeout de supervision no implica ausencia de trabajo: se crearon runs y
  agentes despues del retorno del comando. El operador no puede asumir que un
  timeout equivalga a no enviado.
- En la fase `generate_visual_asset`, Orquesta genero visuales utiles en disco,
  pero varios jobs seguian `pending` en OPES. El replay por
  `POST /api/jobs/{id}/artifacts` fallo con `FOREIGN KEY constraint failed`
  porque el job visual solo tenia refs de programa (`program_topic:00x` o
  `placement_ref`) y OPES intento materializar el visual como `content_block`
  contra `canonical_topics/topic_chapters`, que aun estaban vacios para este
  curso. Esto no invalida el visual como insumo, pero no puede marcarse como
  bloque visual final hasta que exista tema/capitulo canonico o el bridge lo
  registre como artefacto trazable no materializable.
- En la integracion de bancos de test, `POST /api/v0/runs/supervise` devolvio
  `estado=error`, `stop_reason=runtime_error` y evidencias
  `external-wait-exhausted` / `operational-director-plan-state:blocked`, pero
  el agente real seguia vivo, escribio artefactos validos y termino entregando
  ACK. El operador, al ver el job OPES `pending` y el artefacto local ya
  validado, hizo recuperacion manual por `POST /api/jobs/{id}/artifacts`; unos
  segundos despues Orquesta registro tambien el ACK tardio, dejando dos
  artefactos validos para el mismo job con distintas idempotency keys. La
  supervision no debe emitir estado terminal de error si hay proceso vivo o
  entrega recuperable pendiente de submit/reconciliacion.
- En la fase `generate_audio_asset`, el job OPES
  `e0590279d369f06ac6945ea602e4f960` fue lanzado como external-work real, pero
  el agente Codex quedo dentro de sandbox con `network: restricted` y bloqueo
  `edge-tts` en una prueba minima durante mas de 90 segundos. La misma prueba
  `edge-tts --voice es-ES-AlvaroNeural --text "Prueba breve de audio OPES."`
  funciono desde el entorno principal en menos de un segundo. Orquesta debe
  clasificar trabajos que necesitan red externa controlada, como TTS Edge, y
  lanzarlos con perfil opt-in adecuado o delegarlos a un runner autorizado; no
  debe dejar al agente colgado ni sustituir automaticamente por un motor local
  no canonico.
- El control publico `POST /api/v0/runs/control` acepto `action=stop` y dejo
  `status=stop_requested`, pero no materializo `orquesta_shutdown_request.json`
  en el workdir del agente. El agente siguio tras el corte del smoke y empezo a
  preparar un artefacto alternativo con `espeak-ng`. Hubo que terminar el arbol
  de la run con `SIGTERM` para evitar que un audio no canonico entrara en el
  paquete OPES. El stop debe llegar al agente y a sus procesos hijo, y el
  supervisor debe confirmar la parada sin inspeccion manual.
- La documentacion OPES vigente apunta a `scripts/opes_audio_app.py`. El
  2026-06-24 se corrigio la instalacion local: el script canonico existe y
  `scripts/tcae_audio_app.py` queda como wrapper historico. Orquesta no debe
  depender de una ruta supuesta: debe validar el tool path efectivo antes de
  crear trabajos de audio y exponer un bloqueo accionable si la herramienta
  canonica no esta instalada.

## Evidencia local

- Curso OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/oficial-de-servicios-multiples/`
- Programa OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/oficial-de-servicios-multiples/02_fuentes_oficiales/programa/programa_orquesta_2026-06-20.txt`
- Drenaje inicial:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/servicios_multiples_2026-06-20/runtime/opes_drain_draft_parents.out.json`
- Reworks causales:
  `7664ddc95bd59db118ea30465e978b75` para tema 10 y
  `ab92feaae204e63246e2f8179dfc9ba9` para tema 16.
- Integracion tardia de tests:
  run `run-external-work-opes-701302ba15d5732273615f051845d67c-opes-job-701302ba15d5732273615f051845d67c`,
  job OPES `701302ba15d5732273615f051845d67c`, artefacto manual
  `b8c211471e8247215b9e4263c8c049b7` y artefacto posterior de Orquesta con
  idempotency `idem-domain-work-artifact-opes-701302ba15d5732273615f051845d67c-local_html_site`.
- Audio Edge bloqueado por sandbox:
  run `run-external-work-opes-e0590279d369f06ac6945ea602e4f960-opes-job-e0590279d369f06ac6945ea602e4f960`,
  job OPES `e0590279d369f06ac6945ea602e4f960`; `runs/supervise` devolvio
  `runtime_error`, el proceso seguia vivo, `edge-tts` quedo bloqueado dentro de
  `network: restricted`, `POST /api/v0/runs/control` devolvio
  `stop_requested`, y el operador termino los procesos
  `3027159/3027161/3027173` con `SIGTERM` tras comprobar que el agente seguia
  sin checkpoint de apagado efectivo.
- Smoke comparativo fuera de sandbox:
  `/tmp/opes_edge_smoke_main_e0590279.mp3` se genero correctamente con
  `edge-tts` desde el entorno principal.
- Run aceptada sin despacho residente:
  job OPES `b305970086cb77343865513f32df7a66` (`plan_temario`) se envio con
  `opes-drain-once` filtrado por `JOB_REF` exacto y creo la run
  `run-external-work-opes-b305970086cb77343865513f32df7a66-opes-job-b305970086cb77343865513f32df7a66`.
  El bridge devolvio `submitted` y `resident_director_pending`, pero tras mas
  de 30 segundos el job seguia `pending`, no habia proceso Codex nuevo y el
  operador tuvo que despertar manualmente la supervision. Tarea creada:
  `SRV-TASK-024`.
- Reconciliacion de ACK valida bloqueada por conflicto:
  despues del despertar manual, un agente de Orquesta escribio
  `external/opes/plan_temario/b305970086cb77343865513f32df7a66/artifact_document_plan.json`,
  dejo ACK completado y paso
  `opes-domain-test-plan_temario-b305970086cb77343865513f32df7a66`. El retry
  posterior tambien dejo ACK y prueba pasada. Sin embargo, una supervision
  dirigida para ingerir el ACK devolvio `runtime_error` con diagnostico
  `domain_work_submit_conflict`; la proyeccion indicaba `open_tasks=0` y
  `requested_agents=2`, mientras el job OPES seguia `pending` y
  `/api/jobs/b305970086cb77343865513f32df7a66/artifacts` devolvia `null`.
  Tarea creada: `SRV-TASK-025`.
- Supervision OPES terminada en falso por quiescencia:
  el job OPES `b8702608bde2d7ebc6ca18f53d65495e` (`validate_topic`) se envio
  con `opes-drain-once` filtrado por `JOB_REF` exacto y creo la run
  `run-external-work-opes-b8702608bde2d7ebc6ca18f53d65495e-opes-job-b8702608bde2d7ebc6ca18f53d65495e`.
  El bridge devolvio `submitted` y `resident_director_pending`. Una llamada
  manual posterior a `POST /api/v0/runs/supervise` devolvio `estado=ok`,
  `stop_reason=done`, `ticks=1`, `quiescent`,
  `projection-tasks-0`, `projection-open-tasks-0` y
  `projection-requested-agents-0`; sin embargo, el job OPES siguio
  `pending`, `attempts=0`, `last_error=""` y sin artefacto de validacion. La
  supervision no debe cerrar como `done/quiescent` una run externa cuyo job de
  dominio no ha avanzado. Tarea creada: `SRV-TASK-026`.
- El mismo patron se repitio con el job OPES
  `28c7d0694eeb31f8fb65e5b54cefa9fa`
  (`generate_help_manual_assets`): `opes-drain-once` devolvio `submitted` y
  `resident_director_pending`; `POST /api/v0/runs/supervise` devolvio
  `estado=ok`, `stop_reason=done`, `ticks=1`, `quiescent`,
  `projection-tasks-0`, `projection-open-tasks-0` y
  `projection-requested-agents-0`; el job siguio `pending`, `attempts=0`,
  `last_error=""` y sin artefacto de manual. Este segundo caso confirma que no
  es especifico de `validate_topic`, sino del puente/supervisor OPES cuando la
  run queda sin tareas internas.
- Tras registrar manualmente en OPES un `help_manual_package` validado para ese
  mismo job, aparecio tarde un agente real de la run
  `run-external-work-opes-28c7d0694eeb31f8fb65e5b54cefa9fa-opes-job-28c7d0694eeb31f8fb65e5b54cefa9fa`.
  Empezo a copiar capturas a `raw/` y anuncio que iba a reescribir el manual con
  nombres canonicos. Como el job de dominio ya estaba `completed`, el operador
  pidio `POST /api/v0/runs/control action=stop` con idempotency key
  `stop-help-manual-late-agent-28c7-20260621`; Orquesta devolvio
  `stop_requested` y `checkpoint_recorded=true`, y el proceso termino. El
  puente debe reconciliar agentes tardios contra el estado real del job antes de
  permitir escrituras que dupliquen artefactos ya aceptados.

## Tarea tecnica

Orquesta debe cerrar este frente en el bridge/director, no como workaround de
OPES:

- si el submit devuelve `supervision_unavailable`, dejar un estado durable que
  el Director residente consuma automaticamente, sin depender de llamada manual;
- reconciliar jobs enviados tras timeout con el ledger para no duplicar ni
  perder claims;
- validar contrato minimo de ACK/artifact antes del submit a OPES:
  `artifact_type`, `complete_job`, `idempotency_key` y payload trazable;
- validar causalidad de materializacion: si el `job.type` puede crear bloque
  OPES (`visual_asset`, `content_block`, `audio_asset`, etc.), el bridge debe
  comprobar que existen `canonical_topic_id` y `chapter_id` vivos, o entregar
  el resultado como artefacto de evidencia/insumo no materializable hasta la
  fase de ensamblado;
- no relanzar trabajos que ya dejaron artefacto valido en disco solo porque el
  submit fallo por FK; reconciliar el fichero, normalizar refs vivas cuando
  existan y conservar el insumo;
- si falta metadato recuperable, normalizar o pedir rework causal inmediato
  conservando el contenido como insumo, sin marcar cierre valido falso;
- exponer en status publico contadores de `submitted_after_timeout`,
  `supervision_unavailable_pending` y `artifact_contract_rework_created`.
- si un operador o reconciliador registra una entrega recuperable antes del ACK
  tardio, el cierre automatico debe deduplicar por `run_ref + job_ref +
  artifact_type + payload/evidence_ref` o marcar la segunda entrega como replay
  reconciliado, no como nuevo artefacto independiente.
- clasificar trabajos OPES con dependencia externa controlada (`edge-tts`,
  Whisper si descarga/modelo externo, API visual, etc.) y lanzarlos con perfil
  de red autorizado, timeout duro y fallback de bloqueo accionable; no usar
  sandbox sin red para comandos que necesitan red.
- validar el tool path de audio OPES (`opes_audio_app.py` o wrapper compatible)
  antes de lanzar el job; si falta, crear rework causal de herramienta en vez de
  colgar el agente.
- hacer que `runs/control action=stop` escriba checkpoint real en el workdir
  del agente y detenga tambien comandos hijo bloqueados; el estado publico debe
  diferenciar `stop_requested`, `stop_propagated` y `stop_confirmed`.
- hacer que el supervisor residente convierta runs OPES en
  `resident_director_pending` en despacho real o bloqueo causal visible antes de
  dos ticks, sin llamada manual a `/api/v0/runs/supervise`.
- reconciliar ACKs OPES ya completados con artefacto local valido sin caer en
  `domain_work_submit_conflict` terminal; si hay conflicto real, exponer
  `reconcile_manual_required` con job, artefacto, idempotency y causa exacta.
- impedir cierres `done/quiescent` sin progreso de dominio: si `projection`
  tiene cero tareas pero el job OPES externo sigue `pending`, el supervisor debe
  lanzar agente, reencontrar artefacto o publicar
  `quiescent_without_domain_progress`, nunca cerrar silenciosamente.
- reconciliar agentes tardios: si el job OPES asociado ya esta `completed`, la
  run debe pasar a no-op/replay reconciliado y no escribir ni subir un segundo
  artefacto salvo rework causal explicitamente autorizado.

## Criterio de cierre

Smoke real OPES temporal con al menos 17 jobs `draft_content_block`:

- todos los trabajos quedan enviados o reencontrados de forma idempotente;
- ningun job queda indefinidamente en `retry_pending` por supervision no
  disponible;
- los ACK incompletos producen normalizacion o rework causal automatico;
- los visuales de programa no producen FK 500: quedan como artefacto trazable
  no materializado o se reenvian con `canonical_topic_id/chapter_id` validos;
- el operador no necesita llamar manualmente a `/api/v0/runs/supervise`;
- `runs/supervise` no devuelve error terminal mientras exista proceso vivo,
  ACK pendiente o artefacto local recuperable; debe devolver `wait` con
  siguiente accion durable o cerrar/reconciliar;
- no se crean artefactos duplicados cuando una recuperacion manual o residente
  precede al ACK tardio de Orquesta;
- los trabajos `generate_audio_asset` con `edge-tts` se ejecutan con red
  controlada, timeout por comando y sin sustitucion silenciosa por motores no
  canonicos;
- `runs/control action=stop` corta una run bloqueada y sus procesos hijo, y el
  agente no escribe nuevos artefactos despues de la parada;
- el bridge detecta herramienta de audio canonica ausente y abre tarea
  accionable, no un agente colgado;
- una run OPES aceptada por `opes-drain-once` no queda indefinidamente en
  `resident_director_pending`: arranca agente, se reencuentra entrega o publica
  bloqueo causal sin intervencion manual;
- un ACK OPES completado con test pasado y artefacto local no deja el job
  `pending` sin artefactos; se registra una entrega idempotente o una accion
  durable de reconciliacion manual;
- una run OPES con `projection-tasks-0` no puede devolver `done/quiescent`
  mientras su job externo siga `pending`; debe producir progreso, reencuentro o
  bloqueo causal visible;
- un agente tardio no puede sobrescribir ni duplicar un artefacto ya aceptado
  para un job OPES `completed`; debe detectar el cierre y terminar con
  reconciliacion idempotente;
- las evidencias no filtran rutas sensibles fuera de diagnostico local.
