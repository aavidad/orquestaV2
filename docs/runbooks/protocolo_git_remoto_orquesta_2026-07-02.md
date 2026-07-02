# Protocolo Git remoto Orquesta

Fecha: 2026-07-02.

Este runbook aplica al entorno aislado del servidor `uso.dipgra.cloud` usado
para autoprogramacion de Orquesta. No aplica a servicios productivos ni a OPES
productivo.

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

## Reglas de seguridad

- No usar root.
- No cambiar servicios productivos.
- No abrir puertos externos.
- No tocar temarios productivos.
- No subir artefactos temporales grandes ni runtime.
- No imprimir tokens ni claves privadas.
- Si una prueba usa OPES, debe ser temporal/aislada y documentada.
