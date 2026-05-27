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

Familias actuales:

- `text_policy_v0.go`: patrones de texto con pinta de valor sensible efectivo.
- `detail_field_policy_v0.go`: activacion opt-in/acotada por frontera y campo.

Estado T15 2026-05-27: la reactivacion de `detalle_prohibido` queda cerrada y
reconciliada. El servidor usa `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto
en modo produccion con scope acotado, preserva `off` en modo `programming` y no
bloquea vocabulario operativo opaco como `runtime`, `provider`, `model`,
`codex`, `git`, `db` o `sql`.
