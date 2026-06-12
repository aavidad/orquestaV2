# Pruebas: orquesta-runtime-worktree

```bash
go test -count=1 ./modulos/orquesta-runtime-worktree
```

Cobertura:

- snapshot de ficheros regulares con paths relativos;
- modo estricto de presupuesto de snapshot y modo parcial opt-in con
  `omitted_paths` relativos y recibos de exclusion;
- ignora prefijos de control inyectados;
- acepta cambios dentro del write-set;
- rechaza cambios fuera del write-set;
- rechaza borrados detectados aunque el path pertenezca al write-set;
- clasifica y bloquea truncados fuertes, renombres/movimientos ambiguos y
  reemplazos con delta grande dentro del write-set;
- permite `write_set=["."]` para apps nuevas completas;
- rechaza project workdir invalido y write-set inseguro;
- prepara worktree aislada preservando `branch_ref` opaco y sin filtrar paths
  absolutos;
- prepara worktree aislada con snapshot parcial opt-in si un fichero supera el
  presupuesto de lectura;
- rechaza `branch_ref` con forma de ruta y preparacion no aislada;
- revisa un repo Git limpio con `review_repo` sin modificarlo y sin filtrar
  rutas absolutas.
- reconciliacion T208 se valida con la bateria cruzada requerida, sin ampliar el
  contrato local ni convertir refs opacas en rutas.
