package orquestacionnucleoapp

import (
	"context"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ContextBundleRuntimeMaterializerV0 struct {
	Reader orquestacontext.ContextRefReaderV0
}

var _ RuntimeContextMaterializerPortV0 = ContextBundleRuntimeMaterializerV0{}

func (materializer ContextBundleRuntimeMaterializerV0) MaterializeRuntimeContextV0(
	ctx context.Context,
	request orquestaruntime.RuntimeLaunchRequestV0,
) (orquestacontext.ContextMaterializedBundleV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacontext.ContextMaterializedBundleV0{}, err
	}
	if request.ContextBundle == nil {
		return orquestacontext.ContextMaterializedBundleV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"context_bundle",
			"context_bundle requerido",
		)
	}
	result := orquestacontext.MaterializeContextBundleV0(*request.ContextBundle, materializer.Reader)
	if !result.Valid() {
		return result, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"context_materialized_bundle",
			"context_materialized_bundle_invalido",
		)
	}
	return result, nil
}
