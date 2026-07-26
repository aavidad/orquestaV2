# Separación reversible del legado para consulta

Fecha: 2026-07-26

Estado: aplicada localmente, sin borrar contenido ni historia.

## Objetivo

Evitar que los agentes confundan el producto nuevo con las generaciones
anteriores. La superficie activa debe mostrar el único núcleo nuevo; el código
legacy se conserva fuera del directorio de trabajo para caracterización,
comparación y recuperación.

## Ubicaciones

- producto activo: `/home/alberto/Trabajo/orquestaV2`;
- copia completa navegable: `/home/alberto/Trabajo/orquestaV2-legacy-consulta`;
- bundle Git independiente:
  `/home/alberto/Trabajo/orquestaV2-backups/2026-07-26-pre-separacion-legacy/orquestaV2-completo.bundle`;
- rama remota de rescate:
  `backup/pre-separacion-legacy-20260726`;
- tag remoto de rescate:
  `backup/pre-separacion-legacy-20260726`.

Las tres superficies apuntan al corte de producto
`82644ed80f4eb826099d437a643df59a9c10315b`. El bundle fue validado con
`git bundle verify`; su SHA-256 es
`d819211492e7e95d837fe54806e40037b328b6c18e3cac2cc30489fedf25ba97`.

## Qué queda fuera de la vista activa

- `modulos/**`: 106 módulos y 3.961 ficheros legacy;
- `cmd/orquesta-server/**`;
- `cmd/orquesta-guardian/**`;
- `cmd/orquesta-cli/**`;
- `cmd/orquesta-bootstrap-diagnostic/**`.
- `external/**`, que solo contenía una salida OPES histórica;
- `opes-salidas/**`;
- `temas_opes_a1_2026-05-20/**`;
- `tcae.jpeg`.

`cmd/orquesta/**` permanece visible porque es el único binario productivo de la
reconstrucción.

No se han apartado en bloque `docs/**`, `scripts/**`, `testdata/**` ni datos
OPES: contienen material histórico y soporte vigente mezclados. Su separación
requiere un manifiesto por fichero con disposición y consumidor; ocultarlos por
intuición podría retirar evidencias, gates o herramientas de V1-V22.

## Mecanismo

El worktree activo usa estos patrones locales:

```text
/*
!/modulos/
!/cmd/orquesta-server/
!/cmd/orquesta-cli/
!/cmd/orquesta-guardian/
!/cmd/orquesta-bootstrap-diagnostic/
!/external/
!/opes-salidas/
!/temas_opes_a1_2026-05-20/
!/tcae.jpeg
```

Esto no crea una eliminación en Git y mantiene limpio `git status`. `rg`,
compiladores y agentes lanzados dentro del directorio activo no ven esas rutas.
`git grep` sí puede consultar el índice histórico, por lo que no debe usarse
como fuente de producto sobre rutas congeladas.

## Consulta y reversión

La consulta se realiza en la copia externa, sin editarla:

```bash
cd /home/alberto/Trabajo/orquestaV2-legacy-consulta
git status --short --branch
```

La separación local se puede revertir, si el operador lo ordena, con:

```bash
cd /home/alberto/Trabajo/orquestaV2
git sparse-checkout disable
```

No es necesario revertirla para restaurar contenido: el bundle puede clonarse,
la rama/tag remota puede extraerse y la copia de consulta conserva un worktree
completo verificado.

## Límite de esta medida

El `sparse-checkout` es una barrera local de contexto, no el cutover V34 ni una
eliminación de GitHub. Un clone o worktree nuevo debe aplicar la misma vista si
va a trabajar solo en el producto nuevo. La retirada versionada de las rutas
legacy sigue exigiendo manifiesto, equivalencia acreditada y ausencia de
consumidores.
