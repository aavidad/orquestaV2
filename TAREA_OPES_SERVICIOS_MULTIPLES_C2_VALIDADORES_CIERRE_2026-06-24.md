# Tarea Orquesta: Cierre OPES Servicios Múltiples C2 Con Validadores Bloqueantes

Fecha: 2026-06-24.

## Problema Detectado

Orquesta produjo material útil para `oficial-de-servicios-multiples`, pero el
flujo no lo materializó ni cerró correctamente en la ruta canónica OPES. El
paquete externo contenía HTML, tests, RAG, audio provisional y capturas, pero
quedó fuera del curso canónico y sin cierre global bloqueante.

Además, el sistema permitió que existiera una entrega aparentemente completa
aunque:

- 13 de 17 temas ampliados no llegaban al mínimo C2 de 4.500 palabras;
- había metacomentarios visibles en texto publicable;
- quedaban SVG de tema como arte final;
- faltaban visuales raster profesionales en varios temas;
- el audio estaba marcado como provisional y no podía contar para publicación.

## Cambio Necesario En Orquesta

El director OPES de Orquesta debe convertir los validadores de cierre en gates
duros antes de marcar un curso como terminado o candidato:

1. Ejecutar validador de extensión por nivel y materializar rework por tema si
   falla.
2. Ejecutar validador de texto público sin notas/metacomentarios y abrir rework
   textual si falla.
3. Ejecutar validador de visuales profesionales y rechazar SVG de tema en HTML
   publicable.
4. Bloquear audio hasta texto aprobado y exigir Whisper tras regeneración.
5. Integrar artefactos externos aceptados en la ruta canónica del curso, no
   dejarlos solo en `external/opes`.
6. Registrar en ACK final qué informes pasaron y cuáles siguen pendientes.

## Caso De Prueba

Curso:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/oficial-de-servicios-multiples`

Informes que deben gobernar el cierre:

- `09_validacion/informe_extension_temario.json`
- `09_validacion/informe_texto_publico_sin_notas_autor.json`
- `09_validacion/informe_visuales_profesionales.json`
- `09_validacion/INFORME_AUDIO_PROVISIONAL_PENDIENTE_QA_TEXTUAL_20260621.md`

Resultado esperado: Orquesta debe lanzar rework de los temas pendientes y no
marcar el curso como `ready`, `ready_candidate_html`, `ready_profesional` ni
`apto_para_subida_controlada` mientras cualquiera de esos informes falle.

## Incidencias De Ejecución Observadas En La Ola 2026-06-24

Durante la ola separada en `127.0.0.1:8799` se registraron 119 trabajos
externos: 17 padres y 102 subagentes. El reintento idempotente confirmó
`119/119` registrados, con 111 respuestas aceptadas y 8 conflictos de cambio ya
existente que indican alta previa.

Fallos que deben corregirse en Orquesta, no resolver solo con operación manual:

1. `POST /api/v0/external-work/run` puede tardar más de 25-35 segundos por
   trabajo bajo carga. El cliente ve timeouts y al reintentar recibe conflicto
   de cambio existente. Orquesta debe devolver confirmación rápida, exponer
   lectura por `request_id/change_ref` y hacer el alta masiva idempotente sin
   ambigüedad.
2. Algunos `POST /api/v0/external-work/run` devolvieron `500` con payloads de
   trabajo largos o bajo carga. La API debe degradar a error validable con
   código público, detalle accionable y sin dejar al director sin saber si el
   trabajo fue persistido.
3. `POST /api/v0/runs/supervise` quedó sin respuesta durante más de 60 segundos
   y hubo que interrumpirlo manualmente. La supervisión debe tener timeout,
   respuesta parcial o ejecución asíncrona con `supervision_ref`.
4. Se observaron `agent_ack.json` en disco antes de que el estado agregado
   reflejara entregas (`deliveries=0`). Orquesta debe ingerir ACKs de runtime
   de forma residente y marcar cada run como entregado, fallido o pendiente con
   evidencia visible.
5. La cola mantiene runs en `running` aunque ya tienen ACK o aunque no existe
   proceso/runtime asociado. En la ola de Servicios Múltiples C2, 25 runs
   figuraban como `running` pese a tener `agent_ack.json`, y el tema 4 quedó
   como único tema bajo mínimo porque `tema_004-rework-padre` y
   `tema_004-sub-redaccion_expansion` estaban marcados como `running/ready` sin
   arrancar agente de redacción. El scheduler llegó a lanzar roles auxiliares
   (`tests_tutor`, `visuales`) antes que el rol que desbloqueaba el validador de
   extensión.
6. La recuperación operativa por `POST /api/v0/runs/control` + `POST
   /api/v0/runs/queue/priority` funcionó a nivel de API, pero no bastó para
   priorizar inmediatamente el trabajo bloqueante. Orquesta debe tener un modo
   de rescate que materialice el siguiente paso causal necesario, no solo que
   cambie el estado de cola.

Estos puntos afectan a la autonomía real: el director humano no debería tener
que contar ACKs en el filesystem ni reconciliar manualmente cola, procesos y
entregas.

## Resultado Del Desbloqueo Manual Acotado

El 2026-06-24 21:59 Europe/Madrid se corrigió localmente el bloqueo del tema 4
porque era el único que impedía pasar el validador textual C2 y el run causal de
redacción no había sido priorizado correctamente. Cambios aplicados en OPES:

- `tema_004/tema_ampliado.md` y `tema_resumen.md`: limpieza de duplicados de
  tutor, retirada de metacomentarios visibles y ajuste de redacción.
- `html_final/tema_04.html` y `html_ampliado/tema_04.html`: sincronización del
  texto corregido y de la sección ampliada.
- `09_validacion/informe_extension_temario.{json,md}`: `pass`, 17/17 temas por
  encima de 4.500 palabras.
- `09_validacion/informe_texto_publico_sin_notas_autor.{json,md}`: `pass`, 0
  hallazgos en 36 ficheros.

Esto no sustituye la corrección de Orquesta. El producto esperado sigue siendo
que el scheduler detecte el validador fallido, materialice la tarea causal
necesaria, priorice redacción antes de trabajos auxiliares y cierre el ACK sin
intervención manual.
