# orquesta-runtime-required-test

Adaptador opt-in para ejecutar comandos de test requeridos por el Director
Operativo y devolver `RequiredTestEvidenceV0` mediante el runner del nucleo.

El modulo ejecuta solo binarios permitidos por configuracion, sin shell y sin
heredar entorno. La salida se guarda como artefacto local y al nucleo solo se
devuelve una ref relativa.
