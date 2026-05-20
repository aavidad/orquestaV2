# Corte OPES Psicologia A1/A2 con Orquesta - 2026-05-20

## Resumen

Estado al corte: lote 003 terminado; produccion de Psicologia A1/A2 aun en
curso porque faltan lotes posteriores.

- Temas con `tema_a1.md` producido y en rango A1: 15 temas, del 019 al 033.
- Temas que pasan el validador mecanico v1 completo: 10 temas, del 024 al 033.
- Temas que necesitan normalizacion quirurgica bajo v1: 5 temas, del 019 al
  023.
- Temas pendientes sin iniciar en esta tanda: 39 temas del manifiesto actual.
- Temas comunes: el usuario indico que ya existen y no deben rehacerse.
- Palabras totales en `tema_a1.md` de los temas 019-033: 313.013.

Directorio de produccion:

- `/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/produccion_orquesta_2026-05-20`

Manifiesto de lotes:

- `/home/alberto/Trabajo/orquesta/.orquesta-runtime/opes-psicologia-complete-20260520/batches.json`

## Regla operativa fijada

- No borrar nada sin revisar utilidad.
- No tocar temas existentes sin copia local previa.
- Si un tema esta mal, estudiar primero el fallo y corregir de forma localizada.
  Rehacer entero solo es excepcion justificada cuando sea mas sencillo o seguro.
- En cada ejecucion nueva, purgar solo el runtime nuevo con `--purge-runtime`.
  No purgar directorios de produccion ni artefactos de tema.
- Si hay duda sobre parar, borrar, relanzar encima, sobrescribir o cambiar una
  ejecucion viva, preguntar antes al usuario.

## Incidencia

Se corto indebidamente el lote `psico-a1-lote-003-20260520` con `SIGTERM`
cuando el usuario pidio documentar y guardar. Esto fue un error operativo.

Para preservar datos:

- No se borro ningun directorio de tema.
- Se guardo copia del estado interrumpido en:
  `/home/alberto/Trabajo/OPES/backups/psicologia_lote_003_estado_interrumpido_2026-05-20_19-56-06`
- Se relanzo una continuacion con runtime nuevo:
  `psico-a1-lote-003-continuacion-20260520`
- La continuacion debe leer y reutilizar todo lo ya producido, no empezar desde
  cero.

## Estado de temas

### Producidos

Lote 001:

- Tema 019: texto en rango; pendiente HTML v1.
- Tema 020: texto en rango; pendiente HTML v1 y 2 duplicados largos.
- Tema 021: texto en rango; pendiente HTML v1 y 7 duplicados largos.
- Tema 022: texto en rango; pendiente HTML v1.
- Tema 023: texto en rango; pendiente HTML v1 y 2 duplicados largos.

Informe:

- `/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/produccion_orquesta_2026-05-20/revision/ORQUESTA_VALIDACION_LOTE_001_2026-05-20.md`

Lote 002:

- Tema 024: corregido de forma localizada y validado.
- Tema 025: validado.
- Tema 026: corregido de forma localizada y validado.
- Tema 027: corregido de forma localizada y validado.
- Tema 028: corregido de forma localizada y validado.

Informe:

- `/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/produccion_orquesta_2026-05-20/revision/ORQUESTA_VALIDACION_LOTE_002_2026-05-20.md`

### Cerrados v1

Lote 003:

- Tema 029: validado v1.
- Tema 030: validado v1.
- Tema 031: validado v1.
- Tema 032: validado v1.
- Tema 033: validado v1.

Runtime original interrumpido:

- `/home/alberto/Trabajo/orquesta/.orquesta-runtime/codex-waves/psicologia/psico-a1-lote-003-20260520`

Runtime de continuacion terminado:

- `/home/alberto/Trabajo/orquesta/.orquesta-runtime/codex-waves/psicologia/psico-a1-lote-003-continuacion-20260520`

Objetivo de continuacion:

- `/home/alberto/Trabajo/orquesta/.orquesta-runtime/opes-psicologia-complete-20260520/objectives/psico-a1-lote-003-continuacion-20260520.md`

## Cambios hechos en Orquesta

Se dejo programado que los agentes OPES reciban reglas de creacion de temas y
web:

- Reglas editoriales OPES en prompts Codex cuando `project_ref/domain_ref` es
  `opes`.
- Regla de modificacion incremental de temas existentes.
- Prohibicion de lanzar subagentes manuales desde los agentes: Orquesta los
  materializa.
- Prohibicion de usar generadores para redactar doctrina final o inflar
  palabras.
- Plantilla HTML canonica tipo Tema 11:
  `modulos/orquesta-opes-bridge/templates/opes_html_topic_template_v1.html`
- Renderer:
  `modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py`
- Validador mecanico:
  `modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`

Ficheros tocados en Orquesta:

- `cmd/orquesta-server/codex_director_wave_command_v0.go`
- `cmd/orquesta-server/codex_director_wave_command_v0_test.go`
- `modulos/orquesta-opes-bridge/README.md`
- `modulos/orquesta-opes-bridge/docs/contratos.md`
- `modulos/orquesta-opes-bridge/docs/pruebas.md`
- `modulos/orquesta-opes-bridge/document_plan_contract_v0.go`
- `modulos/orquesta-opes-bridge/mapper_v0.go`
- `modulos/orquesta-opes-bridge/mapper_v0_test.go`
- `modulos/orquesta-opes-bridge/templates/opes_html_topic_template_v1.html`
- `modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py`
- `modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`

Pendiente de decidir antes de commitear:

- `temas_opes_a1_2026-05-20/` es una carpeta local de conveniencia con temas
  copiados para revision. No mezclar en el commit de codigo sin revisar si debe
  versionarse.

## Verificacion ejecutada

Antes de este corte:

```bash
git diff --check
go test -count=1 ./modulos/orquesta-opes-bridge
go test -count=1 ./cmd/orquesta-server -run TestCodexLaunchDirectorWaveCommandV0OPESRecursiveDryRunMaterializaHijosYReglas
go test -count=1 ./...
python3 -m py_compile modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py
```

Resultado observado: verde.

## Comandos utiles para continuar

Estado de padres de la continuacion:

```bash
go run ./cmd/orquesta-server codex-wave-status \
  --runtime-dir /home/alberto/Trabajo/orquesta/.orquesta-runtime/codex-waves/psicologia/psico-a1-lote-003-continuacion-20260520 \
  | jq -r '.agents[] | [.agent_ref,.status,.pid,.stderr_bytes] | @tsv'
```

Estado de hijos:

```bash
for d in /home/alberto/Trabajo/orquesta/.orquesta-runtime/codex-waves/psicologia/psico-a1-lote-003-continuacion-20260520/children/*; do
  basename "$d"
  go run ./cmd/orquesta-server codex-wave-status --runtime-dir "$d" \
    | jq -r '[.agents[] | .status] | group_by(.) | map({(.[0]): length}) | add'
done
```

Validar lote 003 cuando todos paren:

```bash
python3 modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py \
  --manifest /home/alberto/Trabajo/orquesta/.orquesta-runtime/opes-psicologia-complete-20260520/batches.json \
  --batch-index 2 \
  --base-dir /home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/produccion_orquesta_2026-05-20 \
  --report /home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/produccion_orquesta_2026-05-20/revision/ORQUESTA_VALIDACION_LOTE_003_2026-05-20.md
```

Regenerar HTML de un tema con plantilla si hace falta:

```bash
python3 modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py \
  --markdown "$TOPIC_DIR/tema_a1.md" \
  --output "$TOPIC_DIR/tema_a1.html" \
  --base-dir "$TOPIC_DIR" \
  --eyebrow 'Psicologia A1/A2'
```

## Proxima sesion

1. No parar la continuacion viva salvo confirmacion expresa del usuario.
2. Normalizar quirurgicamente lote 001 con copia previa.
3. Validar lote 001 con `modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`.
4. Cuando lote 001 este OK, lanzar lote 004 con `--purge-runtime` en runtime
   nuevo y con la misma regla de 5 temas x 6 subagentes.
