# Incidencia OPES ASG Audio Tool Path

Fecha: 2026-06-24.

## Contexto

Curso OPES:
`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-de-servicios-generales/`

Run afectada:
`run-external-work-aux-servicios-generales-c2-audio-final-20260624`.

## Fallo observado

Orquesta lanzo un agente de audio, pero el agente intento ejecutar
`scripts/opes_audio_app.py` desde `/home/alberto/Trabajo/OPES`. Esa ruta no
existia en ese momento en el entorno de herramientas. El generador operativo
real estaba en `/home/alberto/Trabajo/USO/web/scripts/tcae_audio_app.py`, con
nombre historico de TCAE.

La run se paro con:

```http
POST /api/v0/runs/control
action=stop
forced=true
```

El stop quedo confirmado: `agents_stop_requested=1`,
`agents_stop_confirmed=1`, `agents_in_flight=0`.

## Correccion aplicada fuera del nucleo

Se renombro la herramienta de audio en la app web:

- canonico: `/home/alberto/Trabajo/USO/web/scripts/opes_audio_app.py`;
- compatibilidad: `/home/alberto/Trabajo/USO/web/scripts/tcae_audio_app.py`.

Se actualizo la skill OPES de audio para indicar que `opes_audio_app.py` es la
ruta canonica y que `tcae_audio_app.py` es solo wrapper historico.

## Tarea tecnica para Orquesta

`ORQ-OPES-AUDIO-TOOLPATH-20260624`

Antes de lanzar un trabajo `generate_audio_asset` o equivalente, Orquesta debe:

1. resolver el directorio de herramientas (`/home/alberto/Trabajo/USO/web` para
   generacion local de audio OPES);
2. validar que existe `scripts/opes_audio_app.py`;
3. ejecutar `python3 scripts/opes_audio_app.py --help` como preflight corto;
4. si falla, no lanzar agente de generacion: abrir bloqueo accionable
   `opes_audio_tool_missing`;
5. incluir en el prompt del agente la ruta absoluta del `cwd` correcto y el
   comando canonico.

No se modifica el nucleo de Orquesta en esta incidencia para no pisar trabajo
del agente activo de nucleo; queda como tarea documentada.
