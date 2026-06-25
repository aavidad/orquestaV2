# Contexto local: orquesta-goal

Lee este archivo y `README.md` antes de editar el modulo.

## Responsabilidad

Contrato neutral para trabajo dirigido por `goal`. Orquesta prepara reglas,
contexto, write-set, tests, artefactos y politicas de cierre; el runtime que
recibe el goal ejecuta el bucle operativo.

## Reglas

- Mantener el modulo puro: DTOs, puertos y validacion estructural.
- No importar `cmd`, DB, filesystem, HTTP, MCP, web, runtime real, Codex, OPES,
  HOME, OAuth, tokens, proveedor ni modelo.
- No convertir reglas blandas en veto automatico de contenido.
- Los cortes duros solo pertenecen a seguridad, causalidad, refs imposibles,
  datos sensibles o efectos externos no autorizados.
- El goal puede dirigir el trabajo por dentro; Orquesta gobierna por fuera con
  contrato, evidencia y validacion.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-goal
```
