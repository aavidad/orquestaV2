# Incidencia: topic_registry liberaba paquete final con QA generica

Fecha: 2026-07-02

## Resumen

`topic_registry` podia liberar un `completed_syllabus_package` con evidencia
generica `opes-final-evidence:qa` si el resto de categorias estaban presentes.
Eso no garantizaba que el paquete hubiese pasado la terna QA exigida por el
contrato OPES vigente.

## Causa

El stack final y el bridge OPES ya separaban `extension_pass`,
`official_text_qa_pass` y `strict_editorial_qa_pass`, pero
`topic_registry` seguia aceptando una categoria historica `qa` como suficiente.
Habia dos contratos de cierre de paquete final en paralelo.

## Cierre aplicado

- `topic_registry` exige manifest compatible `manifest_cierre` con:
  - evidencias requeridas `html`, `rag`, `audio`, `tests`, `visual`, `qa`;
  - `qa_passes.extension_pass`;
  - `qa_passes.official_text_qa_pass`;
  - `qa_passes.strict_editorial_qa_pass`;
  - `qa_report_refs.extension`, `official_text` y `strict_editorial`.
- Si no hay manifest compatible, exige evidencias estrictas separadas para la
  terna QA.
- La evidencia historica `opes-final-evidence:qa` ya no libera el registro por
  si sola.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-opes-director
```

Casos cubiertos:

- Manifest + QA generica no libera el paquete.
- Manifest compatible + terna QA libera el paquete.

## Estado

Cerrado en codigo local. Pendiente de commit/push y sincronizacion remota junto
al lote verificado.
