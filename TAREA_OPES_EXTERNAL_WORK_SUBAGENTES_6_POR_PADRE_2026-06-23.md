# Tarea Orquesta: materializar 6 subagentes OPES por padre

Fecha: 2026-06-23

## Problema

En OPES, un trabajo de temario completo debe crear un padre por tema y seis
subagentes/subroles por padre. En la prueba con Auxiliar de Servicios Generales
C2, `POST /api/v0/external-work/run` aceptó los 20 padres, pero no materializó
seis hijos reales por padre. Los seis subroles quedaron como instrucciones del
paquete del padre.

Además, al intentar someter 120 trabajos hijos mientras los 20 padres estaban
vivos, el endpoint `external-work/run` hizo timeout. Esto indica falta de cola
o admisión no bloqueante para expansión masiva OPES.

## Evidencia

- Curso:
  `/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-de-servicios-generales`
- Jobs padres:
  `00_control/orquesta_jobs/external_work_tema_001.json` ...
  `external_work_tema_020.json`
- Incidencia ampliada:
  `docs/incidencia_opes_external_work_no_materializa_6_subagentes_por_padre_2026-06-23.md`

## Criterio de aceptación

- Un external-work OPES con `interface_refs=["opes.padre-tema-6-subroles.v1"]`
  o `input_fields.subroles_required` crea una cohorte causal: 1 padre + 6 hijos.
- Los hijos tienen write-set propio dentro del tema y ACK causal propio.
- La API informa estado separado: padre, hijos pendientes, vivos, completados,
  fallidos, bloqueados.
- La admisión de lotes OPES grandes no bloquea el endpoint HTTP mientras hay
  muchos padres vivos; si no puede arrancar, encola rápido y devuelve estado
  `queued`.
- Tests: caso OPES de 1 tema produce 6 hijos; caso OPES de 20 temas puede
  aceptar 120 hijos sin timeout; caso no OPES mantiene comportamiento actual.
