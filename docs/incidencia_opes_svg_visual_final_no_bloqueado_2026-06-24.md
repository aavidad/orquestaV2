# Incidencia OPES: SVG Como Visual Final No Bloqueado

Fecha: 2026-06-24.

## Problema

Durante la integración local de `auxiliar-de-servicios-generales`, se copiaron
SVG heredados de comunes/canones a `html_final/img` y `html_ampliado/img` como
si fueran infografías finales. Esto contradice la regla OPES: SVG, cajas,
flechas, bocetos y dibujos locales no son arte final publicable.

## Riesgo

Orquesta puede marcar como aceptado un curso con visuales pobres si solo comprueba
que existe un fichero de imagen o un manifest, sin distinguir entre:

- insumo visual reutilizable;
- raster profesional final para alumnado.

## Requisito

La app de Orquesta debe tratar como bloqueante cualquier cierre OPES de HTML,
QA, paquete o producción cuando haya:

- `.svg` en `html_final`, `html_ampliado`, `public` o paquete importable;
- `.svg` declarado como visual final en manifest;
- visuales pendientes sin tarea de rework;
- esquemas de cajas/flechas/bocetos integrados como arte final.

## Validadores OPES Disponibles

En OPES se han añadido/endurecido:

```bash
python3 /home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tools/validate_visual_asset_reuse.py /ruta/al/curso
python3 /home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tools/validate_professional_infographics.py /ruta/al/curso --require-all-topics
```

Orquesta debe invocarlos o incorporar una política equivalente en sus cierres
OPES. El primer validador permite estados pendientes documentados sin SVG final;
el segundo falla el cierre completo si faltan infografías/fotos profesionales.

## Criterio De Aceptación

Una run OPES no puede emitir `ready`, `apto`, `ready_profesional`,
`html_validado` ni subida si cualquiera de esos validadores devuelve fallo en
modo cierre. Si solo hay SVG heredado, el estado correcto es
`pendiente_infografia_profesional` con brief por tema.
