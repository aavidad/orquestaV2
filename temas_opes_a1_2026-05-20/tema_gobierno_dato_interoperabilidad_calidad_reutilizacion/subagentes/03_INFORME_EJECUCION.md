# Informe de ejecución - bloque 03

## Alcance

Se ha creado un bloque de desarrollo teórico principal para el tema "Gobierno del dato, interoperabilidad semántica, calidad del dato y reutilización de la información pública".

El bloque está pensado como material integrable por el responsable del tema final. No se ha editado `tema_a1.md`, `tema_a1.html`, bancos de preguntas finales ni artefactos fuera de `subagentes/`.

## Artefactos producidos

- `03_desarrollo_teorico_principal.md`: desarrollo conceptual con orientación de examen, mapa inicial, definiciones, tablas, ejemplos administrativos, supuesto práctico, notas de test separadas, errores frecuentes, modo tutor, repaso final, plan de visuales y fuentes oficiales de referencia.

## Recuento

- Palabras del bloque teórico: 8.455.
- Estado: material parcial, no tema A1 completo.

## Validaciones ejecutadas

```bash
wc -w external/opes/a1/tema_gobierno_dato_interoperabilidad_calidad_reutilizacion/subagentes/03_desarrollo_teorico_principal.md
```

Resultado: 8.455 palabras.

```bash
rg -n "https?://|www\\." external/opes/a1/tema_gobierno_dato_interoperabilidad_calidad_reutilizacion/subagentes/03_desarrollo_teorico_principal.md
```

Resultado: sin coincidencias.

```bash
rg -n "Orquesta|Codex|prompt|subagente|agente|wave_ref|run_ref|parent_agent|child_agent|arquitectura de ejecucion|arquitectura de ejecución|ruta interna" external/opes/a1/tema_gobierno_dato_interoperabilidad_calidad_reutilizacion/subagentes/03_desarrollo_teorico_principal.md
```

Resultado: sin coincidencias.

## Validadores obligatorios del tema final

```bash
words=$(wc -w < external/opes/a1/tema_gobierno_dato_interoperabilidad_calidad_reutilizacion/subagentes/03_desarrollo_teorico_principal.md); if [ "$words" -ge 20250 ] && [ "$words" -le 22500 ]; then echo "PASS validar-palabras-a1-20250-22500 words=$words"; else echo "FAIL validar-palabras-a1-20250-22500 words=$words expected=20250-22500 artifact=partial"; fi
```

Resultado: `FAIL validar-palabras-a1-20250-22500 words=8455 expected=20250-22500 artifact=partial`.

Interpretación: fallo esperado sobre este bloque parcial. El tema final deberá alcanzar 20.250-22.500 palabras antes de declararse listo.

```bash
file=external/opes/a1/tema_gobierno_dato_interoperabilidad_calidad_reutilizacion/subagentes/03_desarrollo_teorico_principal.md; missing=0; for p in "Orientación de examen" "Mapa inicial" "Definiciones" "| Eje |" "Ejemplos administrativos" "Supuesto práctico" "Notas de test" "Errores frecuentes" "Repaso final" "Plan de visuales" "Fuentes oficiales"; do if ! rg -q "$p" "$file"; then echo "MISSING $p"; missing=1; fi; done; if [ "$missing" -eq 0 ]; then echo "PASS validar-politica-editorial-opes-a1-local estructura_minima_bloque"; else echo "FAIL validar-politica-editorial-opes-a1-local estructura_minima_bloque"; fi
```

Resultado: `PASS validar-politica-editorial-opes-a1-local estructura_minima_bloque`.

Interpretación: comprobación local parcial de estructura y política editorial. La validación final debe ejecutarse sobre el `tema_a1.md` ensamblado y sincronizado con el HTML.

## Fuentes consultadas

Se han usado como base fuentes oficiales y de referencia pública sobre ENI, reutilización de información pública, datos abiertos, protección de datos, seguridad e interoperabilidad europea. En el bloque integrable se citan editorialmente sin URLs visibles para facilitar la posterior política de publicación.

## Huecos pendientes para integración

- Completar el resto del tema hasta el rango A1 exigido.
- Ensamblar `tema_a1.md` sin perder continuidad pedagógica.
- Crear o sincronizar `tema_a1.html`, `fuentes.md`, `checklist_a1.md`, `assets/` y `banco_preguntas_i18n/es/`.
- Ejecutar los validadores finales sobre el tema completo.
