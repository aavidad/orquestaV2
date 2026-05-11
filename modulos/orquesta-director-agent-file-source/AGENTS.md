# orquesta-director-agent-file-source

Contexto local para Codex arrancado desde este directorio.

Reglas:
- Este modulo es un adaptador de entrada. No contiene nucleo, scheduling ni decisiones de producto.
- Las rutas de artefactos llegan por puertos inyectados; no se descubren HOME, credenciales ni directorios por defecto.
- No se mencionan proveedores, modelos, cuentas, cuotas ni bases de datos en codigo de produccion.
- El contrato de salida es `DirectorAgentDecisionSourcePortV0`.
- Ficheros pequenos y funciones acotadas. Si una responsabilidad crece, partirla antes de seguir.
- Errores publicos compactos; no exponer contenido completo del artefacto.

Lectura inicial:
1. `README.md`
2. `docs/contratos.md`
3. `docs/decisiones.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
