# Contexto para agentes: orquesta-estado-vivo

Lee este archivo antes de editar el modulo.

## Responsabilidad

Modulo neutral para proyectar el ciclo de vida vivo de trabajos de Orquesta a
partir de evidencias aportadas por fuentes externas.

Las fuentes aportan evidencia; SOLO ConstruirProyeccionCicloVidaV0 decide fase

## Reglas

- Mantener el modulo puro: tipos, contratos y funcion de proyeccion determinista.
- No importar otros modulos de Orquesta, `cmd`, DB, filesystem, red, HTTP,
  MCP, web, runtime real, Codex, OPES, proveedores, modelos, HOME, OAuth ni
  credenciales.
- No leer reloj interno: `ConstruirProyeccionCicloVidaV0` recibe `ahora`.
- No resolver conflictos en silencio: proceso vivo y terminal simultaneos
  produce fase `conflicto` con codigo `proceso_vivo_tras_terminal`.
- No aceptar `Terminal=true` sin fuente y al menos una referencia durable;
  queda indeterminado y requiere reparacion.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-estado-vivo
go test -count=1 ./
go build ./...
```
