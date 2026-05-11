# orquesta-app-runner

Lee primero este archivo, `README.md` y los docs locales necesarios.

Reglas locales:

- Adaptador de aplicacion: compone `AppSpecV0`, planner y nucleo por puertos.
- No decide proveedor, modelo, HOME, credenciales, DB, runtime ni filesystem.
- No importa `cmd` ni `db`.
- No ejecuta agentes reales por defecto; los conectores entran inyectados desde
  capas exteriores.
- Mantener ficheros pequenos y funciones pequenas.
