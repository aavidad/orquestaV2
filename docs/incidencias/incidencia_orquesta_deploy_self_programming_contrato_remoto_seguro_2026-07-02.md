# Incidencia Orquesta Deploy Self-Programming Contrato Remoto Seguro 2026-07-02

ID: `BUG-ORQ-20260702-116`
Estado: `cerrado`
Area: `orquesta-deploy` / `deploy/self-programming`

## Hallazgo

El perfil remoto aislado de self-programming tenia las reglas de seguridad en
documentacion y compose, pero no habia una prueba focal que fijara los
invariantes de Docker/env: puerto solo loopback, binds bajo
`/srv/orquesta-self`, ausencia de Docker socket y rutas productivas, usuario
no-root, `no-new-privileges`, `cap_drop: ALL`, OPES/DomainWork desactivado,
promocion desactivada y backend Goal por `app_server_tmux`.

Ademas, el compose dejaba el filesystem raiz mutable con `read_only: false`,
aunque los write paths esperados ya estan declarados como binds o tmpfs bajo
`/workspace` y `/tmp`.

## Riesgo

Una regresion pequena en el compose o el env de ejemplo podia abrir puerto
externo, montar una ruta productiva, reactivar OPES/DomainWork/promocion o
ejecutar el contenedor con mas superficie de escritura de la necesaria antes de
que el operador lo detectara por inspeccion manual.

## Cierre

- `deploy/self-programming/docker-compose.yml` pasa a `read_only: true`.
- `deploy/self-programming/self_programming_contract_test.go` valida el contrato
  estatico de compose, env y runbook.
- `deploy/self-programming/README.md` documenta el test y amplia el check de
  `docker inspect` para `ReadonlyRootfs`, `SecurityOpt` y `CapDrop`.

## Pruebas

- `go test -count=1 ./deploy/self-programming`
- `go test -count=1 ./modulos/orquesta-deploy`
- `git diff --check -- deploy/self-programming modulos/orquesta-deploy docs/incidencias docs/inventario_bugs_orquesta_2026-06-30.md`

## Limitacion

El cierre es contractual/estatico. No arranca Docker remoto ni sustituye el
`docker inspect` en el host despues del despliegue.
