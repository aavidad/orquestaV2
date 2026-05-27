# orquesta-director-agent

Contrato puro para que un agente director externo proponga el siguiente comando de Orquesta sin acoplar el core a una IA concreta.

Responsabilidad:

- validar `DirectorAgentDecisionV0`;
- mantener la decision compacta y sin detalles operativos;
- permitir que un adaptador cambie el director externo sin tocar el nucleo.
- dirigir con bajo coste de contexto: elegir el siguiente paso y delegar el trabajo pesado.
- proponer plan/equipo autonomo como DTO puro antes de materializar microtareas.

Fuera de alcance:

- elegir proveedor, credenciales, HOME o binarios;
- persistir estado;
- lanzar procesos;
- aplicar comandos al workflow.
- analizar toda la app en una sola respuesta o acumular contexto gigante.

Estado T15 2026-05-27: el director-agent no es owner del rail de detalle. Usa
`orquesta-rails` como politica comun, acepta refs opacas y corta solo valores
sensibles efectivos, material crudo o detalle operativo real en campos
activados.
