# BUG-ORQ-20260711-220: primera migracion runtime_models insegura

Fecha: 2026-07-11. Estado: cerrado localmente por `bc5a2cdcf`.

## Hallazgo

La primera implementacion de `runtime_models` pasaba las pruebas funcionales,
pero una revision adversarial encontro problemas estructurales: comprobacion
`Lstat` seguida de `ReadFile` vulnerable a sustitucion, raiz/componentes sin
propiedad gobernada, secreto relativo al snapshot daemon en vez del proyecto,
lectura sin limite, URL y timeout no validados, relectura best-effort del
fichero y `enabled:false` ignorado si habia `base_url`.

Clasificacion: falso verde de configuracion/seguridad. Un valor no esta
gobernado solo porque aparezca en el fichero canonico; parser, snapshot,
secreto, proyeccion efectiva y adaptador deben compartir la misma resolucion.

## Cierre

- `runtime_models` entra en `orquesta.config.json` con `enabled`, `base_url`,
  `timeout_seconds` y `bearer_token_file`; no admite token crudo en JSON.
- La precedencia es env explicita, fichero y default. `enabled` explicito gana;
  solo en ausencia total de ese campo una URL conserva el auto-enable legacy.
- URL exige HTTP(S), host y ausencia de userinfo/query/fragment. Timeout exige
  entero entre 1 y 3600 segundos; entradas invalidas abortan startup.
- El secreto se resuelve respecto a `ProjectWorkDir`, no al snapshot. `os.Root`
  confina la apertura; raiz, directorios y fichero se validan antes/despues por
  identidad, propietario y permisos; symlinks y cambios se rechazan; lectura
  maxima 64 KiB.
- `effective_config` usa el mismo proyecto ya parseado, publica solo refs de
  presencia y marca env coexistente como `deprecated_env_used`.

Pruebas adversariales: traversal, symlink final/intermedio/raiz, directorio
escribible por grupo, fichero grande, snapshot en otra ruta, URL invalida,
timeout invalido/overflow, env malformada y `enabled:false`. Verificacion:
`go test -count=1 ./...`, ratchets env/T90 y `git diff --check` verdes.
