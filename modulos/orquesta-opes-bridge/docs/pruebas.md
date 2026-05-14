# Pruebas

- Mapper de `summarize_topic` a `topic_summary`.
- Mapper de `expand_topic_from_summary` a `topic_expansion_package`.
- Contrato multiformato de expansion: tema grande, tema mediano, resumen,
  esquema de repaso y plan de visuales.
- Preservacion de payload JSON como `input_fields`.
- Hidratacion de `topic_blocks` para `summarize_topic`.
- Write-set unico para evitar entregas multiples innecesarias.

Comando:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge
```
