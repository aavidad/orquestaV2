# Pruebas

- Mapper de `summarize_topic` a `topic_summary`.
- Mapper de `expand_topic_from_summary` a `topic_expansion_package`.
- Mapper de `plan_tema` a `document_plan` con schema
  `domain_document_plan.v0`, contexto `large`, partes minimas del plan y
  criterios de aceptacion para no redactar el documento final, incluida la
  politica OPES de derivacion descendente desde maestro A1/A2 o A1 cuando
  exista equivalente superior.
- Mapper de `plan_temario` de operadores a `document_plan` con write-set
  `external/opes/plan_temario/<job>`, contexto `large`, schema
  `domain_document_plan.v0`, contrato de plan documental, flujo editorial OPES,
  metodo de asimilacion y requisitos de calidad.
- Contrato multiformato de expansion: tema grande, tema mediano, resumen,
  esquema de repaso y plan de visuales.
- Preservacion de payload JSON como `input_fields`.
- Hidratacion de `topic_blocks` para `summarize_topic`.
- Write-set unico para evitar entregas multiples innecesarias.

Comando:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge
```
