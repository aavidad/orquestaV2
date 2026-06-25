# Tarea OPES Orquesta: runner TTS para edge-tts y corte de rework repetido

Fecha: 2026-06-25

## Problema observado

En el cierre de audio local del temario `oficial-de-servicios-multiples`,
Orquesta lanzo agentes reales y genero diagnostico util, pero la generacion con
`edge-tts` dentro del sandbox del agente expiro de forma repetida. Tras el primer
bloqueo, Orquesta abrio otro rework con el mismo camino tecnico y volvio a
quedar bloqueado.

Evidencia OPES:

- curso:
  `/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/oficial-de-servicios-multiples`
- informe de bloqueo:
  `09_validacion/audio_qa_final_20260625/INFORME_REWORK_AUDIO_LOCAL_20260625.md`
- plan generado por el agente:
  `09_validacion/audio_qa_final_20260625/audio_regeneration_plan_20260625.json`

## Comportamiento correcto esperado

Orquesta debe distinguir entre:

- rework editorial o de integracion que un agente puede resolver;
- efecto externo bloqueado por capacidad/sandbox, como TTS con red o permisos.

Si el mismo bloqueo tecnico se repite, Orquesta no debe abrir nuevas tareas
equivalentes. Debe dejar estado `blocked_external_capability`, conservar
checkpoint, registrar evidencia y pedir un runner host-capable o ejecutar la
tarea mediante una capacidad declarada.

## Cambio solicitado

Implementar en el nucleo de Orquesta:

1. Perfil/capacidad `tts_edge_host_runner` para trabajos OPES de audio.
2. Deteccion de bloqueo repetido por timeout de `edge-tts`.
3. Regla de no relanzar reworks identicos tras el segundo bloqueo con la misma
   firma tecnica.
4. Estado visible en API/UI: `blocked_external_capability`, con razon,
   comando, artefactos conservados y siguiente accion.
5. Evento de auditoria para que el director OPES sepa si debe desbloquear en
   local sin perder trabajo.

## Resultado provisional aplicado

El director OPES detuvo el rework repetido y genero los audios en entorno local
host-capable. No se debe considerar una solucion definitiva de Orquesta.
