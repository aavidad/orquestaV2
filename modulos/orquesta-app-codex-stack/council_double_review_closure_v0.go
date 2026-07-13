package orquestaappcodexstack

import (
	"context"
	"fmt"

	council "orquesta/modulos/orquesta-council"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

// DeliveryReviewSourcePortV0 aporta las revisiones REALES de una entrega. Igual
// que con la cuota: no las declara quien cierra, se observan.
type DeliveryReviewSourcePortV0 interface {
	ObserveDeliveryReviewsV0(
		ctx context.Context,
		runRef string,
	) (authorRef string, authorFamily string, reviews []council.ReviewReceiptV0, err error)
}

// CouncilDoubleReviewConfigV0 exige dos revisiones independientes antes de dar por
// cerrada una entrega material. Es la ultima pieza del encargo del operador: el
// consejo tambien revisa a pares EN TIEMPO DE CREACION, no solo al arrancar.
type CouncilDoubleReviewConfigV0 struct {
	Required bool
	Reviews  DeliveryReviewSourcePortV0
}

const CouncilDoubleReviewMissingCodeV0 = "council_double_review_required"

type councilDoubleReviewClosureValidatorV0 struct {
	inner  orquestagoal.GoalWorkClosureValidatorPortV0
	config CouncilDoubleReviewConfigV0
}

var _ orquestagoal.GoalWorkClosureValidatorPortV0 = councilDoubleReviewClosureValidatorV0{}

// newCouncilDoubleReviewClosureValidatorV0 envuelve el validador real. Si la doble
// revision no se exige, devuelve el validador tal cual. Si SI se exige y falta la
// fuente de revisiones, NO se devuelve el validador desnudo: eso seria fail-open
// por mala configuracion, el mismo error que ya cometi en el gate.
func newCouncilDoubleReviewClosureValidatorV0(
	inner orquestagoal.GoalWorkClosureValidatorPortV0,
	config CouncilDoubleReviewConfigV0,
) orquestagoal.GoalWorkClosureValidatorPortV0 {
	if !config.Required {
		return inner
	}
	return councilDoubleReviewClosureValidatorV0{inner: inner, config: config}
}

func (validator councilDoubleReviewClosureValidatorV0) ValidateGoalWorkClosureV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	validation, err := validator.inner.ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil {
		return validation, err
	}
	// Si el nucleo ya rechaza el cierre, no hay nada que anadir.
	if !validation.Accepted {
		return validation, nil
	}

	if validator.config.Reviews == nil {
		return closureRechazadaPorRevisionV0(
			validation,
			"la doble revision esta exigida pero no hay fuente de revisiones cableada",
		), nil
	}

	authorRef, authorFamily, reviews, err := validator.config.Reviews.ObserveDeliveryReviewsV0(ctx, spec.RunRef)
	if err != nil {
		return closureRechazadaPorRevisionV0(
			validation,
			fmt.Sprintf("no se pudieron observar las revisiones de la entrega: %v", err),
		), nil
	}
	if err := council.ValidateIndependentReviewsV0(authorRef, authorFamily, reviews); err != nil {
		return closureRechazadaPorRevisionV0(validation, err.Error()), nil
	}
	return validation, nil
}

// closureRechazadaPorRevisionV0 convierte un cierre aceptado en uno que necesita
// rework: los tests verdes no sustituyen a dos pares de ojos.
func closureRechazadaPorRevisionV0(
	validation orquestagoal.GoalClosureValidationV0,
	motivo string,
) orquestagoal.GoalClosureValidationV0 {
	validation.Accepted = false
	validation.NeedsRework = true
	validation.Issues = append(validation.Issues, orquestagoal.GoalWorkIssueV0{
		Code:   CouncilDoubleReviewMissingCodeV0,
		Detail: motivo,
	})
	return validation
}
