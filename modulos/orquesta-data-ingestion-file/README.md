# orquesta-data-ingestion-file

Opt-in hexagonal adapter for `orquesta-data-ingestion` local CSV and JSON
sources. Construction accepts an explicit allowed root and an opaque-ref
catalogue. It implements `DataSourcePortV0` and `DataProfilerPortV0` using Go
stdlib only.

Only regular files beneath the allowed root are accepted. Traversal, absolute
paths and any symlink in a registered source path are rejected. CSV is read as
a header plus records; JSON is read as a top-level array of object rows.

The adapter hashes content with SHA-256 and exposes opaque content-addressed
snapshot/provenance refs. Its configured byte and row limits apply to hashing
and profiling. It does not implement mapping, domain validation, XLSX, ODS,
Parquet, SQL, server wiring, or SDK/capability integration.
