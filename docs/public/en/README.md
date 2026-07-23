# Orquesta

Orquesta coordinates durable work through projects and Goals. The core owns
lifecycle decisions; HTTP, MCP, and CLI are adapters of the same command
registry.

## Language

Spanish is the default and fallback language. Responses never translate their
identifiers, references, hashes, error codes, or semantic keys. Only
human-facing text changes.

Current installations bundle Spanish and English. A compatible regional variant
uses its base language; a valid but unavailable locale uses the Spanish
fallback. An invalid tag returns an explicit error.

## Current surfaces

- The CLI renders human help and errors from the catalog.
- MCP renders instructions and descriptions from the same catalog.
- HTTP preserves the command registry's machine contract.
- This public documentation keeps an equivalent pair for each language.

Web, Wizard, notifications, and prompts will consume this contract when their
verticals exist. Their future presence is not presented as current
functionality.

## Guarantees

The catalog fails on missing keys, locale parity gaps, duplicate keys, or
incompatible formatting variables. Plurals, numbers, currency, dates, and time
zones are checked per locale without changing protocol data.
