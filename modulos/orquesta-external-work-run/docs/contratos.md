# Contratos: orquesta-external-work-run

## `StartExternalWorkRunV0`

Entrada:

- `StartExternalWorkRunRequestV0`.
- `StartExternalWorkRunPortsV0`.
- `StartExternalWorkRunConfigV0`.

Salida:

- `StartExternalWorkRunResultV0`.

Responsabilidad:

- normalizar una solicitud de trabajo externo ya definida;
- crear el run si no existe;
- abrir directamente `programacion`;
- registrar el `AppChangeRequestV0`;
- encolar el run para el supervisor global.

Invariantes:

- no arranca director LLM inicial;
- no conoce OPES, DB, runtime, proveedor, modelo ni filesystem;
- todos los efectos salen por puertos;
- exige `external_work` porque este flujo no sustituye a crear app completa;
- mantiene idempotencia por refs compactas de run, comandos y cola.
