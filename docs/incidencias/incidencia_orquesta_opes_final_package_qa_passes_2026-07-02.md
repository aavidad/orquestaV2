# Incidencia: paquete final OPES cerraba con QA generica

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-097`.

Relacionado con: `BUG-ORQ-20260701-093` y `BUG-ORQ-20260702-095`.

## Sintoma

El cierre de un `completed_syllabus_package` OPES podia considerar suficiente
una categoria generica `qa` dentro de `manifest_cierre`. Eso no separaba tres
gates distintos:

- `extension_pass`
- `official_text_qa_pass`
- `strict_editorial_qa_pass`

Con esa forma, un paquete podia parecer listo aunque faltase QA estricta contra
andamiaje interno, contaminacion cruzada o textos no publicables.

## Causa

El contrato final OPES estaba agregado en una evidencia `qa` generica. Los
required tests y el validador del stack no exigian que el manifest incluyese
`qa_passes` terminales ni `qa_report_refs` auditables.

## Cierre

El puente OPES crea required tests separados para `extension_pass`,
`official_text_qa_pass` y `strict_editorial_qa_pass`. El stack Codex exige en
`manifest_cierre`:

- `qa_passes.extension_pass=true`
- `qa_passes.official_text_qa_pass=true`
- `qa_passes.strict_editorial_qa_pass=true`
- `qa_report_refs.extension`
- `qa_report_refs.official_text`
- `qa_report_refs.strict_editorial`

Si falta o falla la QA estricta, el cierre goal-first devuelve
`domain_work_opes_strict_editorial_qa_missing` y el builder deja el paquete
como no terminal.

## Evidencia

Pruebas locales:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web
git diff --check -- modulos/orquesta-opes-bridge modulos/orquesta-app-codex-stack modulos/orquesta-web
```
