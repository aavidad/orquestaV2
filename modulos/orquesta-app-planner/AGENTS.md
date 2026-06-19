# orquesta-app-planner

Lee primero este archivo y `README.md`. Despues lee solo los docs locales
necesarios para tu microtarea.

Reglas locales:

- Modulo puro de planificacion de app: no lanza agentes, no toca procesos, no
  lee archivos reales y no decide proveedor, modelo, HOME u OAuth.
- La salida debe ser compacta: microtareas, write-set, dependencias y candidatos
  para puertos existentes.
- Todo plan de app completa debe emitir tareas compatibles con hexagonalidad
  estricta: domain/application, ports, adapters y bootstrap/cmd como unica
  composicion.
- No introducir DB de Orquesta ni motor concreto. Si la app generada necesita
  persistencia, debe aparecer como requisito externo futuro, no como dependencia
  del nucleo.
- Mantener ficheros pequenos. Si una funcion crece, dividir por responsabilidad.
- No usar `cmd` ni `db` legacy.
