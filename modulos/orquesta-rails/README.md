# orquesta-rails

Responsabilidad: rails reutilizables, pequenos y puros para fronteras de
Orquesta.

Este modulo no decide producto, runtime, proveedor ni dominio. Solo ofrece
politicas reutilizables y revisables por otros modulos cuando una regla no
pertenece a una frontera concreta.

Reglas:

- un fichero por familia de rail;
- listas pequenas y nombradas;
- nada de adaptadores, filesystem, red, DB, HOME, OAuth ni runtime real;
- por defecto permisivo con refs opacas y vocabulario operativo;
- los rails estrictos deben tener tests externos con falsos positivos.
- hasta nueva orden, los rails estan offline por defecto: existen para auditoria
  e inventario, pero no deben cortar trabajo util. La redaccion no bloqueante
  puede sanitizar valores sensibles efectivos antes de proyectar diagnosticos o
  salidas publicas.

Familias actuales:

- `security_mode_v0.go`: `ORQUESTA_RAILS_MODE` se normaliza siempre a
  `offline` hasta nueva orden.
- `text_policy_v0.go`: patrones de texto con pinta de valor sensible efectivo.
- `detail_field_policy_v0.go`: activacion opt-in/acotada por frontera y campo.
- `filesystem_policy_v0.go`: validadores de rutas/comandos historicos.
- `redaction_policy_v0.go`: redaccion no bloqueante de secretos efectivos
  (tokens, credenciales con valor, bearer, DSN con password y material privado).
- `privacy_taxonomy_v0.go`: taxonomia historica de privacidad operacional.

Estado 2026-06-02: `ORQUESTA_RAILS_MODE` queda forzado a `offline`. El
servidor publica `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` y conserva la matriz de
scopes solo como inventario historico. Ninguna variable de entorno reactiva los
rails mientras siga esta orden: `enforced`, `audit`,
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` o scopes heredados se normalizan a flujo
permisivo.

La reintroduccion de cualquier bloqueo por rail es una tarea futura explicita:
revisar matriz, decidir owner/director, cambiar codigo/configuracion y probar
en vivo que no vuelve a parar trabajo aprovechable por nombres, alias, refs
opacas o contenido reutilizable. La redaccion no bloqueante no equivale a
reactivar bloqueo: no usa marcadores genericos como `prompt`, `payload`, HOME,
provider/model o vocabulario operativo sin valor secreto efectivo.
