# Pruebas: orquesta-runtime-worktree

```bash
go test -count=1 ./modulos/orquesta-runtime-worktree
```

Cobertura:

- snapshot de ficheros regulares con paths relativos;
- ignora prefijos de control inyectados;
- acepta cambios dentro del write-set;
- rechaza cambios fuera del write-set;
- permite `write_set=["."]` para apps nuevas completas;
- rechaza project workdir invalido y write-set inseguro;
- prepara worktree aislada preservando `branch_ref` opaco y sin filtrar paths
  absolutos;
- rechaza `branch_ref` con forma de ruta y preparacion no aislada.
