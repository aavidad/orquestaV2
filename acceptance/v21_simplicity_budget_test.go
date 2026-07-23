package acceptance_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestV21SimplicityBudgetMeasuresProductInsteadOfTrustingFixture(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(fixture.ManifestPath))); os.IsNotExist(err) {
		t.Log("V21 product absent; physical LOC budget becomes mandatory when manifest appears")
		return
	} else if err != nil {
		t.Fatal(err)
	}

	i18nGo, err := filepath.Glob(filepath.Join(root, "internal", "i18n", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	i18nProduct := make([]string, 0, len(i18nGo))
	for _, path := range i18nGo {
		if !strings.HasSuffix(path, "_test.go") {
			i18nProduct = append(i18nProduct, path)
		}
	}
	if got := v21CountPhysicalLines(t, i18nProduct); got > fixture.SimplicityBudget.I18NProductLOC {
		t.Errorf("V21 i18n product LOC=%d max=%d", got, fixture.SimplicityBudget.I18NProductLOC)
	}

	dataPaths := []string{filepath.Join(root, filepath.FromSlash(fixture.ManifestPath))}
	for _, relative := range fixture.CatalogPaths {
		dataPaths = append(dataPaths, filepath.Join(root, filepath.FromSlash(relative)))
	}
	if got := v21CountPhysicalLines(t, dataPaths); got > fixture.SimplicityBudget.CatalogManifestDataLOC {
		t.Errorf("V21 catalog/manifest LOC=%d max=%d", got, fixture.SimplicityBudget.CatalogManifestDataLOC)
	}
	docPaths := make([]string, 0, len(fixture.PublicDocumentPairs)*2)
	for _, document := range fixture.PublicDocumentPairs {
		docPaths = append(docPaths,
			filepath.Join(root, filepath.FromSlash(document.ES)),
			filepath.Join(root, filepath.FromSlash(document.EN)),
		)
	}
	if got := v21CountPhysicalLines(t, docPaths); got > fixture.SimplicityBudget.PublicDocumentLOC {
		t.Errorf("V21 public docs LOC=%d max=%d", got, fixture.SimplicityBudget.PublicDocumentLOC)
	}

	output, err := evidenceGit(root, "diff", "--numstat", v21ContractBaseGitCommitOID, "--",
		"internal/interfaces", "internal/bootstrap", "cmd/orquesta")
	if err != nil {
		t.Fatal(err)
	}
	added := 0
	for _, row := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if row == "" {
			continue
		}
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" {
			t.Fatalf("V21 cannot measure integration row %q", row)
		}
		if strings.HasSuffix(fields[2], "_test.go") {
			continue
		}
		value, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatal(err)
		}
		added += value
	}
	if added > fixture.SimplicityBudget.BindingIntegrationLOC {
		t.Errorf("V21 binding integration added LOC=%d max=%d", added, fixture.SimplicityBudget.BindingIntegrationLOC)
	}

	vendorPaths := v21VendorDeltaPaths(t, root)
	if len(vendorPaths) == 0 {
		t.Fatal("V21 catalog formatter dependency delta is absent")
	}
	if len(vendorPaths) > fixture.SimplicityBudget.VendorDeltaFileMax {
		t.Errorf("V21 vendor delta files=%d max=%d", len(vendorPaths), fixture.SimplicityBudget.VendorDeltaFileMax)
	}
	prefix := "vendor/" + fixture.SimplicityBudget.VendorModule + "/"
	vendorFiles := make([]string, 0, len(vendorPaths))
	for _, relative := range vendorPaths {
		if relative == "vendor/modules.txt" {
			continue
		}
		if !strings.HasPrefix(relative, prefix) {
			t.Errorf("V21 vendored delta %q is outside declared module %q", relative, fixture.SimplicityBudget.VendorModule)
			continue
		}
		vendorFiles = append(vendorFiles, filepath.Join(root, filepath.FromSlash(relative)))
	}
	if got := v21CountPhysicalLines(t, vendorFiles); got > fixture.SimplicityBudget.VendorDeltaLOCMax {
		t.Errorf("V21 vendor delta LOC=%d max=%d", got, fixture.SimplicityBudget.VendorDeltaLOCMax)
	}
}

func v21VendorDeltaPaths(t *testing.T, root string) []string {
	t.Helper()
	paths := make([]string, 0)
	for _, subject := range v21CurrentCandidateSubjects(t, root) {
		if strings.HasPrefix(subject, "vendor/") {
			paths = append(paths, subject)
		}
	}
	return paths
}
