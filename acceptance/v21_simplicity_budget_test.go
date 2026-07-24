package acceptance_test

import (
	"os"
	"path/filepath"
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
	owner := fixture.ProductDeltaSealedGitCommitOID

	if got := simplicityCountAddedGoLines(t, root, v21ContractBaseGitCommitOID, owner,
		[]string{"internal/i18n"}, nil); got > fixture.SimplicityBudget.I18NProductLOC {
		t.Errorf("V21 i18n product LOC=%d max=%d", got, fixture.SimplicityBudget.I18NProductLOC)
	}

	dataPaths := []string{fixture.ManifestPath}
	for _, relative := range fixture.CatalogPaths {
		dataPaths = append(dataPaths, relative)
	}
	if got := simplicityCountAddedLines(t, root, v21ContractBaseGitCommitOID, owner,
		dataPaths); got > fixture.SimplicityBudget.CatalogManifestDataLOC {
		t.Errorf("V21 catalog/manifest LOC=%d max=%d", got, fixture.SimplicityBudget.CatalogManifestDataLOC)
	}
	docPaths := make([]string, 0, len(fixture.PublicDocumentPairs)*2)
	for _, document := range fixture.PublicDocumentPairs {
		docPaths = append(docPaths, document.ES, document.EN)
	}
	if got := simplicityCountAddedLines(t, root, v21ContractBaseGitCommitOID, owner,
		docPaths); got > fixture.SimplicityBudget.PublicDocumentLOC {
		t.Errorf("V21 public docs LOC=%d max=%d", got, fixture.SimplicityBudget.PublicDocumentLOC)
	}

	added := simplicityCountAddedGoLines(t, root, v21ContractBaseGitCommitOID, owner,
		[]string{"internal/interfaces", "internal/bootstrap", "cmd/orquesta"}, nil)
	if added > fixture.SimplicityBudget.BindingIntegrationLOC {
		t.Errorf("V21 binding integration added LOC=%d max=%d", added, fixture.SimplicityBudget.BindingIntegrationLOC)
	}

	vendorPaths := v21VendorDeltaPaths(t, root, owner)
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
		vendorFiles = append(vendorFiles, relative)
	}
	if got := simplicityCountAddedLines(t, root, v21ContractBaseGitCommitOID, owner,
		vendorFiles); got > fixture.SimplicityBudget.VendorDeltaLOCMax {
		t.Errorf("V21 vendor delta LOC=%d max=%d", got, fixture.SimplicityBudget.VendorDeltaLOCMax)
	}
}

func v21VendorDeltaPaths(t *testing.T, root, owner string) []string {
	t.Helper()
	output, err := evidenceGit(root, "diff", "--name-only", "--no-renames",
		v21ContractBaseGitCommitOID, owner, "--", "vendor")
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0)
	for _, subject := range strings.Fields(string(output)) {
		if strings.HasPrefix(subject, "vendor/") {
			paths = append(paths, subject)
		}
	}
	return paths
}
