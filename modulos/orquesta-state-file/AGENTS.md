# Contexto local

Lee este fichero antes de trabajar en `orquesta-state-file`.

Reglas:

- Este modulo es un adaptador periferico, no core.
- Persistencia por fichero JSON atomico; no introducir DB ni runtime concreto.
- Implementar puertos hexagonales de otros modulos sin filtrar detalles de HOME,
  proveedor, credenciales o sesiones reales.
- Mantener ficheros y funciones pequenos.
- Tests obligatorios para recrear instancia y recuperar estado.

