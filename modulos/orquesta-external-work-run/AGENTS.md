# AGENTS

Contexto local: este modulo crea runs operativos para trabajos externos ya definidos.

Reglas:
- mantener arquitectura hexagonal; todos los stores, colas y notificadores son puertos;
- no conocer OPES, DB, proveedor de modelo, runtime ni filesystem concreto;
- no arrancar un director LLM inicial;
- dejar el run listo para que el supervisor procese la tarea por cola;
- mantener ficheros pequenos y pruebas del contrato publico.
