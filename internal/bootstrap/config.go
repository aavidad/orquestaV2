package bootstrap

import (
	"context"

	"orquesta/internal/adapters/config/effectivefile"
	configtoml "orquesta/internal/adapters/config/toml"
	"orquesta/internal/config"
)

// loadConfigSnapshot composes the local TOML adapter with the pure config
// resolver. Process environment is captured once through the canonical
// registry and never read by downstream components.
func loadConfigSnapshot(ctx context.Context, sourcePath string) (config.Snapshot, error) {
	store, err := configtoml.Open(configtoml.Options{
		Path:            sourcePath,
		MaxSourceBytes:  config.SourceMaxBytes(),
		MaxReceiptBytes: config.AuditEntryMaxBytes(),
	})
	if err != nil {
		return config.Snapshot{}, err
	}
	document, err := store.Read(ctx)
	if err != nil {
		return config.Snapshot{}, err
	}
	environment, err := config.CaptureEnvironment()
	if err != nil {
		return config.Snapshot{}, err
	}
	return config.Resolve(config.ResolveOptions{
		TOML: document.Content, Environment: environment, SourcePath: sourcePath,
	})
}

func writeEffectiveSnapshot(ctx context.Context, snapshot config.Snapshot) error {
	content, err := snapshot.EffectiveJSON()
	if err != nil {
		return err
	}
	return effectivefile.Write(ctx, effectivefile.Options{
		Path: snapshot.ConfigEffectivePath(), Content: content,
		MaxExistingBytes: snapshot.ConfigEffectiveMaxExistingBytes(),
	})
}
