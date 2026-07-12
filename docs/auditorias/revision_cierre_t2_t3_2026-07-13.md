# Revisión independiente del cierre T2/T3

Fecha: 2026-07-13. Commits revisados: `9621951ef0` (T2) y `915475ac97`
(T3).

## Evidencia verde

- `go test -mod=vendor -count=1 ./cmd/orquesta-server -run 'TestMCPDocumentTextExtract|TestMCPDataProfile'`
- `go test -mod=vendor -count=1 ./modulos/orquesta-document-extraction ./modulos/orquesta-document-extraction-pdf ./modulos/orquesta-data-ingestion ./modulos/orquesta-data-ingestion-file ./modulos/orquesta-mcp`
- Ambas tools quedan registradas con binding real en la composición canónica de
  test y rechazan referencias que escapan de sus inbox.

## Residuales que impiden acreditar cierre completo

1. La imagen `Dockerfile.self-programming` no instala `pdftotext` ni
   `poppler-utils`. El test T2 corre en el host, donde sí existe el binario, pero
   la misma tool dentro del Docker oficial falla con extractor no disponible.
2. T2 solo expone MCP. Falta la superficie HTTP dedicada exigida por H5-A.
3. La salida T2 limita número de páginas, no bytes ni líneas. Hasta veinte
   páginas completas pueden rebasar el límite de transporte de 64 KiB.
4. T3 llama directamente `ResolveDataSourceV0` y `ProfileDataSetV0`; no existe
   ningún caller productivo de `IngestDataV0`. Siguen sin ejecutarse mapper,
   validator y receipt store durable. La capacidad completa de ingesta no está
   conectada: solo lo está el perfilador.
5. El catálogo T3 anuncia `.xlsx` y `.ods`, pero el adaptador file efectivo
   implementa CSV/JSON. No se deben anunciar formatos sin ejecución real.
6. El smoke vivo cubre CSV, no JSON, y no acredita mapping, validación, receipt
   ni replay durable.

Veredicto: avances funcionales válidos, pero T2/T3 permanecen parciales hasta
cerrar esos residuales o acotar explícitamente el contrato público sin afirmar
que toda la capacidad está conectada.
