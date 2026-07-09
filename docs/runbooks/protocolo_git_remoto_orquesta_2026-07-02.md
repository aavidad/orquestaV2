# Protocolo Git remoto Orquesta

Fecha: 2026-07-02.
Actualizado: 2026-07-09.

Este runbook aplica al entorno aislado del servidor `uso.dipgra.cloud` usado
para autoprogramacion de Orquesta. No aplica a servicios productivos ni a OPES
productivo.

Este es el protocolo unico para cerrar trabajo remoto de Orquesta con cambios
de codigo. Un goal `complete`, un `orquesta_goal_result.v0` o una cola vacia no
equivalen a integracion si no existe recibo Git verificable.

## Criterio de cierre obligatorio

Un cambio remoto queda integrado solo si se conserva una de estas evidencias:

- `integration_status=integrated` con `integration_receipt_ref` que apunte a un
  commit local en la rama canonica y, si procede, push a GitHub.
- `integration_status=pending_integration` con bundle/patch exportado,
  `git status` del worktree origen y accion siguiente explicita.
- `integration_status=blocked_push` con error de `fetch`, `rebase`, `push` o
  autenticacion, sin ocultar la entrega del agente.

No cerrar como hecho si solo existe:

- resultado durable del agente sin `commit_sha`;
- cambios en un worktree piloto;
- `queue.count=0`;
- `pending_push` sin recibo;
- remote `origin` apuntando a un bundle temporal no sincronizable.

## Estado configurado

En el servidor, el repo aislado actual:

`/srv/orquesta-self/runtime/audit-13611445`

tiene `origin` apuntando a:

`git@github.com:aavidad/orquestador.git`

La autenticacion usa una deploy key dedicada:

`/home/berserk/.ssh/orquesta_github_ed25519`

La clave es solo para este servidor/repo. No copiar claves personales ni meter
claves privadas en el repositorio.

## Comprobar acceso

```bash
cd /srv/orquesta-self/runtime/audit-13611445
ssh -T git@github.com
git ls-remote --heads origin trabajo/plataforma-agentes
```

Resultado esperado de `ssh -T`: autenticacion correcta y aviso de que GitHub no
ofrece shell.

## Identidad Git

Antes de crear commits en un worktree remoto, comprobar que la identidad no es
anonima ni root:

```bash
git config user.name
git config user.email
id -un
```

Si falta identidad en el worktree aislado, configurarla localmente al repo con
un valor operativo de Orquesta. No usar identidad personal de otro operador:

```bash
git config user.name "Orquesta remoto"
git config user.email "orquesta-remoto@users.noreply.github.com"
```

## Checkout y worktrees

La rama canonica para esta tanda es `trabajo/plataforma-agentes`.

Antes de integrar desde un piloto:

```bash
git -C /srv/orquesta-self/worktrees/orquesta status --short --branch
git -C /srv/orquesta-self/worktrees/orquesta remote -v
git -C /srv/orquesta-self/worktrees/pilot-remoto-1 status --short --branch
```

El integrador debe copiar/aplicar solo el write-set declarado por el goal. No
mezclar artefactos de runtime, logs, caches, JSON temporales ni cambios de otro
agente. Si hay cambios ajenos en el worktree principal, parar y documentar
`pending_integration` con la evidencia; no resolver a ciegas.

## Commit seguro desde remoto

1. Revisar cambios:

```bash
git status --short
git diff --check
```

2. Staging explicito. No usar `git add .` si hay artefactos generados,
runtime, JSON de goal o docs de evidencia que no pertenezcan al cambio:

```bash
git add <fichero1> <fichero2>
git diff --cached --stat
git diff --cached --check
```

3. Ejecutar pruebas focales y, si el cambio es transversal, la suite acordada:

```bash
go test -count=1 ./modulos/<paquete> ./cmd/orquesta-server
```

4. Crear commit:

```bash
git commit -m "Mensaje breve del cambio"
```

## Sincronizar antes de push

Antes de empujar, traer la rama actual de GitHub:

```bash
git fetch origin trabajo/plataforma-agentes
```

Si el remoto esta en detached HEAD, crear/actualizar una rama local en el commit
actual:

```bash
git switch -C trabajo/plataforma-agentes
```

Rebase normal sobre GitHub:

```bash
git rebase origin/trabajo/plataforma-agentes
```

Si hay conflictos, resolverlos manteniendo:

- las filas nuevas del inventario de bugs;
- las incidencias nuevas;
- los tests del cambio remoto;
- los commits ya subidos por otros agentes.

Despues:

```bash
git diff --check
go test -count=1 ./modulos/<paquete-tocado> ./cmd/orquesta-server
```

## Push

```bash
git push origin HEAD:trabajo/plataforma-agentes
```

Si GitHub rechaza por non-fast-forward:

```bash
git fetch origin trabajo/plataforma-agentes
git rebase origin/trabajo/plataforma-agentes
git push origin HEAD:trabajo/plataforma-agentes
```

No usar `--force` ni `--force-with-lease` salvo orden explicita del operador.

## Modo bundle/patch oficial

Si el servidor no puede hacer `fetch`/`push` directo a GitHub, no se considera
fallo del agente de codigo. Se exporta una entrega integrable:

```bash
git status --short --branch > /srv/orquesta-self/runtime/integration/<ref>/status.txt
git diff --binary > /srv/orquesta-self/runtime/integration/<ref>/changes.patch
git diff --stat > /srv/orquesta-self/runtime/integration/<ref>/changes.stat
git bundle create /srv/orquesta-self/runtime/integration/<ref>/orquesta.bundle HEAD
```

El summary del goal o del integrador debe incluir:

- worktree origen;
- branch/HEAD;
- write-set aplicado;
- pruebas ejecutadas y resultado;
- ruta del patch o bundle;
- motivo exacto por el que no se hizo push directo;
- accion siguiente para el entorno con remote GitHub canonico.

El estado publico debe quedar como `pending_integration` o `blocked_push`, no
como cola vacia sin siguiente accion.

## Reglas de seguridad

- No usar root.
- No cambiar servicios productivos.
- No abrir puertos externos.
- No tocar temarios productivos.
- No subir artefactos temporales grandes ni runtime.
- No imprimir tokens ni claves privadas.
- Si una prueba usa OPES, debe ser temporal/aislada y documentada.
