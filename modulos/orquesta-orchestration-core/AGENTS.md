# Contexto para agentes: orquesta-orchestration-core

Lee este fichero antes de trabajar en este directorio.

## Reglas

- Mantener funciones y ficheros pequenos.
- No importar `orquesta/cmd`, `orquesta/db` ni adaptadores de base de datos.
- No hardcodear SQLite, Postgres, MySQL, Redis ni ningun motor.
- No conocer proveedores de agentes concretos ni rutas HOME/OAuth/tokens.
- Todo efecto externo entra por puerto.
- No recuperar `cmd`, `db`, `internal` ni control-plane legacy desde este miniproyecto.

## Objetivo local

Unir los modulos limpios del nucleo:

- `orquesta-director-supervised-burst`;
- `orquesta-director-cycle`;
- `orquesta-director-scheduler`;
- `orquesta-director-supervisor`;
- `orquesta-core-workflow`;
- `orquesta-core-replanner`;
- `orquesta-core-leases`;
- `orquesta-director-cycle-outbox`.

## Test minimo

```bash
go test ./modulos/orquesta-orchestration-core -count=1
```

Si se cambia un contrato importado desde `modulos/`, ejecutar tambien el test
del modulo afectado.
