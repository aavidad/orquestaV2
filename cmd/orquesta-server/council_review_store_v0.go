package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	council "orquesta/modulos/orquesta-council"
)

// Las revisiones de una entrega son DURABLES: si se perdieran al reiniciar, el
// cierre volveria a pedirlas y el trabajo se atascaria; y si se pudieran fabricar,
// no acreditarian nada.
const councilReviewsDirNameV0 = "council-reviews"

const councilDeliveryReviewsSchemaVersionV0 = "orquesta_council_delivery_reviews.v0"

var ErrCouncilReviewsCorruptV0 = errors.New("council_reviews_corrupt")

type councilReviewReceiptRecordV0 struct {
	ReviewerRef   string `json:"reviewer_ref"`
	FamilyRef     string `json:"family_ref,omitempty"`
	Verdict       string `json:"verdict"`
	EvidenceRef   string `json:"evidence_ref"`
	ReviewedAtUTC string `json:"reviewed_at_utc"`
}

type councilDeliveryReviewsV0 struct {
	SchemaVersion string                         `json:"schema_version"`
	RunRef        string                         `json:"run_ref"`
	AuthorRef     string                         `json:"author_ref"`
	AuthorFamily  string                         `json:"author_family,omitempty"`
	Reviews       []councilReviewReceiptRecordV0 `json:"reviews"`
}

type councilReviewStoreV0 struct {
	dir string
}

var _ orquestaappcodexstack.DeliveryReviewSourcePortV0 = councilReviewStoreV0{}

func newCouncilReviewStoreV0(stateDir string) (councilReviewStoreV0, error) {
	dir := filepath.Join(stateDir, councilReviewsDirNameV0)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return councilReviewStoreV0{}, fmt.Errorf("council reviews: %w", err)
	}
	return councilReviewStoreV0{dir: dir}, nil
}

func (store councilReviewStoreV0) pathV0(runRef string) (string, error) {
	ref := strings.TrimSpace(runRef)
	if ref == "" {
		return "", fmt.Errorf("run_ref_requerido")
	}
	if strings.ContainsAny(ref, "/\\") || ref == "." || ref == ".." {
		return "", fmt.Errorf("run_ref_invalido")
	}
	return filepath.Join(store.dir, ref+".json"), nil
}

func (store councilReviewStoreV0) loadV0(runRef string) (councilDeliveryReviewsV0, bool, error) {
	path, err := store.pathV0(runRef)
	if err != nil {
		return councilDeliveryReviewsV0{}, false, err
	}
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return councilDeliveryReviewsV0{}, false, nil
	}
	if err != nil {
		return councilDeliveryReviewsV0{}, false, fmt.Errorf("%w: %v", ErrCouncilReviewsCorruptV0, err)
	}
	var registro councilDeliveryReviewsV0
	if err := json.Unmarshal(bytes, &registro); err != nil {
		return councilDeliveryReviewsV0{}, false, fmt.Errorf("%w: %s", ErrCouncilReviewsCorruptV0, runRef)
	}
	if registro.SchemaVersion != councilDeliveryReviewsSchemaVersionV0 ||
		strings.TrimSpace(registro.RunRef) != strings.TrimSpace(runRef) {
		return councilDeliveryReviewsV0{}, false, fmt.Errorf("%w: %s", ErrCouncilReviewsCorruptV0, runRef)
	}
	return registro, true, nil
}

// RecordReviewV0 anota la revision de UN revisor sobre una entrega. Un revisor no
// puede revisar dos veces la misma entrega: repetirse no crea un segundo par de
// ojos.
func (store councilReviewStoreV0) RecordReviewV0(
	runRef string,
	authorRef string,
	authorFamily string,
	receipt councilReviewReceiptRecordV0,
) error {
	path, err := store.pathV0(runRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(receipt.ReviewerRef) == "" {
		return fmt.Errorf("council_review_reviewer_requerido")
	}
	if strings.TrimSpace(receipt.EvidenceRef) == "" {
		return fmt.Errorf("%w: %s", council.ErrRevisionSinEvidenciaV0, receipt.ReviewerRef)
	}
	if receipt.ReviewerRef == strings.TrimSpace(authorRef) {
		return fmt.Errorf("%w: %s", council.ErrRevisionDelAutorV0, receipt.ReviewerRef)
	}

	registro, existe, err := store.loadV0(runRef)
	if err != nil {
		return err
	}
	if !existe {
		registro = councilDeliveryReviewsV0{
			SchemaVersion: councilDeliveryReviewsSchemaVersionV0,
			RunRef:        strings.TrimSpace(runRef),
			AuthorRef:     strings.TrimSpace(authorRef),
			AuthorFamily:  strings.TrimSpace(authorFamily),
		}
	}
	for _, previa := range registro.Reviews {
		if previa.ReviewerRef == receipt.ReviewerRef {
			return fmt.Errorf("%w: %s", council.ErrRevisionDuplicadaV0, receipt.ReviewerRef)
		}
	}
	if receipt.ReviewedAtUTC == "" {
		receipt.ReviewedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	registro.Reviews = append(registro.Reviews, receipt)

	bytes, err := json.MarshalIndent(registro, "", "  ")
	if err != nil {
		return err
	}
	temporal, err := os.CreateTemp(store.dir, "reviews-*.tmp")
	if err != nil {
		return err
	}
	temporalPath := temporal.Name()
	defer os.Remove(temporalPath)
	if _, err := temporal.Write(bytes); err != nil {
		temporal.Close()
		return err
	}
	if err := temporal.Chmod(0o600); err != nil {
		temporal.Close()
		return err
	}
	if err := temporal.Close(); err != nil {
		return err
	}
	return os.Rename(temporalPath, path)
}

// ObserveDeliveryReviewsV0 es lo que consulta el validador de cierre. Las
// revisiones se OBSERVAN de aqui: no las declara quien cierra.
func (store councilReviewStoreV0) ObserveDeliveryReviewsV0(
	_ context.Context,
	runRef string,
) (string, string, []council.ReviewReceiptV0, error) {
	registro, existe, err := store.loadV0(runRef)
	if err != nil {
		return "", "", nil, err
	}
	if !existe {
		// Sin revisiones no hay cierre. No es un error: es que nadie ha mirado.
		return "", "", nil, nil
	}
	reviews := make([]council.ReviewReceiptV0, 0, len(registro.Reviews))
	for _, receipt := range registro.Reviews {
		reviews = append(reviews, council.ReviewReceiptV0{
			ReviewerRef: receipt.ReviewerRef,
			FamilyRef:   receipt.FamilyRef,
			Verdict:     council.VoteV0(receipt.Verdict),
			EvidenceRef: receipt.EvidenceRef,
		})
	}
	return registro.AuthorRef, registro.AuthorFamily, reviews, nil
}
