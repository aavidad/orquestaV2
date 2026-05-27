# orquesta-factory-http

Adaptador HTTP para `orquesta-factory`.

Expone el puerto `POST /api/v0/apps/spec` y delega negocio en
`orquesta/modulos/orquesta-factory`.

## Frontera JSON publica

- Ruta canonica: `POST /api/v0/apps/spec`.
- Body maximo: 256 KiB.
- `Content-Type`: vacio legacy, `application/json` o subtipo `+json`.
- Rechaza JSON invalido, datos trailing y campos desconocidos con errores
  publicos compactos; no devuelve el body crudo.
- La respuesta correcta es preview de factory: `app_spec` y `backlog`. No crea
  plan operativo, run, cola ni efectos externos.
