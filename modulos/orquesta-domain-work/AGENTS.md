# AGENTS: orquesta-domain-work

Lee este archivo y `README.md` antes de editar este modulo.

## Reglas

- Mantener el modulo puro y pequeno.
- Hexagonal siempre: solo contratos, DTOs y puertos.
- No importar DB, filesystem productivo, red, runtime, proveedores, modelos ni
  adaptadores concretos.
- No introducir semantica de programacion, OPES ni ningun dominio especifico.
- i18n queda fuera: aqui solo hay enums, codigos y refs opacas.
- Si hace falta un conector real, crear otro modulo.

## Alcance

Define el contrato generico para que una app externa pida trabajo a Orquesta y
reciba artefactos de vuelta sin conocer agentes, sesiones ni proveedores.
