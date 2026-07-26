package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func TestBundledCatalogStrictResolutionAndManifest(t *testing.T) {
	catalog, err := LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v", err)
	}
	if got := catalog.Locales(); !reflect.DeepEqual(got, []string{"es", "en"}) {
		t.Fatalf("Locales() = %v", got)
	}
	if got := len(catalog.Keys()); got != 870 {
		t.Fatalf("Keys() count = %d", got)
	}
	for _, key := range []string{
		"command.intakes.dossier.confirm.description",
		"command.intakes.dossier.get.description",
		"command.intakes.dossier.prepare.description",
		"command.intakes.context.get.description",
		"command.intakes.recommendations.accept.description",
	} {
		if !contains(catalog.Keys(), key) {
			t.Errorf("missing dossier command key: %s", key)
		}
	}
	for _, obsolete := range []string{
		"resource.goal.description", "tool.artifacts.read.description",
		"tool.goals.amend.description", "tool.goals.create.description",
		"tool.goals.get.description", "tool.goals.list.description",
		"tool.system.status.description",
	} {
		if contains(catalog.Keys(), obsolete) {
			t.Errorf("obsolete key remains: %s", obsolete)
		}
	}

	tests := []struct {
		raw       string
		requested string
		resolved  string
		fallback  bool
	}{
		{"", "", "es", true},
		{"es", "es", "es", false},
		{"es-ES", "es-ES", "es", false},
		{"en-GB", "en-GB", "en", false},
		{"zh-Hans", "zh-Hans", "es", true},
		{"ca-ES-valencia", "ca-ES-valencia", "es", true},
	}
	for _, test := range tests {
		resolution, resolveErr := catalog.Resolve(test.raw)
		if resolveErr != nil {
			t.Errorf("Resolve(%q): %v", test.raw, resolveErr)
			continue
		}
		want := Resolution{Requested: test.requested, Locale: test.resolved, Fallback: test.fallback}
		if resolution != want {
			t.Errorf("Resolve(%q) = %+v want %+v", test.raw, resolution, want)
		}
	}
	for _, invalid := range []string{" en", "en_US", "not a locale", "x", "-es"} {
		if _, resolveErr := catalog.Resolve(invalid); !errors.Is(resolveErr, ErrLocaleInvalid) {
			t.Errorf("Resolve(%q) error = %v", invalid, resolveErr)
		}
	}

	english, err := catalog.Text("en", "command.goals.create.description")
	if err != nil || english != "Create a durable Goal for the selected project." {
		t.Fatalf("english text = %q, %v", english, err)
	}
	fallback, err := catalog.Text("zh-Hans", "error.not_found")
	if err != nil || fallback != "No se encontró el recurso solicitado." {
		t.Fatalf("fallback text = %q, %v", fallback, err)
	}
	if _, err := catalog.Text("es", "missing.key"); !errors.Is(err, ErrKeyMissing) {
		t.Fatalf("missing key error = %v", err)
	}

	manifest := catalog.Manifest()
	if manifest.DefaultLocale != "es" || manifest.FallbackLocale != "es" ||
		len(manifest.Surfaces) != 9 || len(manifest.PublicDocuments) != 1 {
		t.Fatalf("Manifest() = %+v", manifest)
	}
	manifest.EnabledLocales[0] = "mutated"
	manifest.Surfaces[0].KeySources[0] = "mutated"
	manifest.PublicDocuments[0].Locales["es"] = "mutated"
	if fresh := catalog.Manifest(); fresh.EnabledLocales[0] != "es" ||
		fresh.Surfaces[0].KeySources[0] == "mutated" ||
		fresh.PublicDocuments[0].Locales["es"] == "mutated" {
		t.Fatal("Manifest() exposes mutable catalog state")
	}
	locales := catalog.Locales()
	keys := catalog.Keys()
	locales[0], keys[0] = "mutated", "mutated"
	if catalog.Locales()[0] != "es" || catalog.Keys()[0] == "mutated" {
		t.Fatal("Locales()/Keys() expose mutable catalog state")
	}
}

func TestCodexPromptCatalogOwnsExactSafePlaceholderSet(t *testing.T) {
	catalog, err := LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	want := sortedCopy([]string{
		"app_spec_generation", "artifact_media_type", "capability_refs", "execution_ref",
		"goal_ref", "objective", "output_contract", "phase_criterion_refs", "phase_input_refs",
		"phase_key", "phase_ref", "phase_template_ref", "plan_generation", "project_ref",
		"role_key", "skill_refs", "tool_refs", "work_item_ref", "write_set",
	})
	for _, locale := range catalog.Locales() {
		message, err := catalog.message(locale, "prompt.codex.agent")
		if err != nil || !reflect.DeepEqual(message.placeholders, want) {
			t.Fatalf("locale=%s placeholders=%v want=%v error=%v", locale, message.placeholders, want, err)
		}
		for _, forbidden := range []string{"spec_hash", "session_ref", "secret", "token"} {
			if contains(message.placeholders, forbidden) {
				t.Fatalf("locale=%s forbidden prompt placeholder=%s", locale, forbidden)
			}
		}
	}
}

func TestCatalogPluralAndDeterministicFormatters(t *testing.T) {
	catalog, err := loadCatalog(testCatalogFS(
		map[string]any{"sample.items": map[string]string{"one": "%d elemento", "other": "%d elementos"}},
		map[string]any{"sample.items": map[string]string{"one": "%d item", "other": "%d items"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locale string
		count  int64
		want   string
	}{
		{"es", 0, "0 elementos"},
		{"es", 1, "1 elemento"},
		{"es", 2, "2 elementos"},
		{"es", 10_000_001, "10.000.001 elementos"},
		{"es", math.MaxInt64, "9.223.372.036.854.775.807 elementos"},
		{"es", math.MinInt64, "-9.223.372.036.854.775.808 elementos"},
		{"en", -1, "-1 item"},
		{"en", -10_000_001, "-10,000,001 items"},
		{"en", 12_345, "12,345 items"},
		{"zh-Hans", 2, "2 elementos"},
	} {
		got, pluralErr := catalog.Plural(test.locale, "sample.items", test.count)
		if pluralErr != nil || got != test.want {
			t.Errorf("Plural(%q,%d) = %q, %v want %q", test.locale, test.count, got, pluralErr, test.want)
		}
	}
	if _, err := catalog.Text("es", "sample.items"); !errors.Is(err, ErrMessageKind) {
		t.Fatalf("Text(plural) error = %v", err)
	}

	bundled, err := LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locale string
		value  int64
		want   string
	}{
		{"es", 1_234_567, "1.234.567"},
		{"en", 1_234_567, "1,234,567"},
		{"zh-Hans", -1_234, "-1.234"},
	} {
		got, formatErr := bundled.FormatNumber(test.locale, test.value)
		if formatErr != nil || got != test.want {
			t.Errorf("FormatNumber(%q,%d) = %q, %v want %q", test.locale, test.value, got, formatErr, test.want)
		}
	}
	for _, test := range []struct {
		locale string
		micros int64
		want   string
	}{
		{"es", 1_234_567_890, "EUR 1.234,567890"},
		{"en", 1_234_567_890, "EUR 1,234.567890"},
		{"zh-Hans", -1, "EUR -0,000001"},
	} {
		got, formatErr := bundled.FormatCurrency(test.locale, "EUR", test.micros)
		if formatErr != nil || got != test.want {
			t.Errorf("FormatCurrency(%q,%d) = %q, %v want %q", test.locale, test.micros, got, formatErr, test.want)
		}
	}
	if _, err := bundled.FormatCurrency("es", "eur", 1); !errors.Is(err, ErrCurrencyInvalid) {
		t.Fatalf("invalid currency error = %v", err)
	}

	instant := time.Date(2026, time.July, 23, 10, 4, 5, 0, time.UTC)
	if got, err := bundled.FormatDate("es", instant); err != nil || got != "23/07/2026" {
		t.Fatalf("Spanish date = %q, %v", got, err)
	}
	if got, err := bundled.FormatDate("en", instant); err != nil || got != "07/23/2026" {
		t.Fatalf("English date = %q, %v", got, err)
	}
	if got, err := bundled.FormatTimeZone("es", instant, "Europe/Madrid"); err != nil ||
		got != "23/07/2026 12:04:05 +02:00 Europe/Madrid" {
		t.Fatalf("Spanish datetime = %q, %v", got, err)
	}
	if got, err := bundled.FormatTimeZone("en", instant, "America/New_York"); err != nil ||
		got != "07/23/2026 06:04:05 AM -04:00 America/New_York" {
		t.Fatalf("English datetime = %q, %v", got, err)
	}
	if _, err := bundled.FormatTimeZone("es", instant, "../Madrid"); !errors.Is(err, ErrTimeZoneInvalid) {
		t.Fatalf("invalid timezone error = %v", err)
	}
	if _, err := bundled.FormatTimeZone("es", instant, "Local"); !errors.Is(err, ErrTimeZoneInvalid) {
		t.Fatalf("host-dependent timezone error = %v", err)
	}
}

func TestCatalogRejectsMalformedCatalogsAndManifest(t *testing.T) {
	validText := `{"sample.text":"value"}`
	tests := []struct {
		name string
		fsys fs.FS
		want error
	}{
		{"duplicate catalog key", rawCatalogFS(`{"sample.text":"uno","sample.text":"dos"}`, validText), ErrCatalogInvalid},
		{"trailing catalog JSON", rawCatalogFS(validText+` {}`, validText), ErrCatalogInvalid},
		{"empty text", rawCatalogFS(`{"sample.text":"  "}`, validText), ErrCatalogInvalid},
		{"duplicate plural form", rawCatalogFS(`{"sample.text":{"one":"%d uno","one":"%d otra","other":"%d otros"}}`, `{"sample.text":{"one":"%d one","other":"%d other"}}`), ErrCatalogInvalid},
		{"unknown plural form", rawCatalogFS(`{"sample.text":{"singular":"%d uno","other":"%d otros"}}`, `{"sample.text":{"one":"%d one","other":"%d other"}}`), ErrCatalogInvalid},
		{"invalid plural placeholder", rawCatalogFS(`{"sample.text":{"one":"uno","other":"otros"}}`, `{"sample.text":{"one":"%d one","other":"%d other"}}`), ErrCatalogInvalid},
		{"missing locale key", rawCatalogFS(validText, `{}`), ErrCatalogInvalid},
		{"kind mismatch", rawCatalogFS(validText, `{"sample.text":{"one":"%d one","other":"%d other"}}`), ErrCatalogInvalid},
		{"text placeholder mismatch", rawCatalogFS(`{"sample.text":"Hola {name}"}`, `{"sample.text":"Hello {actor}"}`), ErrCatalogInvalid},
		{"plural placeholder mismatch", rawCatalogFS(`{"sample.text":{"one":"%d {name}","other":"%d {name}"}}`, `{"sample.text":{"one":"%d {actor}","other":"%d {actor}"}}`), ErrCatalogInvalid},
		{"plural forms internal placeholder mismatch", rawCatalogFS(`{"sample.text":{"one":"%d {name}","other":"%d {actor}"}}`, `{"sample.text":{"one":"%d {name}","other":"%d {name}"}}`), ErrCatalogInvalid},
		{"malformed placeholder", rawCatalogFS(`{"sample.text":"Hola {name"}`, validText), ErrCatalogInvalid},
		{"noncanonical filename", withExtraFile(rawCatalogFS(validText, validText), "catalogs/EN.json", validText), ErrLocaleInvalid},
		{"undeclared catalog", withExtraFile(rawCatalogFS(validText, validText), "catalogs/fr.json", validText), ErrCatalogInvalid},
		{"duplicate manifest field", withManifest(rawCatalogFS(validText, validText), duplicateManifest()), ErrManifestInvalid},
		{"unknown manifest field", withManifest(rawCatalogFS(validText, validText), unknownManifest()), ErrManifestInvalid},
		{"wrong surface policy", withManifest(rawCatalogFS(validText, validText), wrongSurfacePolicy()), ErrManifestInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadCatalog(test.fsys)
			if !errors.Is(err, test.want) {
				t.Fatalf("loadCatalog() error = %v want %v", err, test.want)
			}
		})
	}
}

func TestCatalogConcurrentReadsAreRaceFree(t *testing.T) {
	catalog, err := LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errorsFound := make(chan error, 128)
	for index := 0; index < 128; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			locale := "es"
			if index%2 == 0 {
				locale = "en-GB"
			}
			if _, err := catalog.Text(locale, "error.conflict"); err != nil {
				errorsFound <- err
			}
			if _, err := catalog.FormatCurrency(locale, "EUR", int64(index)*1_000_001); err != nil {
				errorsFound <- err
			}
			_ = catalog.Keys()
			_ = catalog.Locales()
			_ = catalog.Manifest()
		}(index)
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func testCatalogFS(spanish, english map[string]any) fstest.MapFS {
	es, _ := json.Marshal(spanish)
	en, _ := json.Marshal(english)
	keys := make([]string, 0, len(spanish))
	for key := range spanish {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return fstest.MapFS{
		"manifest.json":    {Data: testManifest(keys)},
		"catalogs/es.json": {Data: es},
		"catalogs/en.json": {Data: en},
	}
}

func rawCatalogFS(spanish, english string) fstest.MapFS {
	return fstest.MapFS{
		"manifest.json":    {Data: testManifest([]string{"sample.text"})},
		"catalogs/es.json": {Data: []byte(spanish)},
		"catalogs/en.json": {Data: []byte(english)},
	}
}

func testManifest(keys []string) []byte {
	sources := make([]string, 0, len(keys))
	for _, key := range keys {
		sources = append(sources, catalogSourcePrefix+key)
	}
	manifest := Manifest{
		SchemaVersion: 1, DefaultLocale: "es", FallbackLocale: "es",
		EnabledLocales: []string{"es", "en"},
		Catalogs: []CatalogSource{
			{Locale: "es", Path: "catalogs/es.json"},
			{Locale: "en", Path: "catalogs/en.json"},
		},
		Surfaces: []Surface{
			{ID: "cli", State: "active", KeySources: sources, LiteralPolicy: "catalog_only"},
			{ID: "command_registry", State: "active", KeySources: []string{"surface:cli"}, LiteralPolicy: "registry_keys_catalog_values"},
			{ID: "http", State: "active", KeySources: []string{"surface:cli"}, LiteralPolicy: "machine_envelope_catalog_presenter"},
			{ID: "mcp", State: "active", KeySources: []string{"surface:cli"}, LiteralPolicy: "catalog_only"},
			{ID: "public_docs", State: "active", KeySources: []string{"public_documents"}, LiteralPolicy: "localized_document_bundle"},
			{ID: "notifications", State: "future", LiteralPolicy: "catalog_required_before_activation"},
			{ID: "prompts", State: "active", KeySources: []string{"surface:cli"}, LiteralPolicy: "catalog_only"},
			{ID: "web", State: "future", LiteralPolicy: "catalog_required_before_activation"},
			{ID: "wizard", State: "active", KeySources: []string{"surface:prompts"}, LiteralPolicy: "catalog_only"},
		},
		PublicDocuments: []PublicDocument{{
			ID: "orquesta.quickstart",
			Locales: map[string]string{
				"es": "docs/public/es/README.md", "en": "docs/public/en/README.md",
			},
		}},
	}
	content, _ := json.Marshal(manifest)
	return content
}

func withExtraFile(source fstest.MapFS, name, content string) fstest.MapFS {
	result := cloneMapFS(source)
	result[name] = &fstest.MapFile{Data: []byte(content)}
	return result
}

func withManifest(source fstest.MapFS, content []byte) fstest.MapFS {
	result := cloneMapFS(source)
	result["manifest.json"] = &fstest.MapFile{Data: content}
	return result
}

func cloneMapFS(source fstest.MapFS) fstest.MapFS {
	result := make(fstest.MapFS, len(source))
	for name, file := range source {
		copyFile := *file
		copyFile.Data = append([]byte(nil), file.Data...)
		result[name] = &copyFile
	}
	return result
}

func duplicateManifest() []byte {
	valid := string(testManifest([]string{"sample.text"}))
	return []byte(strings.Replace(valid, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1))
}

func unknownManifest() []byte {
	valid := string(testManifest([]string{"sample.text"}))
	return []byte(strings.TrimSuffix(valid, "}") + `,"unknown":true}`)
}

func wrongSurfacePolicy() []byte {
	valid := string(testManifest([]string{"sample.text"}))
	return []byte(strings.Replace(valid, `"literal_policy":"catalog_only"`, `"literal_policy":"unknown"`, 1))
}

func TestWizardManifestRejectsArbitraryStateAndSources(t *testing.T) {
	valid := testManifest([]string{"sample.text"})
	var manifest Manifest
	if err := json.Unmarshal(valid, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Surface)
	}{
		{"future state", func(surface *Surface) {
			surface.State = "future"
			surface.LiteralPolicy = "catalog_required_before_activation"
		}},
		{"empty sources", func(surface *Surface) { surface.KeySources = nil }},
		{"duplicate catalog claim", func(surface *Surface) {
			surface.KeySources = []string{"catalog:sample.text"}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneManifest(manifest)
			for index := range candidate.Surfaces {
				if candidate.Surfaces[index].ID == "wizard" {
					test.mutate(&candidate.Surfaces[index])
				}
			}
			content, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := loadCatalog(withManifest(rawCatalogFS(`{"sample.text":"uno"}`, `{"sample.text":"one"}`), content)); !errors.Is(err, ErrManifestInvalid) {
				t.Fatalf("loadCatalog() error = %v, want %v", err, ErrManifestInvalid)
			}
		})
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestNilCatalogReturnsStableErrors(t *testing.T) {
	var catalog *Catalog
	if _, err := catalog.Resolve("es"); !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("Resolve nil error = %v", err)
	}
	if _, err := catalog.Text("es", "error.internal"); !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("Text nil error = %v", err)
	}
	if got := fmt.Sprint(catalog.Keys(), catalog.Locales()); got != "[] []" {
		t.Fatalf("nil slices = %s", got)
	}
}
