---
name: orquesta-programacion-release
description: Cerrar cambios de codigo: revisar worktree, archivar artefactos, ejecutar verificaciones, hacer commit, push y preparar despliegue seguro.
---

# Orquesta Programacion Release

Usa esta skill cuando el trabajo de programacion deba quedar limpio,
commiteado, pusheado o listo para despliegue.

## Worktree

1. `git status --short --branch`.
2. Separar codigo/docs utiles de artefactos generados.
3. Archivar salidas fuera del repo si no deben versionarse.
4. No borrar cambios ajenos ni ramas con worktree activo.
5. Revisar deletes antes de commit.

## Verificacion

- `git diff --check`.
- Pruebas focales.
- Suite completa si el cambio es transversal.
- Capturas/local si hay web.

## Commit

- Agrupar solo trabajo coherente.
- Mensaje breve y descriptivo.
- Rebase/pull antes de push si el remoto avanzo.
- No reiniciar ni tocar produccion sin confirmacion del operador.

## Entrega

Commit, branch, push, pruebas y pendientes futuros.
