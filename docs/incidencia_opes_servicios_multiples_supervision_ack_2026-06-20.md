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

## Criterio de cierre

Smoke real OPES temporal con al menos 17 jobs `draft_content_block`:

- todos los trabajos quedan enviados o reencontrados de forma idempotente;
- ningun job queda indefinidamente en `retry_pending` por supervision no
  disponible;
- los ACK incompletos producen normalizacion o rework causal automatico;
- los visuales de programa no producen FK 500: quedan como artefacto trazable
  no materializado o se reenvian con `canonical_topic_id/chapter_id` validos;
- el operador no necesita llamar manualmente a `/api/v0/runs/supervise`;
- las evidencias no filtran rutas sensibles fuera de diagnostico local.
