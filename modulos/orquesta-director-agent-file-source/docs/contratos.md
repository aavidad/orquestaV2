# Contratos

## DirectorAgentDecisionFileSourceV0

Implementa `DirectorAgentDecisionSourcePortV0`.

Entrada:
- `DirectorAgentDecisionSourceRequestV0`, con `Run.RunID` y metadatos compactos.

Puertos requeridos:
- `DirectorAgentDecisionFileDescriptorProviderPortV0`;
- `DirectorAgentDecisionFileReaderPortV0`.

Descriptor:
- `descriptor_ref`;
- `run_id`;
- `path`.

Formatos aceptados:
- una decision `director_agent_decision.v0`;
- lista JSON de decisiones;
- sobre `director_agent_decisions_file.v0` con `decisions`.

Salida:
- lista de `DirectorAgentDecisionV0` validada y lista para aplicar en workflow.

Errores:
- `director_agent_file_source_invalido: <field>`.

Modo tolerante opt-in:
- `IgnoreInvalidFiles=true` ignora artefactos de decision con contenido invalido
  o esquema no soportado y continua con los demas descriptores validos.
- No debe usarse como default del modulo: sirve para composiciones externas
  donde un agente de trabajo puede dejar ficheros auxiliares no contractuales
  junto al ACK sin que eso bloquee el run completo.
