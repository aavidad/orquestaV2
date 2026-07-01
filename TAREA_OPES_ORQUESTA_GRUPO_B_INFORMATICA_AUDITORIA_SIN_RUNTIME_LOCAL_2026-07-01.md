# Tarea Orquesta - Auditoría Grupo B Informática sin runtime local

Fecha: 2026-07-01

## Contexto

Durante la auditoría OPES del temario `Grupo B Informática`, Codex intentó usar Orquesta como superficie de dirección, pero no encontró servicio local respondiendo en los puertos habituales `19023-19028`.

No se ha tocado código de núcleo Orquesta. Esta tarea documenta la excepción para que el agente de Orquesta la revise si procede.

## Evidencia

Comprobación realizada:

`curl -fsS --max-time 2 http://127.0.0.1:<puerto>/healthz`

Puertos comprobados:

- `19023`
- `19024`
- `19025`
- `19026`
- `19027`
- `19028`

Resultado: conexión rechazada en todos los puertos.

## Impacto en OPES

La auditoría se pudo cerrar con validadores directos, consultas locales y consultas remotas, pero no quedó registrada como ola dirigida por Orquesta.

Esto no cambia el dictamen del temario: Grupo B Informática queda en `rework_mayor`.

Informe OPES generado:

`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-06-04/rework/package_grupo_b_final_local_2026-06-04/09_validacion/auditoria_calidad_2026-07-01/INFORME_AUDITORIA_CALIDAD_GRUPO_B_INFORMATICA_2026-07-01.md`

## Acción esperada

Revisar si el arranque actual de Orquesta debe exponer una forma estable y documentada para que OPES pueda:

1. detectar runtime activo;
2. lanzar auditoría de calidad de temario existente;
3. registrar estado `rework_mayor`;
4. materializar tareas de rework por tema;
5. devolver estado vivo de la ola.

No se pide parche de emergencia. Se pide integrar este caso en el flujo normal si todavía no está cubierto.
